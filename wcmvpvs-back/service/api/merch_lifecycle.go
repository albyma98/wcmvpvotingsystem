package api

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type merchAdminAction struct {
	Action      string `json:"action"`
	AmountCents int64  `json:"amount_cents"`
	Method      string `json:"method"`
	Reference   string `json:"reference"`
	Notes       string `json:"notes"`
	Confirmed   bool   `json:"confirmed"`
}

func consumeMerchStock(ctx context.Context, tx *sql.Tx, order *merchOrder) error {
	if order.StockStatus == "CONSUMED" {
		return nil
	}
	if order.StockStatus != "HELD" && order.StockStatus != "DEFERRED" {
		return merchConflict("Lo stock riservato è stato rilasciato")
	}
	// Validate every line first. A paid event can still record payment and enter
	// review without committing a partial inventory update.
	for _, item := range order.Items {
		if !item.tracked {
			continue
		}
		var stock, held int
		if item.VariantID == "" {
			err := tx.QueryRowContext(ctx, "SELECT stock FROM merch_products WHERE organization_id=? AND id=?", order.OrganizationID, item.ProductID).Scan(&stock)
			if err != nil {
				return err
			}
		} else {
			err := tx.QueryRowContext(ctx, "SELECT stock FROM merch_product_variants WHERE organization_id=? AND product_id=? AND id=?", order.OrganizationID, item.ProductID, item.VariantID).Scan(&stock)
			if err != nil {
				return err
			}
		}
		if err := tx.QueryRowContext(ctx, merchHoldSum, order.OrganizationID, item.ProductID, item.VariantID).Scan(&held); err != nil {
			return err
		}
		if order.StockStatus == "HELD" {
			held -= item.Quantity
		}
		if stock-held < item.Quantity {
			return merchConflict("Disponibilità insufficiente per " + item.Name)
		}
	}
	for _, item := range order.Items {
		if !item.tracked {
			continue
		}
		if item.VariantID == "" {
			if _, err := tx.ExecContext(ctx, "UPDATE merch_products SET stock=stock-?,version=version+1 WHERE organization_id=? AND id=?", item.Quantity, order.OrganizationID, item.ProductID); err != nil {
				return err
			}
		} else {
			if _, err := tx.ExecContext(ctx, "UPDATE merch_product_variants SET stock=stock-? WHERE organization_id=? AND product_id=? AND id=?", item.Quantity, order.OrganizationID, item.ProductID, item.VariantID); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, "UPDATE merch_products SET version=version+1 WHERE organization_id=? AND id=?", order.OrganizationID, item.ProductID); err != nil {
				return err
			}
		}
	}
	order.StockStatus = "CONSUMED"
	return addMerchInventoryMovements(ctx, tx, *order, "CONSUMED")
}

func releaseMerchStock(ctx context.Context, tx *sql.Tx, order *merchOrder) error {
	switch order.StockStatus {
	case "HELD":
		if err := addMerchInventoryMovements(ctx, tx, *order, "RELEASED"); err != nil {
			return err
		}
	case "DEFERRED":
	default:
		return nil
	}
	order.StockStatus = "RELEASED"
	return nil
}

func saveMerchOrderState(ctx context.Context, tx *sql.Tx, order merchOrder) error {
	_, err := tx.ExecContext(ctx, `UPDATE merch_orders SET payment_status=?,fulfillment_status=?,stock_status=?,stripe_session=?,stripe_payment_intent=?,stripe_awaiting=?,expires_at=?,name=?,email=?,phone=?
		WHERE organization_id=? AND id=?`, order.PaymentStatus, order.FulfillmentStatus, order.StockStatus, order.StripeSession, order.StripePaymentIntent, order.StripeAwaiting, order.ExpiresAt, order.Name, order.Email, order.Phone, order.OrganizationID, order.ID)
	return err
}

func (rt *_router) applyMerchAdminAction(ctx context.Context, org int, id, actor string, a merchAdminAction) (merchOrder, error) {
	var order merchOrder
	a.Method, a.Reference, a.Notes = strings.TrimSpace(a.Method), strings.TrimSpace(a.Reference), strings.TrimSpace(a.Notes)
	if len(a.Method) > 100 || len(a.Reference) > 200 || len(a.Notes) > 500 {
		return order, merchInvalid("Dettagli operazione troppo lunghi")
	}
	switch a.Action {
	case "record_payment", "complete", "cancel", "refund":
		if !a.Confirmed {
			return order, merchInvalid("Conferma l'operazione per proseguire")
		}
	}
	err := rt.merchWrite(ctx, func(tx *sql.Tx) error {
		var err error
		order, err = loadMerchOrder(ctx, tx, org, id)
		if err != nil {
			return err
		}
		offlineUnpaid := order.Mode == "offline" && order.PaymentStatus == "UNPAID_OFFLINE" && order.ExpiresAt > time.Now().Unix()
		paid := order.PaymentStatus == "PAID" || order.PaymentStatus == "PAID_OFFLINE"
		closed := order.FulfillmentStatus == "CANCELLED" || order.FulfillmentStatus == "EXPIRED" || order.FulfillmentStatus == "COMPLETED"
		if closed && a.Action != "refund" {
			return merchConflict("Ordine già chiuso")
		}
		invalid := merchConflict("Operazione incompatibile con lo stato dell'ordine")
		var event string
		switch a.Action {
		case "confirm":
			if !offlineUnpaid || order.FulfillmentStatus != "PENDING" {
				return invalid
			}
			order.FulfillmentStatus, event = "CONFIRMED", "RESERVATION_CONFIRMED"
		case "prepare":
			if (!offlineUnpaid && !paid) || order.FulfillmentStatus != "CONFIRMED" {
				return invalid
			}
			order.FulfillmentStatus, event = "PREPARING", "PREPARING"
		case "ready":
			if (!offlineUnpaid && !paid) || order.Fulfillment != "pickup" || order.FulfillmentStatus != "PREPARING" {
				return invalid
			}
			order.FulfillmentStatus, event = "READY_FOR_PICKUP", "READY_FOR_PICKUP"
		case "record_payment":
			if !offlineUnpaid || a.AmountCents != order.TotalCents || a.Method == "" {
				return merchInvalid("Inserisci il metodo e l'importo esatto da registrare per questa prenotazione")
			}
			if err := consumeMerchStock(ctx, tx, &order); err != nil {
				return err
			}
			if err := addMerchPayment(ctx, tx, order, "offline", "PAID_OFFLINE", a.Method, a.Reference, actor, a.AmountCents); err != nil {
				return err
			}
			order.PaymentStatus, event = "PAID_OFFLINE", "PAYMENT_OFFLINE_RECORDED"
			if order.FulfillmentStatus == "PENDING" {
				order.FulfillmentStatus = "CONFIRMED"
			}
		case "complete":
			if !paid || order.StockStatus != "CONSUMED" || (order.Fulfillment == "pickup" && order.FulfillmentStatus != "READY_FOR_PICKUP") || (order.Fulfillment == "shipping" && order.FulfillmentStatus != "PREPARING") {
				return invalid
			}
			order.FulfillmentStatus, event = "COMPLETED", "COMPLETED"
		case "cancel":
			if !offlineUnpaid {
				return invalid
			}
			if err := releaseMerchStock(ctx, tx, &order); err != nil {
				return err
			}
			order.FulfillmentStatus, event = "CANCELLED", "CANCELLED"
		case "refund":
			if order.Mode != "online" || order.PaymentStatus != "PAID" || order.StripePaymentIntent == "" {
				return invalid
			}
			return rt.addMerchOrderEvent(ctx, tx, order, "REFUND_REQUESTED", actor, a.Notes)
		default:
			return merchInvalid("Operazione non valida")
		}
		if err := saveMerchOrderState(ctx, tx, order); err != nil {
			return err
		}
		return rt.addMerchOrderEvent(ctx, tx, order, event, actor, a.Notes)
	})
	if err != nil {
		return order, err
	}
	if a.Action == "refund" {
		if err := rt.refundMerchStripe(ctx, order); err != nil {
			rt.baseLogger.WithError(err).WithField("order_id", order.ID).Warn("Stripe merchandise refund failed")
			return order, merchError{http.StatusBadGateway, "Impossibile confermare la richiesta di rimborso Stripe. Riprova."}
		}
	}
	order, err = loadMerchOrder(ctx, rt.db.SQLConn(), org, id)
	if err == nil {
		err = loadMerchOrderAudit(ctx, rt.db.SQLConn(), &order)
	}
	return order, err
}

func (rt *_router) attachMerchStripeSession(ctx context.Context, orderID, accountID, sessionID, checkoutURL string) error {
	return rt.merchWrite(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, `UPDATE merch_orders SET stripe_session=?,stripe_url=?
			WHERE id=? AND mode='online' AND stripe_account=? AND (stripe_session='' OR stripe_session=?)
			AND payment_status='PENDING_ONLINE' AND stripe_awaiting=0 AND expires_at>?`, sessionID, checkoutURL, orderID, accountID, sessionID, time.Now().Unix())
		if err != nil {
			return err
		}
		n, err := res.RowsAffected()
		if err != nil || n == 1 {
			return err
		}
		var bound string
		return tx.QueryRowContext(ctx, "SELECT id FROM merch_orders WHERE id=? AND stripe_account=? AND stripe_session=?", orderID, accountID, sessionID).Scan(&bound)
	})
}

func (rt *_router) applyMerchStripeEvent(ctx context.Context, event merchStripeEvent) error {
	if event.EventID == "" || event.AccountID == "" {
		return errors.New("incomplete Stripe event")
	}
	return rt.merchWrite(ctx, func(tx *sql.Tx) error {
		query := "SELECT " + merchOrderColumns + " FROM merch_orders WHERE stripe_account=? AND mode='online' AND "
		field, key := "id", event.OrderID
		if key == "" {
			field, key = "stripe_payment_intent", event.PaymentIntent
		}
		if key == "" {
			return sql.ErrNoRows
		}
		order, err := scanMerchOrder(tx.QueryRowContext(ctx, query+field+"=?", event.AccountID, key))
		if err != nil {
			return err
		}
		if !strings.EqualFold(event.Currency, order.Currency) {
			return errors.New("merchandise Stripe currency mismatch")
		}
		if event.Status != "REFUNDED" {
			if event.TotalCents != order.TotalCents || event.SessionID == "" || (order.StripeSession != "" && event.SessionID != order.StripeSession) {
				return errors.New("merchandise Stripe total or session mismatch")
			}
		}
		if order.StripePaymentIntent != "" && event.PaymentIntent != "" && order.StripePaymentIntent != event.PaymentIntent {
			return errors.New("merchandise Stripe payment intent mismatch")
		}
		result, err := tx.ExecContext(ctx, `INSERT INTO merch_stripe_events (event_id,organization_id,order_id,created_at) VALUES (?,?,?,?) ON CONFLICT(event_id) DO NOTHING`, event.EventID, order.OrganizationID, order.ID, time.Now().Unix())
		if err != nil {
			return err
		}
		n, err := result.RowsAffected()
		if err != nil || n == 0 {
			return err
		}
		if err := loadMerchOrderItems(ctx, tx, &order); err != nil {
			return err
		}
		if order.StripeSession == "" {
			order.StripeSession = event.SessionID
		}
		if order.StripePaymentIntent == "" {
			order.StripePaymentIntent = event.PaymentIntent
		}
		var audit string
		switch event.Status {
		case "PENDING_ONLINE":
			if order.PaymentStatus == "PENDING_ONLINE" && !order.StripeAwaiting {
				order.StripeAwaiting = true
				order.ExpiresAt = time.Now().Add(24 * time.Hour).Unix()
				audit = "PAYMENT_PROCESSING"
			}
		case "FAILED", "EXPIRED":
			if order.PaymentStatus == "PENDING_ONLINE" {
				if err := releaseMerchStock(ctx, tx, &order); err != nil {
					return err
				}
				order.PaymentStatus = event.Status
				order.FulfillmentStatus = "CANCELLED"
				if event.Status == "EXPIRED" {
					order.FulfillmentStatus = "EXPIRED"
				}
				audit = event.Status
			}
		case "PAID":
			if order.PaymentStatus == "PAID" || order.PaymentStatus == "REFUNDED" {
				return nil
			}
			if event.PaymentIntent == "" {
				return errors.New("paid Stripe session lacks a payment intent")
			}
			order.PaymentStatus, order.FulfillmentStatus, audit = "PAID", "CONFIRMED", "PAYMENT_COMPLETED"
			// Released holds and uncertain late payments need staff action.
			paidAt := event.OccurredAt
			if paidAt <= 0 {
				paidAt = time.Now().Unix()
			}
			if order.StockStatus != "HELD" || order.ExpiresAt < paidAt {
				order.FulfillmentStatus = "REVIEW_REQUIRED"
				if err := releaseMerchStock(ctx, tx, &order); err != nil {
					return err
				}
			} else if err := consumeMerchStock(ctx, tx, &order); err != nil {
				var problem merchError
				if !errors.As(err, &problem) {
					return err
				}
				order.FulfillmentStatus = "REVIEW_REQUIRED"
				if err := releaseMerchStock(ctx, tx, &order); err != nil {
					return err
				}
			}
			if order.Name == "" && len(event.CustomerName) <= 160 {
				order.Name = strings.TrimSpace(event.CustomerName)
			}
			if order.Email == "" && validMerchEmail(event.CustomerEmail) {
				order.Email = event.CustomerEmail
			}
			if order.Phone == "" {
				if phone, ok := normalizeE164(event.CustomerPhone); ok {
					order.Phone = phone
				}
			}
			if err := addMerchPayment(ctx, tx, order, "stripe", "PAID", "Stripe", event.PaymentIntent, "Stripe", order.TotalCents); err != nil {
				return err
			}
			if err := addMerchAnalytics(ctx, tx, order.OrganizationID, "order:"+order.ID+":payment", "merch_payment_completed", order.Items[0].ProductID, ""); err != nil {
				return err
			}
		case "REFUNDED":
			if event.PaymentIntent == "" || event.RefundID == "" || event.RefundedCents <= 0 || event.RefundedCents > order.TotalCents {
				return errors.New("invalid merchandise Stripe refund")
			}
			if order.PaymentStatus != "PAID" && order.PaymentStatus != "REFUNDED" {
				return errors.New("merchandise Stripe refund arrived before payment")
			}
			var refunded int64
			var duplicate bool
			err := tx.QueryRowContext(ctx, `SELECT COALESCE(SUM(amount_cents),0),COALESCE(MAX(reference=?),0) FROM merch_payment_records
				WHERE organization_id=? AND order_id=? AND source='stripe' AND status='REFUNDED'`, event.RefundID, order.OrganizationID, order.ID).Scan(&refunded, &duplicate)
			if err != nil || duplicate {
				return err
			}
			refunded += event.RefundedCents
			if refunded > order.TotalCents {
				return errors.New("merchandise refund exceeds recorded payment")
			}
			if err := addMerchPayment(ctx, tx, order, "stripe", "REFUNDED", "Stripe", event.RefundID, "Stripe", event.RefundedCents); err != nil {
				return err
			}
			audit = "PARTIALLY_REFUNDED"
			if refunded == order.TotalCents {
				order.PaymentStatus, audit = "REFUNDED", "REFUNDED"
				if order.FulfillmentStatus != "COMPLETED" {
					order.FulfillmentStatus = "CANCELLED"
				}
			}
		default:
			return errors.New("unknown merchandise Stripe event")
		}
		if err := saveMerchOrderState(ctx, tx, order); err != nil {
			return err
		}
		if audit == "" {
			return nil
		}
		if err := rt.addMerchOrderEvent(ctx, tx, order, audit, "Stripe", ""); err != nil {
			return err
		}
		if order.FulfillmentStatus == "REVIEW_REQUIRED" && audit == "PAYMENT_COMPLETED" {
			return rt.addMerchOrderEvent(ctx, tx, order, "REVIEW_REQUIRED", "system", "Pagamento confermato dopo la scadenza o senza stock riservato")
		}
		return nil
	})
}

func (rt *_router) expireMerchOrder(ctx context.Context, org int, id string) error {
	return rt.merchWrite(ctx, func(tx *sql.Tx) error {
		order, err := loadMerchOrder(ctx, tx, org, id)
		if err != nil {
			return err
		}
		if (order.PaymentStatus != "PENDING_ONLINE" && order.PaymentStatus != "UNPAID_OFFLINE") || order.ExpiresAt > time.Now().Unix() ||
			order.FulfillmentStatus == "CANCELLED" || order.FulfillmentStatus == "EXPIRED" {
			return nil
		}
		if err := releaseMerchStock(ctx, tx, &order); err != nil {
			return err
		}
		order.PaymentStatus, order.FulfillmentStatus = "EXPIRED", "EXPIRED"
		if err := saveMerchOrderState(ctx, tx, order); err != nil {
			return err
		}
		return rt.addMerchOrderEvent(ctx, tx, order, "EXPIRED", "system", "")
	})
}

type merchWorker struct {
	cancel context.CancelFunc
	done   chan struct{}
}

func (rt *_router) startMerchWorker() *merchWorker {
	ctx, cancel := context.WithCancel(context.Background())
	worker := &merchWorker{cancel: cancel, done: make(chan struct{})}
	go func() {
		defer close(worker.done)
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			run, stop := context.WithTimeout(ctx, 50*time.Second)
			err := rt.reconcileMerchOrders(run)
			if err == nil {
				err = rt.processMerchNotifications(run)
			}
			stop()
			if err != nil && ctx.Err() == nil {
				rt.baseLogger.WithError(err).Warn("merchandising reconciliation failed")
			}
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
	return worker
}

func (rt *_router) reconcileMerchOrders(ctx context.Context) error {
	conn := rt.db.SQLConn()
	now := time.Now().Unix()
	rows, err := conn.QueryContext(ctx, `SELECT `+merchOrderColumns+` FROM merch_orders
		WHERE payment_status IN ('PENDING_ONLINE','UNPAID_OFFLINE') AND fulfillment_status NOT IN ('CANCELLED','EXPIRED')
		AND expires_at<=? AND stripe_checked_at<=? ORDER BY expires_at LIMIT 100`, now, now-60)
	if err != nil {
		return err
	}
	var orders []merchOrder
	for rows.Next() {
		order, err := scanMerchOrder(rows)
		if err != nil {
			rows.Close()
			return err
		}
		orders = append(orders, order)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for _, order := range orders {
		if order.Mode == "online" && order.StripeSession != "" {
			if _, err := conn.ExecContext(ctx, "UPDATE merch_orders SET stripe_checked_at=? WHERE organization_id=? AND id=?", now, order.OrganizationID, order.ID); err != nil {
				return err
			}
			request, stop := context.WithTimeout(ctx, 8*time.Second)
			err := rt.reconcileMerchStripeOrder(request, order)
			stop()
			if err != nil {
				rt.baseLogger.WithError(err).WithField("order_id", order.ID).Warn("Stripe merchandise reconciliation failed")
			}
			// During a Stripe outage, allow a short grace period for a verified
			// response. A later verified payment is recorded as REVIEW_REQUIRED.
			if !order.StripeAwaiting && now-order.ExpiresAt < 15*60 {
				continue
			}
		}
		if err := rt.expireMerchOrder(ctx, order.OrganizationID, order.ID); err != nil {
			return fmt.Errorf("merchandise expiry: %w", err)
		}
	}
	if _, err = conn.ExecContext(ctx, "DELETE FROM merch_rate_limits WHERE time_bucket<?", now-86400); err != nil {
		return err
	}
	return cleanUnattachedMerchMedia(ctx, conn, now)
}
