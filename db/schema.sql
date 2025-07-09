-- Schema for MVP voting system

CREATE TABLE matches (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    match_date TIMESTAMP NOT NULL
);

CREATE TABLE players (
    id SERIAL PRIMARY KEY,
    match_id INTEGER NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    image_url TEXT
);

CREATE TABLE votes (
    id SERIAL PRIMARY KEY,
    match_id INTEGER NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
    player_id INTEGER NOT NULL REFERENCES players(id) ON DELETE CASCADE,
    ip_address TEXT NOT NULL,
    user_agent TEXT NOT NULL,
    uuid TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(match_id, ip_address, user_agent, uuid)
);
