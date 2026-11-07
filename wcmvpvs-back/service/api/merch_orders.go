package api

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
)

const merchOnlineDuration = 45 * time.Minute

var merchRequestID = regexp.MustCompile(`^[A-Za-z0-9_-]{22,100}$`)

type merchLine struct {
	ProductID string `json:"product_id"`
	VariantID string `json:"variant_id"`
	Quantity  int    `json:"quantity"`
}

type merchCheckoutInput struct {
	Mode               string      `json:"mode"`
	Fulfillment        string      `json:"fulfillment"`
	Items              []merchLine `json:"items"`
	Name               string      `json:"name"`
	Email              string      `json:"email"`
	Phone              string      `json:"phone"`
	Address            string      `json:"address"`
	Notes              string      `json:"notes"`
	RequestID          string      `json:"request_id"`
	ExpectedTotalCents *int64      `json:"expected_total_cents"`
}

type merchOrderItem struct {
	ProductID   string `json:"product_id"`
	VariantID   string `json:"variant_id,omitempty"`
	Name        string `json:"name"`
	VariantName string `json:"variant_name,omitempty"`
	SKU         string `json:"sku,omitempty"`
	Quantity    int    `json:"quantity"`
	UnitCents   int64  `json:"unit_cents"`
	tracked     bool
}

type merchQuote struct {
	Items         []merchOrderItem `json:"items"`
	SubtotalCents int64            `json:"subtotal_cents"`
	ShippingCents int64            `json:"shipping_cents"`
	TotalCents    int64            `json:"total_cents"`
	Currency      string           `json:"currency"`
}

type merchInstructions struct {
	PickupPoint          string `json:"pickup_point"`
	DeliveryInstructions string `json:"delivery_instructions"`
	PaymentMethods       string `json:"payment_methods"`
	PaymentInstructions  string `json:"payment_instructions"`
	AvailabilityText     string `json:"availability_text"`
	ConfirmationText     string `json:"confirmation_text"`
	SellerInfo           string `json:"seller_info"`
	ContactEmail         string `json:"contact_email"`
	CancellationPolicy   string `json:"cancellation_policy"`
}

type merchOrder struct {
	merchQuote
	merchInstructions
	ID                   string         `json:"id"`
	OrganizationID       int            `json:"-"`
	RequestID            string         `json:"-"`
	RequestHash          string         `json:"-"`
	Code                 string         `json:"code"`
	FanID                int            `json:"-"`
	Name                 string         `json:"name"`
	Email                string         `json:"email"`
	Phone                string         `json:"phone"`
	Address              string         `json:"address"`
	Notes                string         `json:"notes"`
	Mode                 string         `json:"mode"`
	PaymentStatus        string         `json:"payment_status"`
	FulfillmentStatus    string         `json:"fulfillment_status"`
	Fulfillment          string         `json:"fulfillment"`
	StockStatus          string         `json:"-"`
	StripeAccount        string         `json:"-"`
	StripeSession        string         `json:"-"`
	StripePaymentIntent  string         `json:"-"`
	StripeURL            string         `json:"-"`
	StripeAwaiting       bool           `json:"-"`
	CreatedAt            int64          `json:"created_at"`
	ExpiresAt            int64          `json:"expires_at"`
	SMSRecoveryAvailable bool           `json:"sms_recovery_available,omitempty"`
	CheckoutAvailable    bool           `json:"checkout_available,omitempty"`
	Events               []merchAudit   `json:"events,omitempty"`
	Payments             []merchPayment `json:"payments,omitempty"`
}

type merchAudit struct {
	Type      string `json:"type"`
	Actor     string `json:"actor"`
	Notes     string `json:"notes"`
	CreatedAt int64  `json:"created_at"`
}

type merchPayment struct {
	Source      string `json:"source"`
	Status      string `json:"status"`
	AmountCents int64  `json:"amount_cents"`
	Method      string `json:"method"`
	Reference   string `json:"reference"`
	Actor       string `json:"actor"`
	CreatedAt   int64  `json:"created_at"`
}

const merchOrderColumns = `id,organization_id,request_id,request_hash,code,COALESCE(fan_id,0),name,email,phone,address,notes,mode,payment_status,fulfillment_status,fulfillment,stock_status,subtotal_cents,shipping_cents,total_cents,currency,instructions,stripe_account,stripe_session,stripe_payment_intent,stripe_url,stripe_awaiting,created_at,expires_at`

func scanMerchOrder(row merchScanner) (o merchOrder, err error) {
	var snapshot string
	err = row.Scan(&o.ID, &o.OrganizationID, &o.RequestID, &o.RequestHash, &o.Code, &o.FanID, &o.Name, &o.Email, &o.Phone, &o.Address, &o.Notes, &o.Mode, &o.PaymentStatus, &o.FulfillmentStatus, &o.Fulfillment, &o.StockStatus, &o.SubtotalCents, &o.ShippingCents, &o.TotalCents, &o.Currency, &snapshot, &o.StripeAccount, &o.StripeSession, &o.StripePaymentIntent, &o.StripeURL, &o.StripeAwaiting, &o.CreatedAt, &o.ExpiresAt)
	if err == nil {
		err = json.Unmarshal([]byte(snapshot), &o.merchInstructions)
	}
	return
}

func loadMerchOrderItems(ctx context.Context, q merchQuerier, o *merchOrder) error {
	rows, err := q.QueryContext(ctx, `SELECT product_id,variant_id,name,variant_name,sku,quantity,unit_cents,tracked
		FROM merch_order_items WHERE organization_id=? AND order_id=? ORDER BY line_no`, o.OrganizationID, o.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	o.Items = []merchOrderItem{}
	for rows.Next() {
		var i merchOrderItem
		if err := rows.Scan(&i.ProductID, &i.VariantID, &i.Name, &i.VariantName, &i.SKU, &i.Quantity, &i.UnitCents, &i.tracked); err != nil {
			return err
		}
		o.Items = append(o.Items, i)
	}
	return rows.Err()
}

func loadMerchOrder(ctx context.Context, q merchQuerier, org int, id string) (merchOrder, error) {
	o, err := scanMerchOrder(q.QueryRowContext(ctx, "SELECT "+merchOrderColumns+" FROM merch_orders WHERE organization_id=? AND id=?", org, id))
	if err == nil {
		err = loadMerchOrderItems(ctx, q, &o)
	}
	return o, err
}

func loadMerchOrderAudit(ctx context.Context, q merchQuerier, o *merchOrder) error {
	rows, err := q.QueryContext(ctx, "SELECT type,actor,notes,created_at FROM merch_order_events WHERE organization_id=? AND order_id=? ORDER BY id", o.OrganizationID, o.ID)
	if err != nil {
		return err
	}
	for rows.Next() {
		var e merchAudit
		if err := rows.Scan(&e.Type, &e.Actor, &e.Notes, &e.CreatedAt); err != nil {
			rows.Close()
			return err
		}
		o.Events = append(o.Events, e)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	rows, err = q.QueryContext(ctx, "SELECT source,status,amount_cents,method,reference,actor,created_at FROM merch_payment_records WHERE organization_id=? AND order_id=? ORDER BY id", o.OrganizationID, o.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var p merchPayment
		if err := rows.Scan(&p.Source, &p.Status, &p.AmountCents, &p.Method, &p.Reference, &p.Actor, &p.CreatedAt); err != nil {
			return err
		}
		o.Payments = append(o.Payments, p)
	}
	return rows.Err()
}

func merchDigest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func merchGuestToken(orderID, requestID string) string {
	return merchDigest("merch-order-access-v1\x00" + orderID + "\x00" + requestID)
}

func (o merchOrder) acceptsGuestToken(token string) bool {
	if len(token) != sha256.Size*2 || time.Now().Unix() > o.CreatedAt+int64((180*24*time.Hour)/time.Second) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(token), []byte(merchGuestToken(o.ID, o.RequestID))) == 1
}

func (rt *_router) merchFan(r *http.Request, org int) (database.FanProfile, error) {
	token := strings.TrimSpace(r.Header.Get("X-Fan-Session"))
	if token == "" {
		return database.FanProfile{}, nil
	}
	fan, err := rt.db.GetFanBySessionToken(token, "")
	if errors.Is(err, sql.ErrNoRows) {
		return database.FanProfile{}, nil
	}
	if err != nil {
		return database.FanProfile{}, err
	}
	if fan.Profile.OrganizationID != org {
		return database.FanProfile{}, merchError{http.StatusForbidden, "Profilo non valido per questa società"}
	}
	if fan.Profile.RegistrationPending {
		return database.FanProfile{}, nil
	}
	err = rt.db.SQLConn().QueryRowContext(r.Context(), "SELECT COALESCE(phone_verified_at, '') FROM fan_profiles WHERE id=? AND organization_id=? AND phone_e164=phone", fan.Profile.ID, org).Scan(&fan.Profile.PhoneVerifiedAt)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return database.FanProfile{}, err
	}
	return fan.Profile, nil
}

func validateMerchContact(in *merchCheckoutInput, settings merchSettings, fan database.FanProfile) error {
	for _, field := range []struct {
		text *string
		max  int
	}{
		{&in.Name, 160}, {&in.Email, 254}, {&in.Phone, 40}, {&in.Address, 600}, {&in.Notes, 500},
	} {
		*field.text = strings.TrimSpace(*field.text)
		if len(*field.text) > field.max {
			return merchInvalid("Dati di contatto troppo lunghi")
		}
	}
	if in.Email != "" && !validMerchEmail(in.Email) {
		return merchInvalid("Email non valida")
	}
	if in.Phone == "" && fan.PhoneVerifiedAt != "" {
		in.Phone = fan.Phone
	}
	if in.Phone != "" {
		phone, ok := normalizeE164(in.Phone)
		if !ok {
			return merchInvalid("Numero di telefono non valido. Indica il prefisso internazionale.")
		}
		in.Phone = phone
	}
	if in.Fulfillment == "pickup" {
		in.Address = ""
	}
	if in.Fulfillment == "shipping" && (in.Address == "" || in.Name == "" || (in.Mode == "online" && in.Email == "")) {
		return merchInvalid("Inserisci i dati per la spedizione")
	}
	if in.Mode == "offline" {
		if in.Name == "" {
			return merchInvalid("Inserisci nome e cognome")
		}
		switch settings.ContactRequirement {
		case "email":
			if in.Email == "" {
				return merchInvalid("Inserisci l'email")
			}
		case "phone":
			if in.Phone == "" {
				return merchInvalid("Inserisci il numero di telefono")
			}
		default:
			if in.Email == "" && in.Phone == "" {
				return merchInvalid("Inserisci email o telefono")
			}
		}
	}
	return nil
}

func quoteMerchOrder(ctx context.Context, q merchQuerier, org int, settings merchSettings, in merchCheckoutInput) (merchQuote, error) {
	quote := merchQuote{Currency: "eur", Items: []merchOrderItem{}}
	if in.Mode != "online" && in.Mode != "offline" {
		return quote, merchInvalid("Modalità di vendita non valida")
	}
	if !settings.Enabled || (in.Mode == "online" && !settings.OnlineEnabled) || (in.Mode == "offline" && !settings.OfflineEnabled) {
		return quote, merchConflict("Modalità di vendita non disponibile")
	}
	switch in.Fulfillment {
	case "pickup":
		if !settings.PickupEnabled {
			return quote, merchConflict("Ritiro non disponibile")
		}
	case "shipping":
		if !settings.ShippingEnabled {
			return quote, merchConflict("Spedizione non disponibile")
		}
		quote.ShippingCents = settings.ShippingCents
	default:
		return quote, merchInvalid("Seleziona ritiro o spedizione")
	}
	if len(in.Items) == 0 || len(in.Items) > 20 {
		return quote, merchInvalid("Numero di articoli non valido")
	}
	quantities := map[string]int{}
	lines := map[[2]string]bool{}
	for _, line := range in.Items {
		lineKey := [2]string{line.ProductID, line.VariantID}
		if line.Quantity < 1 || line.Quantity > 99 || lines[lineKey] {
			return quote, merchInvalid("Quantità o righe del carrello non valide")
		}
		lines[lineKey] = true
		quantities[line.ProductID] += line.Quantity
		p, err := scanMerchProduct(q.QueryRowContext(ctx, "SELECT "+merchProductColumns+" FROM merch_products WHERE organization_id=? AND id=? AND archived=0 AND published=1", org, line.ProductID))
		if errors.Is(err, sql.ErrNoRows) {
			return quote, merchConflict("Articolo non più disponibile")
		}
		if err != nil {
			return quote, err
		}
		if quantities[p.ID] > p.MaxQuantity {
			return quote, merchConflict("Quantità massima superata per " + p.Name)
		}
		on, off := merchModes(settings, p.SaleMode)
		if (in.Mode == "online" && !on) || (in.Mode == "offline" && !off) {
			return quote, merchConflict("Modalità di vendita non disponibile per " + p.Name)
		}
		if in.Fulfillment == "shipping" && !p.ShippingAllowed {
			return quote, merchConflict("Spedizione non disponibile per " + p.Name)
		}
		item := merchOrderItem{ProductID: p.ID, VariantID: line.VariantID, Name: p.Name, SKU: p.SKU, Quantity: line.Quantity, UnitCents: p.PriceCents, tracked: !p.OnDemand}
		if p.PromoPriceCents != nil {
			item.UnitCents = *p.PromoPriceCents
		}
		stock := p.Stock
		if line.VariantID == "" {
			var hasVariants bool
			err := q.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM merch_product_variants WHERE organization_id=? AND product_id=? AND active=1)", org, p.ID).Scan(&hasVariants)
			if err != nil {
				return quote, err
			}
			if hasVariants {
				return quote, merchInvalid("Seleziona la variante di " + p.Name)
			}
		} else {
			var v merchVariant
			err := q.QueryRowContext(ctx, `SELECT size,color,sku,price_cents,stock,on_demand FROM merch_product_variants
				WHERE organization_id=? AND product_id=? AND id=? AND active=1`, org, p.ID, line.VariantID).Scan(&v.Size, &v.Color, &v.SKU, &v.PriceCents, &v.Stock, &v.OnDemand)
			if errors.Is(err, sql.ErrNoRows) {
				return quote, merchConflict("Variante non disponibile")
			}
			if err != nil {
				return quote, err
			}
			item.VariantName = strings.Trim(strings.Join([]string{v.Size, v.Color}, " / "), " /")
			if v.SKU != "" {
				item.SKU = v.SKU
			}
			if v.PriceCents != nil {
				item.UnitCents = *v.PriceCents
			}
			stock, item.tracked = v.Stock, !v.OnDemand
		}
		if item.tracked {
			var held int
			if err := q.QueryRowContext(ctx, merchHoldSum, org, p.ID, line.VariantID).Scan(&held); err != nil {
				return quote, err
			}
			if stock-held < item.Quantity {
				return quote, merchConflict("Disponibilità insufficiente per " + p.Name)
			}
		}
		quote.Items = append(quote.Items, item)
		quote.SubtotalCents += int64(item.Quantity) * item.UnitCents
	}
	quote.TotalCents = quote.SubtotalCents + quote.ShippingCents
	if in.Mode == "online" && (quote.TotalCents < 50 || quote.TotalCents > 99999999) {
		return quote, merchConflict("Totale fuori dai limiti del pagamento online")
	}
	return quote, nil
}

func (rt *_router) createMerchOrder(ctx context.Context, org int, account string, in merchCheckoutInput, fan database.FanProfile, ip string) (merchOrder, error) {
	var order merchOrder
	if !merchRequestID.MatchString(in.RequestID) || in.ExpectedTotalCents == nil {
		return order, merchInvalid("Riepilogo o identificativo richiesta mancante")
	}
	raw, err := json.Marshal(in)
	if err != nil {
		return order, err
	}
	requestHash := merchDigest(fmt.Sprintf("%d\x00%s", fan.ID, raw))
	var rejection error
	err = rt.merchWrite(ctx, func(tx *sql.Tx) error {
		existing, err := scanMerchOrder(tx.QueryRowContext(ctx, "SELECT "+merchOrderColumns+" FROM merch_orders WHERE organization_id=? AND request_id=?", org, in.RequestID))
		if err == nil {
			if existing.RequestHash != requestHash {
				return merchConflict("Identificativo già utilizzato per un'altra richiesta")
			}
			order = existing
			return loadMerchOrderItems(ctx, tx, &order)
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		ok, err := merchRate(ctx, tx, org, "order", ip, time.Hour, 45)
		if err != nil {
			return err
		}
		if !ok {
			rejection = merchError{http.StatusTooManyRequests, "Troppe richieste. Riprova tra poco."}
			return nil
		}
		settings, err := loadMerchSettings(ctx, tx, org)
		if err != nil {
			return err
		}
		settings.OnlineEnabled = settings.OnlineEnabled && account != ""
		quote, err := quoteMerchOrder(ctx, tx, org, settings, in)
		if err != nil {
			if _, ok := err.(merchError); ok {
				rejection = err
				return nil
			}
			return err
		}
		if quote.TotalCents != *in.ExpectedTotalCents {
			rejection = merchConflict("Il totale è cambiato. Controlla il nuovo riepilogo prima di confermare.")
			return nil
		}
		if err := validateMerchContact(&in, settings, fan); err != nil {
			rejection = err
			return nil
		}
		if in.Mode == "offline" {
			// Bound abusive reservations by verified fan or stated contact as well as IP.
			identity := strings.ToLower(in.Email) + "|" + in.Phone
			if fan.ID != 0 {
				identity = fmt.Sprintf("fan:%d", fan.ID)
			}
			allowed, err := merchRate(ctx, tx, org, "reservation_contact", identity, time.Hour, 5)
			if err != nil {
				return err
			}
			if !allowed {
				rejection = merchError{http.StatusTooManyRequests, "Troppe prenotazioni per questo recapito. Riprova tra poco."}
				return nil
			}
		}
		id, err := merchID()
		if err != nil {
			return err
		}
		now := time.Now()
		order = merchOrder{
			merchQuote: quote, ID: id, OrganizationID: org, RequestID: in.RequestID, RequestHash: requestHash,
			Code: strings.ToUpper(strings.ReplaceAll(id, "-", "")[:12]), FanID: fan.ID, Name: in.Name, Email: in.Email, Phone: in.Phone, Address: in.Address, Notes: in.Notes,
			Mode: in.Mode, FulfillmentStatus: "PENDING", Fulfillment: in.Fulfillment, StockStatus: "HELD", CreatedAt: now.Unix(),
			merchInstructions: merchInstructions{PickupPoint: settings.PickupPoint, DeliveryInstructions: settings.DeliveryInstructions, PaymentMethods: settings.PaymentMethods, PaymentInstructions: settings.PaymentInstructions, AvailabilityText: settings.AvailabilityText, ConfirmationText: settings.ConfirmationText, SellerInfo: settings.SellerInfo, ContactEmail: settings.ContactEmail, CancellationPolicy: settings.CancellationPolicy},
		}
		var event string
		if order.Mode == "online" {
			order.StripeAccount, order.PaymentStatus, order.ExpiresAt = account, "PENDING_ONLINE", now.Add(merchOnlineDuration).Unix()
			event = "CHECKOUT_STARTED"
		} else {
			order.PaymentStatus, order.ExpiresAt = "UNPAID_OFFLINE", now.Add(time.Duration(settings.ReservationHours)*time.Hour).Unix()
			if !settings.ReserveOffline {
				order.StockStatus = "DEFERRED"
			}
			event = "RESERVATION_CREATED"
		}
		snapshot, err := json.Marshal(order.merchInstructions)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO merch_orders (id,organization_id,request_id,request_hash,code,fan_id,name,email,phone,address,notes,mode,payment_status,fulfillment_status,fulfillment,stock_status,subtotal_cents,shipping_cents,total_cents,currency,instructions,stripe_account,created_at,expires_at)
			VALUES (?,?,?,?,?,NULLIF(?,0),?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, order.ID, org, in.RequestID, requestHash, order.Code, fan.ID, order.Name, order.Email, order.Phone, order.Address, order.Notes, order.Mode, order.PaymentStatus, order.FulfillmentStatus, order.Fulfillment, order.StockStatus, order.SubtotalCents, order.ShippingCents, order.TotalCents, order.Currency, string(snapshot), order.StripeAccount, order.CreatedAt, order.ExpiresAt)
		if err != nil {
			return err
		}
		for n, i := range order.Items {
			_, err = tx.ExecContext(ctx, `INSERT INTO merch_order_items (organization_id,order_id,line_no,product_id,variant_id,name,variant_name,sku,quantity,unit_cents,tracked) VALUES (?,?,?,?,?,?,?,?,?,?,?)`, org, order.ID, n, i.ProductID, i.VariantID, i.Name, i.VariantName, i.SKU, i.Quantity, i.UnitCents, i.tracked)
			if err != nil {
				return err
			}
		}
		if order.StockStatus == "HELD" {
			if err := addMerchInventoryMovements(ctx, tx, order, "HELD"); err != nil {
				return err
			}
		}
		if err := rt.addMerchOrderEvent(ctx, tx, order, event, "fan", ""); err != nil {
			return err
		}
		if err := addMerchAnalytics(ctx, tx, org, "order:"+order.ID+":checkout", "merch_checkout_started", order.Items[0].ProductID, ""); err != nil {
			return err
		}
		if order.Mode == "offline" {
			return addMerchAnalytics(ctx, tx, org, "order:"+order.ID+":reservation", "merch_reservation_created", order.Items[0].ProductID, "")
		}
		return nil
	})
	if err == nil {
		err = rejection
	}
	return order, err
}

func addMerchInventoryMovements(ctx context.Context, tx *sql.Tx, order merchOrder, movement string) error {
	for _, i := range order.Items {
		if !i.tracked {
			continue
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO merch_inventory_movements (organization_id,order_id,product_id,variant_id,quantity,type,created_at) VALUES (?,?,?,?,?,?,?)`, order.OrganizationID, order.ID, i.ProductID, i.VariantID, i.Quantity, movement, time.Now().Unix()); err != nil {
			return err
		}
	}
	return nil
}

func (rt *_router) addMerchOrderEvent(ctx context.Context, tx *sql.Tx, order merchOrder, event, actor, notes string) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO merch_order_events (organization_id,order_id,type,actor,notes,created_at) VALUES (?,?,?,?,?,?)`, order.OrganizationID, order.ID, event, actor, notes, time.Now().Unix())
	if err != nil {
		return err
	}
	return rt.queueMerchNotification(ctx, tx, order, event)
}

func addMerchPayment(ctx context.Context, tx *sql.Tx, order merchOrder, source, status, method, reference, actor string, cents int64) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO merch_payment_records (organization_id,order_id,source,status,amount_cents,method,reference,actor,created_at) VALUES (?,?,?,?,?,?,?,?,?)`, order.OrganizationID, order.ID, source, status, cents, method, reference, actor, time.Now().Unix())
	return err
}

func addMerchAnalytics(ctx context.Context, tx *sql.Tx, org int, event, eventType, product, variant string) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO merch_analytics_events (organization_id,event_id,type,product_id,variant_id,created_at) VALUES (?,?,?,?,?,?) ON CONFLICT(organization_id,event_id) DO NOTHING`, org, event, eventType, product, variant, time.Now().Unix())
	return err
}
