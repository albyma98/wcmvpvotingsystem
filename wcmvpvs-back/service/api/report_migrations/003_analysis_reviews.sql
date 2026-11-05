CREATE TABLE IF NOT EXISTS engagement_report_analysis_reviews (
 cache_key TEXT NOT NULL,
 admin_id INTEGER NOT NULL,
 event_id INTEGER NOT NULL REFERENCES events(id) ON DELETE CASCADE,
 generation_version INTEGER NOT NULL CHECK(generation_version>=0),
 state TEXT NOT NULL CHECK(state IN ('draft','approved','discarded')),
 analysis_json TEXT NOT NULL,
 revision INTEGER NOT NULL CHECK(revision>0),
 updated_at TEXT NOT NULL,
 reviewed_at TEXT NOT NULL DEFAULT '',
 reviewed_by TEXT NOT NULL DEFAULT '',
 PRIMARY KEY(cache_key,admin_id)
);
CREATE INDEX IF NOT EXISTS engagement_report_analysis_reviews_event
 ON engagement_report_analysis_reviews(event_id,admin_id);
