package api

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"net/http"
	"strings"
	"time"
)

//go:embed merch_migrations/002_matchday.sql
var merchMatchMigration string

func (rt *_router) merchMatchdayGrantMatches(r *http.Request, orgID, matchID int) bool {
	return rt.matchdayRequests != nil && rt.matchdayRequests.grantEvent(r.Header.Get(matchdayGrantHeader), orgID, time.Now()) == matchID
}

func validateMerchMatch(ctx context.Context, tx *sql.Tx, orgID, matchID int) error {
	if matchID == 0 {
		return nil
	}
	var existing int
	err := tx.QueryRowContext(ctx, `SELECT id FROM events WHERE id=? AND organization_id=?`, matchID, orgID).Scan(&existing)
	if errors.Is(err, sql.ErrNoRows) {
		return merchInvalid("Partita non valida per questa società")
	}
	return err
}

func (rt *_router) matchdayRequestDeviceKey(r *http.Request, orgID int) string {
	device := strings.TrimSpace(rt.deviceIDFromRequest(r))
	if len(device) < 8 || len(device) > 200 {
		return ""
	}
	return reportOperationalDeviceKey(orgID, device)
}

func (rt *_router) markMerchMatchCoverage(ctx context.Context, orgID, matchID int, available bool) {
	_, err := rt.db.SQLConn().ExecContext(ctx, `INSERT INTO merch_match_coverage(event_id,organization_id,enabled)
		SELECT id,organization_id,? FROM events WHERE id=? AND organization_id=?
		ON CONFLICT(event_id) DO UPDATE SET enabled=1 WHERE excluded.enabled=1 AND merch_match_coverage.enabled=0`, available, matchID, orgID)
	if err != nil {
		rt.baseLogger.WithError(err).WithField("event_id", matchID).Warn("merch match coverage")
	}
}
