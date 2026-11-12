CREATE TABLE IF NOT EXISTS matchday_kpi_manual_revisions (
    organization_id INTEGER NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    event_id INTEGER NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    metric_code TEXT NOT NULL,
    revision INTEGER NOT NULL CHECK (revision > 0),
    value REAL NOT NULL CHECK (value BETWEEN 0 AND 1000000000),
    unit TEXT NOT NULL CHECK (unit IN ('count', 'eur')),
    source_note TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    updated_by INTEGER NOT NULL,
    CHECK (unit <> 'count' OR value = CAST(value AS INTEGER)),
    PRIMARY KEY (organization_id, event_id, metric_code, revision)
);
CREATE INDEX IF NOT EXISTS idx_matchday_kpi_manual_event ON matchday_kpi_manual_revisions(event_id);
