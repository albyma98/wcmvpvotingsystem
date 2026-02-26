ALTER TABLE events ADD COLUMN live_score_url TEXT;
ALTER TABLE events ADD COLUMN live_score_source TEXT NOT NULL DEFAULT 'legavolley_html';
