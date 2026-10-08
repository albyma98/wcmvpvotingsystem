package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"
)

var ErrProtectedOrganization = errors.New("protected organization cannot be deleted")
var ErrOrganizationSharedData = errors.New("organization data is referenced by another organization")

// DeleteOrganizationData removes an organization and its owned data atomically.
// Event IDs are returned so the API can remove their local media directories.
func (db *appdbimpl) DeleteOrganizationData(id int) ([]int, error) {
	if id <= 0 {
		return nil, sql.ErrNoRows
	}
	ctx := context.Background()
	conn, err := db.c.Conn(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	// Foreign keys are connection scoped in SQLite; enable them on the exact
	// connection that performs the deletion, regardless of pool configuration.
	if _, err := conn.ExecContext(ctx, `PRAGMA foreign_keys = ON`); err != nil {
		return nil, err
	}
	var foreignKeys int
	if err := conn.QueryRowContext(ctx, `PRAGMA foreign_keys`).Scan(&foreignKeys); err != nil {
		return nil, err
	}
	if foreignKeys != 1 {
		return nil, errors.New("foreign key enforcement unavailable")
	}
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `PRAGMA defer_foreign_keys = ON`); err != nil {
		return nil, err
	}

	var slug string
	var teamID sql.NullInt64
	if err := tx.QueryRowContext(ctx, `SELECT slug, team_id FROM organizations WHERE id = ?`, id).Scan(&slug, &teamID); err != nil {
		return nil, err
	}
	if strings.HasPrefix(slug, "__") {
		return nil, ErrProtectedOrganization
	}

	tempTables := map[string]string{
		"purge_org_events":   `SELECT id FROM events WHERE organization_id = ?`,
		"purge_org_fans":     `SELECT id FROM fan_profiles WHERE organization_id = ?`,
		"purge_org_sponsors": `SELECT id FROM sponsors WHERE organization_id = ?`,
		"purge_org_players":  `SELECT id FROM players WHERE organization_id = ?`,
		"purge_org_sites":    `SELECT id FROM club_sites WHERE organization_id = ?`,
		"purge_org_coupons":  `SELECT id FROM coupons WHERE organization_id = ? OR sponsor_id IN (SELECT id FROM purge_org_sponsors)`,
	}
	for _, name := range []string{"purge_org_events", "purge_org_fans", "purge_org_sponsors", "purge_org_players", "purge_org_sites", "purge_org_coupons"} {
		if _, err := tx.ExecContext(ctx, `CREATE TEMP TABLE `+name+` (id INTEGER PRIMARY KEY)`); err != nil {
			return nil, fmt.Errorf("creating %s: %w", name, err)
		}
	}
	tables, err := organizationDeletionTables(ctx, tx)
	if err != nil {
		return nil, err
	}
	for _, name := range []string{"purge_org_events", "purge_org_fans", "purge_org_sponsors", "purge_org_players", "purge_org_sites", "purge_org_coupons"} {
		if name == "purge_org_sites" && tables["club_sites"] == nil {
			continue
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO `+name+` (id) `+tempTables[name], id); err != nil {
			return nil, fmt.Errorf("collecting %s: %w", name, err)
		}
	}

	eventRows, err := tx.QueryContext(ctx, `SELECT id FROM purge_org_events`)
	if err != nil {
		return nil, err
	}
	var eventIDs []int
	for eventRows.Next() {
		var eventID int
		if err := eventRows.Scan(&eventID); err != nil {
			eventRows.Close()
			return nil, err
		}
		eventIDs = append(eventIDs, eventID)
	}
	err = eventRows.Err()
	eventRows.Close()
	if err != nil {
		return nil, err
	}

	// A player owned by this organization cannot be removed while another
	// organization's event still uses that player in a vote.
	var sharedPlayers int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM votes WHERE player_id IN (SELECT id FROM purge_org_players) AND event_id NOT IN (SELECT id FROM purge_org_events)`).Scan(&sharedPlayers); err != nil {
		return nil, err
	}
	if sharedPlayers > 0 {
		return nil, ErrOrganizationSharedData
	}
	var sharedCoupons int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM coupons WHERE sponsor_id IN (SELECT id FROM purge_org_sponsors) AND organization_id NOT IN (?, 0)`, id).Scan(&sharedCoupons); err != nil {
		return nil, err
	}
	if sharedCoupons > 0 {
		return nil, ErrOrganizationSharedData
	}
	// QR redirects have no organization_id. A redirect to this society's
	// route is still society-specific; match path boundaries to avoid removing
	// redirects to similarly named societies (for example /prima and /primavera).
	if tables["qr_redirects"]["target_path"] {
		basePath := "/" + strings.Trim(slug, "/")
		if _, err := tx.ExecContext(ctx, `DELETE FROM qr_redirects WHERE target_path = ?
			OR substr(target_path, 1, length(?) + 1) IN (? || '/', ? || '?', ? || '#')`,
			basePath, basePath, basePath, basePath, basePath); err != nil {
			return nil, fmt.Errorf("deleting society QR redirects: %w", err)
		}
	}

	// These links do not carry organization_id or event_id. Delete them before
	// their parents so older databases without cascading FKs are covered too.
	exceptions := []struct{ table, condition string }{
		{"event_quiz_questions", `quiz_id IN (SELECT id FROM purge_org_events)`},
		{"court_operator_sessions", `operator_id IN (SELECT id FROM court_operators WHERE event_id IN (SELECT id FROM purge_org_events))`},
		{"club_admin_sessions", `site_id IN (SELECT id FROM purge_org_sites)`},
		{"user_coupons", `coupon_id IN (SELECT id FROM purge_org_coupons) OR user_id IN (SELECT id FROM purge_org_fans)`},
	}
	for _, item := range exceptions {
		if tables[item.table] == nil {
			continue
		}
		if item.table == "court_operator_sessions" && tables["court_operators"] == nil {
			continue
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM `+quoteSQLiteIdentifier(item.table)+` WHERE `+item.condition); err != nil {
			return nil, fmt.Errorf("deleting %s: %w", item.table, err)
		}
	}

	// Preserve records from other organizations that only reference one of this
	// organization's fan profiles; remove the personal link instead.
	for _, item := range []struct{ table, column, replacement string }{
		{"votes", "user_id", "NULL"},
		{"event_prizes", "winner_user_id", "NULL"},
		{"branded_game_participations", "user_id", "NULL"},
		{"ai_interactions", "user_id", "0"},
	} {
		if !tables[item.table][item.column] {
			continue
		}
		query := `UPDATE ` + quoteSQLiteIdentifier(item.table) + ` SET ` + quoteSQLiteIdentifier(item.column) + ` = ` + item.replacement +
			` WHERE ` + quoteSQLiteIdentifier(item.column) + ` IN (SELECT id FROM purge_org_fans)`
		if tables[item.table]["event_id"] {
			query += ` AND event_id NOT IN (SELECT id FROM purge_org_events)`
		}
		if tables[item.table]["organization_id"] {
			query += ` AND organization_id <> ?`
			if _, err := tx.ExecContext(ctx, query, id); err != nil {
				return nil, fmt.Errorf("unlinking %s fans: %w", item.table, err)
			}
		} else if _, err := tx.ExecContext(ctx, query); err != nil {
			return nil, fmt.Errorf("unlinking %s fans: %w", item.table, err)
		}
	}

	// All rows with direct ownership columns are removed, including tables added
	// by optional modules, without relying on every historical FK being present.
	names := make([]string, 0, len(tables))
	for name := range tables {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if name == "organizations" {
			continue
		}
		columns := tables[name]
		var predicates []string
		if columns["organization_id"] {
			predicates = append(predicates, `organization_id = ?`)
		}
		for column, target := range map[string]string{
			"event_id": "purge_org_events", "site_id": "purge_org_sites", "fan_id": "purge_org_fans",
			"sponsor_id": "purge_org_sponsors", "coupon_id": "purge_org_coupons", "player_id": "purge_org_players",
		} {
			if columns[column] {
				predicates = append(predicates, quoteSQLiteIdentifier(column)+` IN (SELECT id FROM `+target+`)`)
			}
		}
		if len(predicates) == 0 {
			continue
		}
		query := `DELETE FROM ` + quoteSQLiteIdentifier(name) + ` WHERE ` + strings.Join(predicates, ` OR `)
		if columns["organization_id"] {
			_, err = tx.ExecContext(ctx, query, id)
		} else {
			_, err = tx.ExecContext(ctx, query)
		}
		if err != nil {
			return nil, fmt.Errorf("deleting %s: %w", name, err)
		}
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM organizations WHERE id = ?`, id); err != nil {
		return nil, err
	}
	// teams is global. Remove the linked row only if no remaining society,
	// event or player references it.
	if teamID.Valid && teamID.Int64 > 0 {
		if _, err := tx.ExecContext(ctx, `DELETE FROM teams WHERE id = ?
			AND NOT EXISTS (SELECT 1 FROM organizations WHERE team_id = ?)
			AND NOT EXISTS (SELECT 1 FROM events WHERE team1_id = ? OR team2_id = ?)
			AND NOT EXISTS (SELECT 1 FROM players WHERE team_id = ?)`, teamID.Int64, teamID.Int64, teamID.Int64, teamID.Int64, teamID.Int64); err != nil {
			return nil, err
		}
	}
	for name := range tempTables {
		if _, err := tx.ExecContext(ctx, `DROP TABLE `+name); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return eventIDs, nil
}

func organizationDeletionTables(ctx context.Context, tx *sql.Tx) (map[string]map[string]bool, error) {
	rows, err := tx.QueryContext(ctx, `SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%'`)
	if err != nil {
		return nil, err
	}
	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			rows.Close()
			return nil, err
		}
		names = append(names, name)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	tables := make(map[string]map[string]bool, len(names))
	for _, name := range names {
		columns := make(map[string]bool)
		info, err := tx.QueryContext(ctx, `PRAGMA table_info(`+quoteSQLiteIdentifier(name)+`)`)
		if err != nil {
			return nil, err
		}
		for info.Next() {
			var cid, required, primary int
			var column, dataType string
			var defaultValue sql.NullString
			if err := info.Scan(&cid, &column, &dataType, &required, &defaultValue, &primary); err != nil {
				info.Close()
				return nil, err
			}
			columns[column] = true
		}
		err = info.Err()
		info.Close()
		if err != nil {
			return nil, err
		}
		tables[name] = columns
	}
	return tables, nil
}

func quoteSQLiteIdentifier(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
}
