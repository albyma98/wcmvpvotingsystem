package api

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
)

const merchNotificationMigration = `
CREATE TABLE IF NOT EXISTS merch_notifications (
    id INTEGER PRIMARY KEY,
    organization_id INTEGER NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    order_id TEXT NOT NULL,
    event_type TEXT NOT NULL,
    state TEXT NOT NULL DEFAULT 'queued' CHECK (state IN ('queued','sending','sent','failed','skipped','unknown')),
    attempts INTEGER NOT NULL DEFAULT 0,
    available_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    provider_id TEXT NOT NULL DEFAULT '',
    UNIQUE(organization_id, order_id, event_type),
    FOREIGN KEY (organization_id, order_id) REFERENCES merch_orders(organization_id, id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS merch_notifications_due ON merch_notifications(state, available_at, id);
CREATE TABLE IF NOT EXISTS merch_recovery_verifications (
    organization_id INTEGER NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    order_id TEXT NOT NULL,
    challenge_sid TEXT NOT NULL,
    expires_at INTEGER NOT NULL,
    verified_at INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY(organization_id, order_id),
    FOREIGN KEY (organization_id, order_id) REFERENCES merch_orders(organization_id, id) ON DELETE CASCADE
);
`

var merchNotificationMessages = map[string]string{
	"RESERVATION_CREATED":   "Prenotazione ricevuta, pagamento da effettuare alla societa.",
	"RESERVATION_CONFIRMED": "Prenotazione confermata.",
	"READY_FOR_PICKUP":      "Ordine pronto al ritiro.",
	"CANCELLED":             "Prenotazione annullata.",
	"EXPIRED":               "Prenotazione scaduta.",
	"PAYMENT_COMPLETED":     "Pagamento online confermato.",
	"RECOVERY_LINK":         "Consulta la tua prenotazione.",
}

func (rt *_router) ensureMerchNotifications(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, merchNotificationMigration)
	return err
}

func (rt *_router) queueMerchNotification(ctx context.Context, tx *sql.Tx, order merchOrder, eventType string) error {
	if merchNotificationMessages[eventType] == "" || order.FanID == 0 || order.Phone == "" {
		return nil
	}
	if order.Mode != "offline" && (eventType == "CANCELLED" || eventType == "EXPIRED") {
		return nil
	}
	now := time.Now().Unix()
	_, err := tx.ExecContext(ctx, `
		INSERT INTO merch_notifications (organization_id, order_id, event_type, available_at, updated_at)
		SELECT ?,?,?,?,?
		WHERE EXISTS (SELECT 1 FROM merch_settings s WHERE s.organization_id=?
			AND json_extract(s.config,'$.sms_notifications')=1)
		AND EXISTS (SELECT 1 FROM fan_profiles p WHERE p.id=? AND p.organization_id=?
			AND p.phone_e164=? AND TRIM(COALESCE(p.phone_verified_at,''))<>'')
		ON CONFLICT(organization_id, order_id, event_type) DO NOTHING`,
		order.OrganizationID, order.ID, eventType, now, now, order.OrganizationID,
		order.FanID, order.OrganizationID, order.Phone)
	return err
}

type merchNotification struct {
	id, attempts                                 int
	event, slug, phone, orderID, requestID, code string
	eligible                                     bool
}

func (rt *_router) processMerchNotifications(ctx context.Context) error {
	if rt.twilioMessaging == nil || !rt.twilioMessaging.enabled() {
		return nil
	}
	db := rt.db.SQLConn()
	now := time.Now().Unix()
	// If the worker stopped after calling the provider, delivery is uncertain.
	// A repeated send would risk a duplicate SMS and a duplicate charge.
	if _, err := db.ExecContext(ctx, `UPDATE merch_notifications SET state='unknown', updated_at=?
		WHERE state='sending' AND updated_at<?`, now, now-120); err != nil {
		return err
	}
	for count := 0; count < 5; count++ {
		n, err := claimMerchNotification(ctx, db)
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		state, sid, delay := "skipped", "", int64(0)
		if n.eligible {
			result, sendErr := rt.twilioMessaging.SendSMS(n.phone, rt.merchNotificationBody(n))
			switch {
			case sendErr == nil && result.SID != "":
				state, sid = "sent", result.SID
			case sendErr != nil:
				state, delay = merchNotificationFailure(sendErr, n.attempts)
			default:
				state = "unknown"
			}
		}
		now = time.Now().Unix()
		// Persist the result even when the worker's interval deadline expired during the send.
		finish, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		_, err = db.ExecContext(finish, `UPDATE merch_notifications SET state=?,provider_id=?,available_at=?,updated_at=?
			WHERE id=? AND state='sending'`, state, sid, now+delay, now, n.id)
		cancel()
		if err != nil {
			return err
		}
	}
	return nil
}

func claimMerchNotification(ctx context.Context, db *sql.DB) (merchNotification, error) {
	var n merchNotification
	now := time.Now().Unix()
	// SQLite serializes the UPDATE and its candidate selection across workers.
	err := db.QueryRowContext(ctx, `UPDATE merch_notifications SET state='sending',updated_at=?,attempts=attempts+1
		WHERE id=(SELECT id FROM merch_notifications WHERE state='queued' AND available_at<=? ORDER BY id LIMIT 1)
		RETURNING id, attempts, event_type`, now, now).Scan(&n.id, &n.attempts, &n.event)
	if err != nil {
		return n, err
	}
	err = db.QueryRowContext(ctx, `SELECT org.slug,o.phone,o.id,o.request_id,o.code,
		COALESCE(json_extract(s.config,'$.sms_notifications')=1,0) AND (
			(n.event_type='RECOVERY_LINK' AND EXISTS (SELECT 1 FROM merch_recovery_verifications v
				WHERE v.organization_id=o.organization_id AND v.order_id=o.id AND v.verified_at>0))
			OR EXISTS (SELECT 1 FROM fan_profiles p WHERE p.id=o.fan_id AND p.organization_id=o.organization_id
				AND p.phone_e164=o.phone AND TRIM(COALESCE(p.phone_verified_at,''))<>''))
		FROM merch_notifications n
		JOIN merch_orders o ON o.id=n.order_id AND o.organization_id=n.organization_id
		JOIN organizations org ON org.id=n.organization_id
		LEFT JOIN merch_settings s ON s.organization_id=n.organization_id WHERE n.id=?`, n.id).
		Scan(&n.slug, &n.phone, &n.orderID, &n.requestID, &n.code, &n.eligible)
	return n, err
}

func merchNotificationFailure(err error, attempts int) (state string, delay int64) {
	var providerErr twilioError
	if errors.As(err, &providerErr) {
		if providerErr.StatusCode == 429 && attempts < 5 {
			return "queued", int64(60 * attempts)
		}
		if providerErr.StatusCode >= 400 && providerErr.StatusCode < 500 {
			return "failed", 0
		}
	}
	return "unknown", 0
}

func (rt *_router) merchNotificationBody(n merchNotification) string {
	body := fmt.Sprintf("ArenaBoostX %s. %s Codice %s.", n.slug, merchNotificationMessages[n.event], n.code)
	if rt.merchStripe == nil || rt.merchStripe.base == nil {
		return body
	}
	target := *rt.merchStripe.base
	target.Path = strings.TrimRight(target.Path, "/") + "/" + n.slug + "/new/orders"
	target.RawPath = ""
	target.Fragment = url.Values{"order": {n.orderID}, "access": {merchGuestToken(n.orderID, n.requestID)}}.Encode()
	return body + " " + target.String()
}
