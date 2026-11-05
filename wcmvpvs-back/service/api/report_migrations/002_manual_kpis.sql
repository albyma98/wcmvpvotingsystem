-- Transactionally preserve existing attendance records while allowing explicit
-- KPI completions, scoped separately to an individual sponsor when necessary.
CREATE TABLE engagement_report_manual_sources_v2 (
 event_id INTEGER NOT NULL REFERENCES events(id) ON DELETE CASCADE,
 field_key TEXT NOT NULL,
 sponsor_id INTEGER NOT NULL DEFAULT 0 CHECK(sponsor_id>=0),
 value REAL NOT NULL CHECK(value>=0 AND value<=1000000000000),
 unit TEXT NOT NULL CHECK(unit IN ('people','count','percent','seconds')),
 source_type TEXT NOT NULL, source_description TEXT NOT NULL,
 entered_by INTEGER NOT NULL, entered_at TEXT NOT NULL, updated_at TEXT NOT NULL,
 verification_status TEXT NOT NULL CHECK(verification_status IN ('verified','unverified')),
 note TEXT NOT NULL DEFAULT '', revision INTEGER NOT NULL DEFAULT 1,
 CHECK(unit NOT IN ('people','count') OR value=CAST(value AS INTEGER)),
 CHECK(unit<>'percent' OR value<=100),
 PRIMARY KEY(event_id,field_key,sponsor_id)
);
INSERT INTO engagement_report_manual_sources_v2
 (event_id,field_key,value,unit,source_type,source_description,entered_by,entered_at,updated_at,verification_status,note,revision)
 SELECT event_id,field_key,value,unit,source_type,source_description,entered_by,entered_at,updated_at,verification_status,note,revision FROM engagement_report_manual_sources;
DROP TABLE engagement_report_manual_sources;
ALTER TABLE engagement_report_manual_sources_v2 RENAME TO engagement_report_manual_sources;
ALTER TABLE engagement_report_source_history ADD COLUMN sponsor_id INTEGER NOT NULL DEFAULT 0;
