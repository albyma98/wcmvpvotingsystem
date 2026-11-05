CREATE TABLE IF NOT EXISTS engagement_report_manual_sources (
 event_id INTEGER NOT NULL REFERENCES events(id) ON DELETE CASCADE,
 field_key TEXT NOT NULL CHECK(field_key='attendees'),
 value REAL NOT NULL CHECK(value>=0 AND value<=10000000 AND value=CAST(value AS INTEGER)),
 unit TEXT NOT NULL CHECK(unit='people'), source_type TEXT NOT NULL,
 source_description TEXT NOT NULL, entered_by INTEGER NOT NULL,
 entered_at TEXT NOT NULL, updated_at TEXT NOT NULL,
 verification_status TEXT NOT NULL CHECK(verification_status IN ('verified','unverified')),
 note TEXT NOT NULL DEFAULT '', revision INTEGER NOT NULL DEFAULT 1,
 PRIMARY KEY(event_id,field_key)
);
CREATE TABLE IF NOT EXISTS engagement_report_source_history (
 id INTEGER PRIMARY KEY AUTOINCREMENT,
 event_id INTEGER NOT NULL REFERENCES events(id) ON DELETE CASCADE,
 field_key TEXT NOT NULL, previous_json TEXT, current_json TEXT NOT NULL,
 changed_by INTEGER NOT NULL, changed_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS engagement_report_source_history_event ON engagement_report_source_history(event_id,id);
CREATE TABLE IF NOT EXISTS engagement_report_analysis_cache (
 cache_key TEXT PRIMARY KEY, event_id INTEGER NOT NULL REFERENCES events(id) ON DELETE CASCADE,
 recipient_profile TEXT NOT NULL, previous_event_id INTEGER NOT NULL DEFAULT 0,
 data_hash TEXT NOT NULL, prompt_version TEXT NOT NULL, model TEXT NOT NULL,
 response_json TEXT NOT NULL DEFAULT '', status TEXT NOT NULL,
 created_at TEXT NOT NULL, updated_at TEXT NOT NULL, version INTEGER NOT NULL DEFAULT 1
);
CREATE INDEX IF NOT EXISTS engagement_report_analysis_event ON engagement_report_analysis_cache(event_id,recipient_profile);
