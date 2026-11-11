CREATE TABLE IF NOT EXISTS club_mvp_matches (
    match_id INTEGER PRIMARY KEY REFERENCES club_matches(id) ON DELETE CASCADE,
    site_id INTEGER NOT NULL REFERENCES club_sites(id) ON DELETE CASCADE,
    enabled INTEGER NOT NULL DEFAULT 0 CHECK (enabled IN (0, 1)),
    qr_target INTEGER NOT NULL DEFAULT 0 CHECK (qr_target IN (0, 1)),
    opened_at TEXT,
    closed_at TEXT,
    finalized_at TEXT
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_club_mvp_qr_target ON club_mvp_matches(site_id) WHERE qr_target = 1;

-- The player ID is copied from the division roster. Keep the match snapshot
-- when the roster entry is edited or removed.
CREATE TABLE IF NOT EXISTS club_mvp_callups (
    match_id INTEGER NOT NULL REFERENCES club_mvp_matches(match_id) ON DELETE CASCADE,
    player_id INTEGER NOT NULL,
    first_name TEXT NOT NULL,
    last_name TEXT NOT NULL,
    jersey_number INTEGER NOT NULL,
    role TEXT NOT NULL,
    position INTEGER NOT NULL,
    PRIMARY KEY (match_id, player_id)
);

CREATE TABLE IF NOT EXISTS club_mvp_votes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    match_id INTEGER NOT NULL,
    player_id INTEGER NOT NULL,
    voter_hash BLOB NOT NULL,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (match_id, voter_hash),
    FOREIGN KEY (match_id, player_id) REFERENCES club_mvp_callups(match_id, player_id)
);

CREATE INDEX IF NOT EXISTS idx_club_mvp_votes_player ON club_mvp_votes(match_id, player_id);

CREATE TABLE IF NOT EXISTS club_mvp_cookie_key (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    secret BLOB NOT NULL CHECK (length(secret) = 32)
);

-- Both the live console and the calendar editor update this row. Close the
-- ballot in the same SQLite write that changes the match state.
CREATE TRIGGER IF NOT EXISTS club_mvp_close_match
AFTER UPDATE OF status ON club_matches
WHEN NEW.status <> OLD.status AND NEW.status <> 'live'
BEGIN
    UPDATE club_mvp_matches
    SET closed_at = CASE WHEN enabled = 1 THEN CURRENT_TIMESTAMP ELSE closed_at END,
        finalized_at = CASE WHEN NEW.status = 'completed' THEN COALESCE(finalized_at, CURRENT_TIMESTAMP) ELSE finalized_at END,
        enabled = 0,
        qr_target = 0
    WHERE match_id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS club_mvp_keep_match_division
BEFORE UPDATE OF team_id ON club_matches
WHEN NEW.team_id <> OLD.team_id AND EXISTS (SELECT 1 FROM club_mvp_votes WHERE match_id = OLD.id)
BEGIN
    SELECT RAISE(ABORT, 'mvp_has_votes');
END;

CREATE TRIGGER IF NOT EXISTS club_mvp_reset_division
AFTER UPDATE OF team_id ON club_matches
WHEN NEW.team_id <> OLD.team_id
BEGIN
    DELETE FROM club_mvp_callups WHERE match_id = OLD.id;
    DELETE FROM club_mvp_matches WHERE match_id = OLD.id;
END;

CREATE TRIGGER IF NOT EXISTS club_mvp_keep_match_history
BEFORE DELETE ON club_matches
WHEN EXISTS (SELECT 1 FROM club_mvp_votes WHERE match_id = OLD.id)
BEGIN
    SELECT RAISE(ABORT, 'mvp_has_votes');
END;

-- Existing deployments may open SQLite with foreign_keys disabled.
CREATE TRIGGER IF NOT EXISTS club_mvp_delete_match
AFTER DELETE ON club_matches
BEGIN
    DELETE FROM club_mvp_callups WHERE match_id = OLD.id;
    DELETE FROM club_mvp_matches WHERE match_id = OLD.id;
END;
