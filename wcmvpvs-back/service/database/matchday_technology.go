package database

import "database/sql"

const MatchdayProbeIntervalSeconds = 30

// A probe writer can run independently of the API and record unsuccessful
// requests even while the API process is down.
func EnsureMatchdayHealthProbes(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS matchday_health_probes (
		slot_start INTEGER PRIMARY KEY,
		healthy INTEGER NOT NULL CHECK (healthy IN (0, 1))
	)`)
	return err
}

func ensureMatchdayTechnology(db *sql.DB) error {
	if err := EnsureMatchdayHealthProbes(db); err != nil {
		return err
	}
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS matchday_technology_windows (
			id INTEGER PRIMARY KEY,
			event_id INTEGER NOT NULL REFERENCES events(id) ON DELETE CASCADE,
			organization_id INTEGER NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
			opened_at INTEGER NOT NULL,
			closed_at INTEGER
		);
		CREATE INDEX IF NOT EXISTS idx_matchday_technology_windows ON matchday_technology_windows(event_id, opened_at);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_matchday_technology_open_window
			ON matchday_technology_windows(event_id) WHERE closed_at IS NULL;
		CREATE TABLE IF NOT EXISTS matchday_api_requests (
			event_id INTEGER NOT NULL REFERENCES events(id) ON DELETE CASCADE,
			organization_id INTEGER NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
			route TEXT NOT NULL,
			http_status INTEGER NOT NULL,
			latency_ms INTEGER NOT NULL,
			samples INTEGER NOT NULL,
			PRIMARY KEY(event_id, route, http_status, latency_ms)
		);
		CREATE TRIGGER IF NOT EXISTS matchday_technology_window_insert AFTER INSERT ON events
		WHEN NEW.is_active=1 AND NEW.is_concluded=0 AND EXISTS(SELECT 1 FROM organizations o WHERE o.id=NEW.organization_id)
		BEGIN
			INSERT INTO matchday_technology_windows(event_id,organization_id,opened_at)
			VALUES(NEW.id,NEW.organization_id,CAST(strftime('%s','now') AS INTEGER)*1000+CAST(substr(strftime('%f','now'),4,3) AS INTEGER));
		END;
		CREATE TRIGGER IF NOT EXISTS matchday_technology_window_open AFTER UPDATE OF is_active,is_concluded ON events
		WHEN NEW.is_active=1 AND NEW.is_concluded=0 AND (OLD.is_active<>1 OR OLD.is_concluded<>0)
		AND EXISTS(SELECT 1 FROM organizations o WHERE o.id=NEW.organization_id)
		BEGIN
			INSERT INTO matchday_technology_windows(event_id,organization_id,opened_at)
			VALUES(NEW.id,NEW.organization_id,CAST(strftime('%s','now') AS INTEGER)*1000+CAST(substr(strftime('%f','now'),4,3) AS INTEGER));
		END;
		CREATE TRIGGER IF NOT EXISTS matchday_technology_window_close AFTER UPDATE OF is_active,is_concluded ON events
		WHEN OLD.is_active=1 AND OLD.is_concluded=0 AND (NEW.is_active<>1 OR NEW.is_concluded<>0)
		BEGIN
			UPDATE matchday_technology_windows SET closed_at=CAST(strftime('%s','now') AS INTEGER)*1000+CAST(substr(strftime('%f','now'),4,3) AS INTEGER)
			WHERE event_id=OLD.id AND closed_at IS NULL;
		END;
		INSERT INTO matchday_technology_windows(event_id,organization_id,opened_at)
		SELECT e.id,e.organization_id,CAST(strftime('%s','now') AS INTEGER)*1000+CAST(substr(strftime('%f','now'),4,3) AS INTEGER)
		FROM events e JOIN organizations o ON o.id=e.organization_id
		WHERE e.is_active=1 AND e.is_concluded=0
		AND NOT EXISTS(SELECT 1 FROM matchday_technology_windows w WHERE w.event_id=e.id AND w.closed_at IS NULL);
	`)
	return err
}
