package api

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const matchdayQRParameter = "abx_qr"

const matchdayQRSchema = `CREATE TABLE IF NOT EXISTS matchday_qr_arrivals (
	token_key TEXT PRIMARY KEY,
	organization_id INTEGER NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
	event_id INTEGER NOT NULL REFERENCES events(id) ON DELETE CASCADE,
	redirect_id INTEGER NOT NULL,
	channel TEXT NOT NULL,
	opened_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
	landed_at TEXT,
	landing_key TEXT
);
CREATE INDEX IF NOT EXISTS idx_matchday_qr_event ON matchday_qr_arrivals(event_id, organization_id);`

var matchdayEntryChannels = map[string]string{
	"led": "LED", "match_programme": "Giornalino", "arena": "Arena", "remote": "Remoto",
	"direct": "Diretto / senza referrer", "qr": "QR fisico", "social": "Social",
	"email": "Email / newsletter", "paid": "Campagne a pagamento", "search": "Ricerca",
	"campaign": "Altre campagne", "internal": "Navigazione interna", "referral": "Referral",
	"unknown": "Sorgente sconosciuta",
}

func matchdayQRChannel(target string) string {
	channel, _ := matchdayEntryChannel(target, "")
	switch channel {
	case "LED":
		return "led"
	case "Giornalino":
		return "match_programme"
	case "Arena":
		return "arena"
	case "Remoto":
		return "remote"
	default:
		return "qr"
	}
}

func matchdayQRTokenKey(token string) string {
	if len(token) != 32 {
		return ""
	}
	data, err := hex.DecodeString(token)
	if err != nil {
		return ""
	}
	key := sha256.Sum256(data)
	return hex.EncodeToString(key[:])
}

// Capture the redirect before the browser starts loading the match page.
// A later page view confirms arrival even if the same signal is retried.
func (rt *_router) matchdayQRDestination(r *http.Request, redirectID int, target string) (string, error) {
	u, err := url.Parse(target)
	if err != nil || u.User != nil || u.Opaque != "" || (u.Host != "" && (!strings.EqualFold(u.Host, r.Host) || (u.Scheme != "http" && u.Scheme != "https"))) || (u.Host == "" && u.Scheme != "") {
		return target, nil
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) != 2 || parts[1] != "new" || parts[0] == "" {
		return target, nil
	}
	if strings.EqualFold(r.Header.Get("Purpose"), "prefetch") || strings.Contains(strings.ToLower(r.Header.Get("Sec-Purpose")), "prefetch") {
		return target, nil
	}
	org, err := rt.db.GetOrganizationBySlug(parts[0])
	if errors.Is(err, sql.ErrNoRows) {
		return target, nil
	}
	if err != nil {
		return target, err
	}
	event, err := rt.db.GetActiveEvent(org.ID)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && event.IsConcluded) {
		return target, nil
	}
	if err != nil {
		return target, err
	}
	query := u.Query()
	for _, key := range []string{"eventId", "eventID"} {
		if values, present := query[key]; present && len(values) > 0 {
			requested, err := strconv.Atoi(values[0])
			if err != nil || requested != event.ID {
				return target, nil
			}
			break
		}
	}
	var token [16]byte
	if _, err := rand.Read(token[:]); err != nil {
		return target, err
	}
	value := hex.EncodeToString(token[:])
	result, err := rt.db.SQLConn().ExecContext(r.Context(), `INSERT INTO matchday_qr_arrivals(token_key,organization_id,event_id,redirect_id,channel)
		SELECT ?,organization_id,id,?,? FROM events
		WHERE organization_id=? AND id=? AND is_active=1 AND is_concluded=0`,
		matchdayQRTokenKey(value), redirectID, matchdayQRChannel(target), org.ID, event.ID)
	if err != nil {
		return target, err
	}
	if n, err := result.RowsAffected(); n != 1 || err != nil {
		return target, err
	}
	query.Set(matchdayQRParameter, value)
	u.RawQuery = query.Encode()
	return u.String(), nil
}

func confirmMatchdayQRArrival(tx *sql.Tx, orgID, eventID int, token, deviceKey, sessionID string) (string, error) {
	key := matchdayQRTokenKey(token)
	if key == "" {
		return "", nil
	}
	session := sha256.Sum256([]byte(deviceKey + ":" + sessionID))
	landingKey := hex.EncodeToString(session[:])
	var channel string
	err := tx.QueryRow(`UPDATE matchday_qr_arrivals SET landed_at=COALESCE(landed_at,CURRENT_TIMESTAMP), landing_key=?
		WHERE token_key=? AND organization_id=? AND event_id=? AND opened_at>=datetime('now','-1 day')
		AND (landing_key IS NULL OR landing_key=?) RETURNING channel`, landingKey, key, orgID, eventID, landingKey).Scan(&channel)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return channel, nil
}

func (f *matchdayFacts) hasModernCollection() bool {
	for _, s := range f.Signals {
		if s.Name == "page_view" && matchdayEntryChannels[reportMetricString(s.Metadata, "entry_channel")] != "" {
			return true
		}
	}
	return false
}

func loadMatchdayQRArrivals(ctx context.Context, q matchdayQueryer, orgID int, data map[int]*matchdayFacts) error {
	in, args := matchdayIDs(data)
	return matchdayRead(ctx, q, `SELECT event_id,COUNT(*),COUNT(landed_at) FROM matchday_qr_arrivals
		WHERE organization_id=? AND event_id IN (`+in+`) GROUP BY event_id`, append([]interface{}{orgID}, args...), func(rows *sql.Rows) error {
		var id, opens, landings int
		if err := rows.Scan(&id, &opens, &landings); err != nil {
			return err
		}
		data[id].qrOpens, data[id].qrLandings = opens, landings
		return nil
	})
}
