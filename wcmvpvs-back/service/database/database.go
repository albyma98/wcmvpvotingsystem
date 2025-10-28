/*
Package database is the middleware between the app database and the code. All data (de)serialization (save/load) from a
persistent database are handled here. Database specific logic should never escape this package.

To use this package you need to apply migrations to the database if needed/wanted, connect to it (using the database
data source name from config), and then initialize an instance of AppDatabase from the DB connection.

For example, this code adds a parameter in `webapi` executable for the database data source name (add it to the
main.WebAPIConfiguration structure):

	DB struct {
		Filename string `conf:""`
	}

This is an example on how to migrate the DB and connect to it:

	// Start Database
	logger.Println("initializing database support")
	db, err := sql.Open("sqlite3", "./foo.db")
	if err != nil {
		logger.WithError(err).Error("error opening SQLite DB")
		return fmt.Errorf("opening SQLite: %w", err)
	}
	defer func() {
		logger.Debug("database stopping")
		_ = db.Close()
	}()

Then you can initialize the AppDatabase and pass it to the api package.
*/
package database

import (
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

// AppDatabase is the high level interface for the DB
type Team struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type Player struct {
	ID           int    `json:"id"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	Role         string `json:"role"`
	JerseyNumber int    `json:"jersey_number"`
	ImageURL     string `json:"image_url"`
	TeamID       int    `json:"team_id"`
}

type Event struct {
	ID            int          `json:"id"`
	Team1ID       int          `json:"team1_id"`
	Team2ID       int          `json:"team2_id"`
	StartDateTime string       `json:"start_datetime"`
	Location      string       `json:"location"`
	IsActive      bool         `json:"is_active"`
	VotesClosed   bool         `json:"votes_closed"`
	IsConcluded   bool         `json:"is_concluded"`
	Team1Name     string       `json:"team1_name,omitempty"`
	Team2Name     string       `json:"team2_name,omitempty"`
	Prizes        []EventPrize `json:"prizes,omitempty"`
}

type EventPrize struct {
	ID       int               `json:"id"`
	EventID  int               `json:"event_id"`
	Name     string            `json:"name"`
	Position int               `json:"position"`
	Winner   *EventPrizeWinner `json:"winner,omitempty"`
}

type EventPrizeWinner struct {
	VoteID          int    `json:"vote_id"`
	TicketCode      string `json:"ticket_code"`
	PlayerID        int    `json:"player_id"`
	PlayerFirstName string `json:"player_first_name"`
	PlayerLastName  string `json:"player_last_name"`
	AssignedAt      string `json:"assigned_at"`
}

type Vote struct {
	ID              int    `json:"id"`
	EventID         int    `json:"event_id"`
	PlayerID        int    `json:"player_id"`
	TicketCode      string `json:"ticket_code"`
	TicketSignature string `json:"ticket_signature"`
	DeviceID        string `json:"device_id"`
	CreatedAt       string `json:"created_at"`
}

type EventVoteResult struct {
	PlayerID   int    `json:"player_id"`
	Votes      int    `json:"votes"`
	LastVoteAt string `json:"last_vote_at"`
}

type EventTicket struct {
	VoteID          int    `json:"vote_id"`
	TicketCode      string `json:"ticket_code"`
	TicketSignature string `json:"ticket_signature"`
	PlayerID        int    `json:"player_id"`
	PlayerFirstName string `json:"player_first_name"`
	PlayerLastName  string `json:"player_last_name"`
	CreatedAt       string `json:"created_at"`
}

type TicketValidationPrize struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Position int    `json:"position"`
}

type TicketValidationResult struct {
	VoteID          int                    `json:"vote_id"`
	EventID         int                    `json:"event_id"`
	PlayerID        int                    `json:"player_id"`
	TicketCode      string                 `json:"ticket_code"`
	TicketSignature string                 `json:"ticket_signature"`
	PlayerFirstName string                 `json:"player_first_name"`
	PlayerLastName  string                 `json:"player_last_name"`
	CreatedAt       string                 `json:"created_at"`
	AssignedPrize   *TicketValidationPrize `json:"assigned_prize,omitempty"`
}

type Admin struct {
	ID           int    `json:"id"`
	Username     string `json:"username"`
	PasswordHash string `json:"password_hash"`
	Role         string `json:"role"`
	CreatedAt    string `json:"created_at"`
}

type Sponsor struct {
	ID       int    `json:"id"`
	Position int    `json:"position"`
	Name     string `json:"name"`
	LogoData string `json:"logo_data"`
	LinkURL  string `json:"link_url"`
	IsActive bool   `json:"is_active"`
}

type SponsorClickStat struct {
	SponsorID int    `json:"sponsor_id"`
	Name      string `json:"name"`
	LinkURL   string `json:"link_url"`
	Clicks    int    `json:"clicks"`
}

type EventMVP struct {
	PlayerID   int    `json:"player_id"`
	FirstName  string `json:"first_name"`
	LastName   string `json:"last_name"`
	Votes      int    `json:"votes"`
	LastVoteAt string `json:"last_vote_at"`
}

// AppDatabase is the high level interface for the DB
type AppDatabase interface {
	GetName() (string, error)
	SetName(name string) error
	AddVote(eventID, playerID int, code, signature, deviceID string) error
	CreateTeam(name string) (int, error)
	ListTeams() ([]Team, error)
	UpdateTeam(id int, name string) error
	DeleteTeam(id int) error
	CreatePlayer(p Player) (int, error)
	ListPlayers() ([]Player, error)
	UpdatePlayer(p Player) error
	DeletePlayer(id int) error
	CreateEvent(e Event) (int, error)
	ListEvents() ([]Event, error)
	UpdateEvent(e Event) error
	DeleteEvent(id int) error
	SetActiveEvent(eventID int) error
	ClearActiveEvent() error
	CloseEventVoting(eventID int) error
	ConcludeEvent(eventID int) error
	GetActiveEvent() (Event, error)
	ListVotes() ([]Vote, error)
	ListEventTickets(eventID int) ([]EventTicket, error)
	ValidateTicket(eventID int, code string) (TicketValidationResult, error)
	RedeemTicket(eventID int, code, signature string) (bool, error)
	ListEventPrizes(eventID int) ([]EventPrize, error)
	AssignPrizeWinner(eventID, prizeID, voteID int) (EventPrize, error)
	ClearPrizeWinner(eventID, prizeID int) error
	GetEventResults(eventID int) ([]EventVoteResult, error)
	GetEventVoteCount(eventID int) (int, error)
	ListEventVoteTimestamps(eventID int) ([]time.Time, error)
	GetEventMVP(eventID int) (EventMVP, error)
	DeleteVote(id int) error
	CreateAdmin(a Admin) (int, error)
	ListAdmins() ([]Admin, error)
	UpdateAdmin(a Admin) error
	DeleteAdmin(id int) error
	GetAdminByUsername(username string) (Admin, error)
	GetAdminByID(id int) (Admin, error)
	CreateSponsor(s Sponsor) (int, error)
	UpdateSponsor(s Sponsor) error
	DeleteSponsor(id int) error
	ListSponsors() ([]Sponsor, error)
	ListActiveSponsors() ([]Sponsor, error)
	RecordSponsorClick(eventID, sponsorID int) error
	GetSponsorClickStats(eventID int) ([]SponsorClickStat, error)
	PurgeEventData(eventID int) error
	Ping() error
}

type appdbimpl struct {
	c *sql.DB
}

const maxSponsorSlots = 4

var (
	ErrMaxSponsors             = errors.New("maximum number of sponsors reached")
	ErrInvalidSponsorPos       = errors.New("invalid sponsor position")
	ErrInvalidSponsorData      = errors.New("invalid sponsor data")
	ErrPrizeAlreadyAssigned    = errors.New("prize already has a winner")
	ErrPrizeWinnerConflict     = errors.New("winner already assigned to another prize")
	ErrPrizeVoteMismatch       = errors.New("selected ticket is not valid for this event")
	ErrPrizeLockedByWinner     = errors.New("cannot remove a prize that already has a winner")
	ErrTicketSignatureMismatch = errors.New("ticket signature mismatch")
	ErrEventAlreadyConcluded   = errors.New("event already concluded")
)

// New returns a new instance of AppDatabase based on the SQLite connection `db`.
// `db` is required - an error will be returned if `db` is `nil`.
func New(db *sql.DB) (AppDatabase, error) {
	if db == nil {
		return nil, errors.New("database is required when building a AppDatabase")
	}

	// Check if table exists. If not, the database is empty, and we need to create the structure
	var tableName string
	err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='example_table';`).Scan(&tableName)
	if errors.Is(err, sql.ErrNoRows) {
		sqlStmt := `CREATE TABLE example_table (id INTEGER NOT NULL PRIMARY KEY, name TEXT);`
		_, err = db.Exec(sqlStmt)
		if err != nil {
			return nil, fmt.Errorf("error creating database structure: %w", err)
		}
	}
	// Create teams table if not exists
	err = db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='teams';`).Scan(&tableName)
	if errors.Is(err, sql.ErrNoRows) {
		sqlStmt := `CREATE TABLE teams (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL);`
		_, err = db.Exec(sqlStmt)
		if err != nil {
			return nil, fmt.Errorf("error creating teams table: %w", err)
		}
	}

	// Create players table if not exists
	err = db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='players';`).Scan(&tableName)
	if errors.Is(err, sql.ErrNoRows) {
		sqlStmt := `CREATE TABLE players (id INTEGER PRIMARY KEY AUTOINCREMENT, first_name TEXT NOT NULL, last_name TEXT NOT NULL, role TEXT NOT NULL, jersey_number INTEGER, image_url TEXT, team_id INTEGER NOT NULL, FOREIGN KEY (team_id) REFERENCES teams(id));`
		_, err = db.Exec(sqlStmt)
		if err != nil {
			return nil, fmt.Errorf("error creating players table: %w", err)
		}
	} else {
		// attempt schema update if column missing
		_, _ = db.Exec(`ALTER TABLE players ADD COLUMN image_url TEXT`)
	}

	// Create events table if not exists
	err = db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='events';`).Scan(&tableName)
	if errors.Is(err, sql.ErrNoRows) {
		sqlStmt := `CREATE TABLE events (id INTEGER PRIMARY KEY AUTOINCREMENT, team1_id INTEGER NOT NULL, team2_id INTEGER NOT NULL, start_datetime TEXT NOT NULL, location TEXT, FOREIGN KEY (team1_id) REFERENCES teams(id), FOREIGN KEY (team2_id) REFERENCES teams(id));`
		_, err = db.Exec(sqlStmt)
		if err != nil {
			return nil, fmt.Errorf("error creating events table: %w", err)
		}

	}

	err = db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='event_prizes';`).Scan(&tableName)
	if errors.Is(err, sql.ErrNoRows) {
		sqlStmt := `CREATE TABLE event_prizes (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        event_id INTEGER NOT NULL,
        name TEXT NOT NULL,
        position INTEGER NOT NULL DEFAULT 1,
        winner_vote_id INTEGER,
        winner_assigned_at TEXT,
        FOREIGN KEY (event_id) REFERENCES events(id) ON DELETE CASCADE,
        FOREIGN KEY (winner_vote_id) REFERENCES votes(id)
);`
		_, err = db.Exec(sqlStmt)
		if err != nil {
			return nil, fmt.Errorf("error creating event_prizes table: %w", err)
		}
	}

	if _, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_event_prizes_event ON event_prizes(event_id)`); err != nil {
		return nil, fmt.Errorf("error ensuring event_prizes event index: %w", err)
	}

	if _, err = db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_event_prizes_event_position ON event_prizes(event_id, position)`); err != nil {
		return nil, fmt.Errorf("error ensuring event_prizes position index: %w", err)
	}

	if _, err = db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_event_prizes_winner_vote ON event_prizes(winner_vote_id)`); err != nil {
		return nil, fmt.Errorf("error ensuring event_prizes winner index: %w", err)
	}

	if _, err = db.Exec(`ALTER TABLE events ADD COLUMN is_active INTEGER NOT NULL DEFAULT 0`); err != nil {
		if !strings.Contains(err.Error(), "duplicate column name") {
			return nil, fmt.Errorf("error ensuring events active column: %w", err)
		}
	}

	if _, err = db.Exec(`ALTER TABLE events ADD COLUMN votes_closed INTEGER NOT NULL DEFAULT 0`); err != nil {
		if !strings.Contains(err.Error(), "duplicate column name") {
			return nil, fmt.Errorf("error ensuring events votes closed column: %w", err)
		}
	}

	if _, err = db.Exec(`ALTER TABLE events ADD COLUMN is_concluded INTEGER NOT NULL DEFAULT 0`); err != nil {
		if !strings.Contains(err.Error(), "duplicate column name") {
			return nil, fmt.Errorf("error ensuring events concluded column: %w", err)
		}
	}

	// Create admins table if not exists
	err = db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='admins';`).Scan(&tableName)
	if errors.Is(err, sql.ErrNoRows) {
		sqlStmt := `CREATE TABLE admins (id INTEGER PRIMARY KEY AUTOINCREMENT, username TEXT NOT NULL UNIQUE, password_hash TEXT NOT NULL, role TEXT DEFAULT 'staff', created_at TEXT DEFAULT CURRENT_TIMESTAMP);`
		_, err = db.Exec(sqlStmt)
		if err != nil {
			return nil, fmt.Errorf("error creating admins table: %w", err)
		}
	}

	// Create votes table if not exists
	err = db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='votes';`).Scan(&tableName)
	if errors.Is(err, sql.ErrNoRows) {
		sqlStmt := `CREATE TABLE votes (id INTEGER PRIMARY KEY AUTOINCREMENT, event_id INTEGER NOT NULL, player_id INTEGER NOT NULL, ticket_code TEXT NOT NULL, ticket_signature TEXT NOT NULL, device_id TEXT NOT NULL, created_at TEXT DEFAULT CURRENT_TIMESTAMP, FOREIGN KEY (event_id) REFERENCES events(id), FOREIGN KEY (player_id) REFERENCES players(id));`
		_, err = db.Exec(sqlStmt)
		if err != nil {
			return nil, fmt.Errorf("error creating votes table: %w", err)
		}
		_, err = db.Exec(`CREATE UNIQUE INDEX unique_vote_per_event_device ON votes (event_id, device_id);`)
		if err != nil {
			return nil, fmt.Errorf("error creating votes index: %w", err)
		}
	}
	_, err = db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS unique_vote_per_event_device ON votes (event_id, device_id);`)
	if err != nil {
		return nil, fmt.Errorf("error ensuring votes device index: %w", err)
	}
	_, err = db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS unique_vote_code_per_event ON votes (event_id, ticket_code);`)
	if err != nil {
		return nil, fmt.Errorf("error ensuring votes code index: %w", err)
	}
	if _, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_votes_event ON votes (event_id);`); err != nil {
		return nil, fmt.Errorf("error ensuring votes event index: %w", err)
	}

	err = db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='tickets';`).Scan(&tableName)
	if errors.Is(err, sql.ErrNoRows) {
		sqlStmt := `CREATE TABLE tickets (
        event_id INTEGER NOT NULL,
        code TEXT NOT NULL,
        signature TEXT NOT NULL,
        redeemed_at TEXT,
        PRIMARY KEY (event_id, code)
);`
		_, err = db.Exec(sqlStmt)
		if err != nil {
			return nil, fmt.Errorf("error creating tickets table: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("error verifying tickets table: %w", err)
	}

	if _, err = db.Exec(`ALTER TABLE tickets ADD COLUMN redeemed_at TEXT`); err != nil {
		if !strings.Contains(err.Error(), "duplicate column name") {
			return nil, fmt.Errorf("error ensuring tickets redeemed_at column: %w", err)
		}
	}

	if _, err = db.Exec(`ALTER TABLE tickets ADD COLUMN signature TEXT`); err != nil {
		if !strings.Contains(err.Error(), "duplicate column name") {
			return nil, fmt.Errorf("error ensuring tickets signature column: %w", err)
		}
	}

	err = db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='sponsors';`).Scan(&tableName)
	if errors.Is(err, sql.ErrNoRows) {
		sqlStmt := `CREATE TABLE sponsors (id INTEGER PRIMARY KEY AUTOINCREMENT, position INTEGER NOT NULL UNIQUE, name TEXT NOT NULL, logo_data TEXT NOT NULL, link_url TEXT, is_active INTEGER NOT NULL DEFAULT 1, CHECK(position BETWEEN 1 AND ` + fmt.Sprint(maxSponsorSlots) + `));`
		_, err = db.Exec(sqlStmt)
		if err != nil {
			return nil, fmt.Errorf("error creating sponsors table: %w", err)
		}
	}

	err = db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='sponsor_clicks';`).Scan(&tableName)
	if errors.Is(err, sql.ErrNoRows) {
		sqlStmt := `CREATE TABLE sponsor_clicks (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        event_id INTEGER NOT NULL,
        sponsor_id INTEGER NOT NULL,
        clicked_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
        FOREIGN KEY (event_id) REFERENCES events(id) ON DELETE CASCADE,
        FOREIGN KEY (sponsor_id) REFERENCES sponsors(id) ON DELETE CASCADE
);`
		_, err = db.Exec(sqlStmt)
		if err != nil {
			return nil, fmt.Errorf("error creating sponsor_clicks table: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("error verifying sponsor_clicks table: %w", err)
	}

	if _, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_sponsor_clicks_event ON sponsor_clicks(event_id)`); err != nil {
		return nil, fmt.Errorf("error ensuring sponsor_clicks event index: %w", err)
	}
	if _, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_sponsor_clicks_sponsor ON sponsor_clicks(sponsor_id)`); err != nil {
		return nil, fmt.Errorf("error ensuring sponsor_clicks sponsor index: %w", err)
	}

	return &appdbimpl{
		c: db,
	}, nil
}

func (db *appdbimpl) Ping() error {
	return db.c.Ping()
}

// AddVote stores a vote in the database
func (db *appdbimpl) AddVote(eventID, playerID int, code, signature, deviceID string) error {
	_, err := db.c.Exec(`INSERT INTO votes (event_id, player_id, ticket_code, ticket_signature, device_id) VALUES (?, ?, ?, ?, ?)`, eventID, playerID, code, signature, deviceID)
	return err
}

// GetEventVoteCount returns the total number of votes for a specific event
func (db *appdbimpl) GetEventVoteCount(eventID int) (int, error) {
	var count int
	err := db.c.QueryRow(`SELECT COUNT(1) FROM votes WHERE event_id = ?`, eventID).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (db *appdbimpl) ListEventVoteTimestamps(eventID int) ([]time.Time, error) {
	rows, err := db.c.Query(`SELECT created_at FROM votes WHERE event_id = ? ORDER BY created_at ASC`, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var timestamps []time.Time
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		ts, parseErr := parseSQLiteTimestamp(raw)
		if parseErr != nil {
			continue
		}
		timestamps = append(timestamps, ts)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return timestamps, nil
}

func (db *appdbimpl) GetEventMVP(eventID int) (EventMVP, error) {
	row := db.c.QueryRow(`
SELECT v.player_id,
       IFNULL(p.first_name, ''),
        IFNULL(p.last_name, ''),
        COUNT(v.id) AS votes,
        IFNULL(MAX(v.created_at), '')
FROM votes v
INNER JOIN players p ON p.id = v.player_id
WHERE v.event_id = ?
GROUP BY v.player_id, p.first_name, p.last_name
ORDER BY votes DESC, MAX(v.created_at) DESC, v.player_id ASC
LIMIT 1
`, eventID)

	var mvp EventMVP
	var lastVoteRaw string
	if err := row.Scan(&mvp.PlayerID, &mvp.FirstName, &mvp.LastName, &mvp.Votes, &lastVoteRaw); err != nil {
		return EventMVP{}, err
	}

	if ts, err := parseSQLiteTimestamp(lastVoteRaw); err == nil && !ts.IsZero() {
		mvp.LastVoteAt = ts.UTC().Format(time.RFC3339)
	} else {
		mvp.LastVoteAt = strings.TrimSpace(lastVoteRaw)
	}

	return mvp, nil
}

// Team operations
func (db *appdbimpl) CreateTeam(name string) (int, error) {
	res, err := db.c.Exec(`INSERT INTO teams (name) VALUES (?)`, name)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	return int(id), nil
}

func (db *appdbimpl) ListTeams() ([]Team, error) {
	rows, err := db.c.Query(`SELECT id, name FROM teams`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ts []Team
	for rows.Next() {
		var t Team
		if err := rows.Scan(&t.ID, &t.Name); err != nil {
			return nil, err
		}
		ts = append(ts, t)
	}
	return ts, nil
}

func (db *appdbimpl) UpdateTeam(id int, name string) error {
	_, err := db.c.Exec(`UPDATE teams SET name=? WHERE id=?`, name, id)
	return err
}

func (db *appdbimpl) DeleteTeam(id int) error {
	_, err := db.c.Exec(`DELETE FROM teams WHERE id=?`, id)
	return err
}

// Player operations
func (db *appdbimpl) CreatePlayer(p Player) (int, error) {
	res, err := db.c.Exec(`INSERT INTO players (first_name, last_name, role, jersey_number, image_url, team_id) VALUES (?, ?, ?, ?, ?, ?)`, p.FirstName, p.LastName, p.Role, p.JerseyNumber, p.ImageURL, p.TeamID)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	return int(id), nil
}

func (db *appdbimpl) ListPlayers() ([]Player, error) {
	rows, err := db.c.Query(`SELECT id, first_name, last_name, role, jersey_number, image_url, team_id FROM players`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ps []Player
	for rows.Next() {
		var p Player
		if err := rows.Scan(&p.ID, &p.FirstName, &p.LastName, &p.Role, &p.JerseyNumber, &p.ImageURL, &p.TeamID); err != nil {
			return nil, err
		}
		ps = append(ps, p)
	}
	return ps, nil
}

func (db *appdbimpl) UpdatePlayer(p Player) error {
	_, err := db.c.Exec(`UPDATE players SET first_name=?, last_name=?, role=?, jersey_number=?, image_url=?, team_id=? WHERE id=?`, p.FirstName, p.LastName, p.Role, p.JerseyNumber, p.ImageURL, p.TeamID, p.ID)
	return err
}

func (db *appdbimpl) DeletePlayer(id int) error {
	_, err := db.c.Exec(`DELETE FROM players WHERE id=?`, id)
	return err
}

// Event operations
func sanitizePrizeInputs(prizes []EventPrize) []EventPrize {
	cleaned := make([]EventPrize, 0, len(prizes))
	for _, prize := range prizes {
		name := strings.TrimSpace(prize.Name)
		if name == "" {
			continue
		}
		cleaned = append(cleaned, EventPrize{
			ID:       prize.ID,
			EventID:  prize.EventID,
			Name:     name,
			Position: prize.Position,
		})
	}
	if len(cleaned) == 0 {
		return cleaned
	}
	sort.SliceStable(cleaned, func(i, j int) bool {
		if cleaned[i].Position == cleaned[j].Position {
			return i < j
		}
		if cleaned[i].Position <= 0 {
			return false
		}
		if cleaned[j].Position <= 0 {
			return true
		}
		return cleaned[i].Position < cleaned[j].Position
	})
	for idx := range cleaned {
		cleaned[idx].Position = idx + 1
	}
	return cleaned
}

func (db *appdbimpl) syncEventPrizesTx(tx *sql.Tx, eventID int, prizes []EventPrize) error {
	cleaned := sanitizePrizeInputs(prizes)

	if _, err := tx.Exec(`UPDATE event_prizes SET position = position + 1000 WHERE event_id = ?`, eventID); err != nil {
		return err
	}

	rows, err := tx.Query(`SELECT id, IFNULL(winner_vote_id, 0) FROM event_prizes WHERE event_id = ?`, eventID)
	if err != nil {
		return err
	}
	defer rows.Close()

	type existingPrize struct {
		id        int
		hasWinner bool
	}

	existing := make(map[int]existingPrize)
	for rows.Next() {
		var id int
		var winnerVoteID int
		if err := rows.Scan(&id, &winnerVoteID); err != nil {
			return err
		}
		existing[id] = existingPrize{id: id, hasWinner: winnerVoteID > 0}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	processed := make(map[int]struct{})

	for _, prize := range cleaned {
		if prize.ID > 0 {
			if _, ok := existing[prize.ID]; ok {
				if _, err := tx.Exec(`UPDATE event_prizes SET name = ?, position = ? WHERE id = ?`, prize.Name, prize.Position, prize.ID); err != nil {
					return err
				}
				processed[prize.ID] = struct{}{}
				continue
			}
		}
		if _, err := tx.Exec(`INSERT INTO event_prizes (event_id, name, position) VALUES (?, ?, ?)`, eventID, prize.Name, prize.Position); err != nil {
			return err
		}
	}

	for id, info := range existing {
		if _, ok := processed[id]; ok {
			continue
		}
		if info.hasWinner {
			return ErrPrizeLockedByWinner
		}
		if _, err := tx.Exec(`DELETE FROM event_prizes WHERE id = ?`, id); err != nil {
			return err
		}
	}

	return nil
}

func (db *appdbimpl) CreateEvent(e Event) (int, error) {
	tx, err := db.c.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	res, err := tx.Exec(`INSERT INTO events (team1_id, team2_id, start_datetime, location) VALUES (?, ?, ?, ?)`, e.Team1ID, e.Team2ID, e.StartDateTime, e.Location)
	if err != nil {
		return 0, err
	}
	id64, _ := res.LastInsertId()
	eventID := int(id64)

	if err := db.syncEventPrizesTx(tx, eventID, e.Prizes); err != nil {
		return 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return eventID, nil
}

func (db *appdbimpl) ListEvents() ([]Event, error) {
	rows, err := db.c.Query(`
SELECT e.id,
       e.team1_id,
       e.team2_id,
       e.start_datetime,
       e.location,
       e.is_active,
       e.votes_closed,
       e.is_concluded,
       IFNULL(t1.name, ''),
       IFNULL(t2.name, '')
FROM events e
LEFT JOIN teams t1 ON t1.id = e.team1_id
LEFT JOIN teams t2 ON t2.id = e.team2_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var es []Event
	for rows.Next() {
		var e Event
		var isActive int
		var votesClosed int
		var isConcluded int
		if err := rows.Scan(&e.ID, &e.Team1ID, &e.Team2ID, &e.StartDateTime, &e.Location, &isActive, &votesClosed, &isConcluded, &e.Team1Name, &e.Team2Name); err != nil {
			return nil, err
		}
		e.IsActive = isActive == 1
		e.VotesClosed = votesClosed == 1
		e.IsConcluded = isConcluded == 1
		es = append(es, e)
	}
	for i := range es {
		prizes, err := db.ListEventPrizes(es[i].ID)
		if err != nil {
			return nil, err
		}
		es[i].Prizes = prizes
	}

	return es, nil
}

func (db *appdbimpl) UpdateEvent(e Event) error {
	tx, err := db.c.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`UPDATE events SET team1_id=?, team2_id=?, start_datetime=?, location=? WHERE id=?`, e.Team1ID, e.Team2ID, e.StartDateTime, e.Location, e.ID); err != nil {
		return err
	}

	if err := db.syncEventPrizesTx(tx, e.ID, e.Prizes); err != nil {
		return err
	}

	return tx.Commit()
}

func (db *appdbimpl) DeleteEvent(id int) error {
	return db.PurgeEventData(id)
}

func (db *appdbimpl) PurgeEventData(eventID int) error {
	if eventID <= 0 {
		return sql.ErrNoRows
	}

	tx, err := db.c.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`UPDATE event_prizes SET winner_vote_id = NULL, winner_assigned_at = NULL WHERE event_id = ?`, eventID); err != nil {
		return err
	}

	if _, err := tx.Exec(`DELETE FROM sponsor_clicks WHERE event_id = ?`, eventID); err != nil {
		return err
	}

	if _, err := tx.Exec(`DELETE FROM votes WHERE event_id = ?`, eventID); err != nil {
		return err
	}

	if _, err := tx.Exec(`DELETE FROM tickets WHERE event_id = ?`, eventID); err != nil {
		return err
	}

	if _, err := tx.Exec(`DELETE FROM event_prizes WHERE event_id = ?`, eventID); err != nil {
		return err
	}

	res, err := tx.Exec(`DELETE FROM events WHERE id = ?`, eventID)
	if err != nil {
		return err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return sql.ErrNoRows
	}

	return tx.Commit()
}

func (db *appdbimpl) SetActiveEvent(eventID int) error {
	tx, err := db.c.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`UPDATE events SET is_active = 0`); err != nil {
		return err
	}

	res, err := tx.Exec(`UPDATE events SET is_active = 1, votes_closed = 0 WHERE id = ? AND is_concluded = 0`, eventID)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		var concluded int
		err := tx.QueryRow(`SELECT is_concluded FROM events WHERE id = ?`, eventID).Scan(&concluded)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return sql.ErrNoRows
			}
			return err
		}
		if concluded == 1 {
			return ErrEventAlreadyConcluded
		}
		return sql.ErrNoRows
	}

	return tx.Commit()
}

func (db *appdbimpl) ClearActiveEvent() error {
	_, err := db.c.Exec(`UPDATE events SET is_active = 0`)
	return err
}

func (db *appdbimpl) CloseEventVoting(eventID int) error {
	res, err := db.c.Exec(`UPDATE events SET votes_closed = 1 WHERE id = ? AND is_active = 1 AND is_concluded = 0`, eventID)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (db *appdbimpl) ConcludeEvent(eventID int) error {
	tx, err := db.c.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var concluded int
	if err := tx.QueryRow(`SELECT is_concluded FROM events WHERE id = ?`, eventID).Scan(&concluded); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return sql.ErrNoRows
		}
		return err
	}
	if concluded == 1 {
		return ErrEventAlreadyConcluded
	}

	if _, err := tx.Exec(`UPDATE events SET is_active = 0, votes_closed = 1, is_concluded = 1 WHERE id = ?`, eventID); err != nil {
		return err
	}

	return tx.Commit()
}

func (db *appdbimpl) GetActiveEvent() (Event, error) {
	var e Event
	var isActive int
	var votesClosed int
	var isConcluded int
	err := db.c.QueryRow(`
SELECT e.id,
       e.team1_id,
       e.team2_id,
       e.start_datetime,
       e.location,
       e.is_active,
       e.votes_closed,
       e.is_concluded,
       IFNULL(t1.name, ''),
       IFNULL(t2.name, '')
FROM events e
LEFT JOIN teams t1 ON t1.id = e.team1_id
LEFT JOIN teams t2 ON t2.id = e.team2_id
WHERE e.is_active = 1
LIMIT 1
`).Scan(&e.ID, &e.Team1ID, &e.Team2ID, &e.StartDateTime, &e.Location, &isActive, &votesClosed, &isConcluded, &e.Team1Name, &e.Team2Name)
	if err != nil {
		return Event{}, err
	}
	e.IsActive = isActive == 1
	e.VotesClosed = votesClosed == 1
	e.IsConcluded = isConcluded == 1
	return e, nil
}

// Votes listing and deletion
func (db *appdbimpl) ListVotes() ([]Vote, error) {
	rows, err := db.c.Query(`SELECT id, event_id, player_id, ticket_code, ticket_signature, device_id, created_at FROM votes`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var vs []Vote
	for rows.Next() {
		var v Vote
		if err := rows.Scan(&v.ID, &v.EventID, &v.PlayerID, &v.TicketCode, &v.TicketSignature, &v.DeviceID, &v.CreatedAt); err != nil {
			return nil, err
		}
		vs = append(vs, v)
	}
	return vs, nil
}

func (db *appdbimpl) ListEventTickets(eventID int) ([]EventTicket, error) {
	rows, err := db.c.Query(`
SELECT v.id, v.ticket_code, v.ticket_signature, v.player_id, IFNULL(p.first_name, ''), IFNULL(p.last_name, ''), v.created_at
FROM votes v
LEFT JOIN players p ON p.id = v.player_id
LEFT JOIN event_prizes ep ON ep.winner_vote_id = v.id AND ep.event_id = ?
WHERE v.event_id = ? AND ep.id IS NULL
ORDER BY v.created_at ASC
`, eventID, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tickets []EventTicket
	for rows.Next() {
		var t EventTicket
		if err := rows.Scan(&t.VoteID, &t.TicketCode, &t.TicketSignature, &t.PlayerID, &t.PlayerFirstName, &t.PlayerLastName, &t.CreatedAt); err != nil {
			return nil, err
		}
		tickets = append(tickets, t)
	}
	return tickets, nil
}

func (db *appdbimpl) ValidateTicket(eventID int, code string) (TicketValidationResult, error) {
	var result TicketValidationResult

	row := db.c.QueryRow(`
SELECT v.id,
       v.event_id,
       v.player_id,
       v.ticket_code,
       v.ticket_signature,
       IFNULL(p.first_name, ''),
       IFNULL(p.last_name, ''),
       v.created_at,
       ep.id,
       IFNULL(ep.name, ''),
       IFNULL(ep.position, 0)
FROM votes v
LEFT JOIN players p ON p.id = v.player_id
LEFT JOIN event_prizes ep ON ep.winner_vote_id = v.id AND ep.event_id = v.event_id
WHERE v.event_id = ? AND v.ticket_code = ?
LIMIT 1
`, eventID, code)

	var prizeID sql.NullInt64
	var prizeName sql.NullString
	var prizePosition sql.NullInt64

	if err := row.Scan(
		&result.VoteID,
		&result.EventID,
		&result.PlayerID,
		&result.TicketCode,
		&result.TicketSignature,
		&result.PlayerFirstName,
		&result.PlayerLastName,
		&result.CreatedAt,
		&prizeID,
		&prizeName,
		&prizePosition,
	); err != nil {
		return TicketValidationResult{}, err
	}

	if prizeID.Valid {
		result.AssignedPrize = &TicketValidationPrize{
			ID:       int(prizeID.Int64),
			Name:     prizeName.String,
			Position: int(prizePosition.Int64),
		}
	}

	return result, nil
}

func (db *appdbimpl) RedeemTicket(eventID int, code, signature string) (bool, error) {
	var storedSignature sql.NullString
	var redeemedAt sql.NullString

	err := db.c.QueryRow(`SELECT signature, redeemed_at FROM tickets WHERE event_id = ? AND code = ? LIMIT 1`, eventID, code).Scan(&storedSignature, &redeemedAt)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		_, err = db.c.Exec(`INSERT INTO tickets (event_id, code, signature, redeemed_at) VALUES (?, ?, ?, CURRENT_TIMESTAMP)`, eventID, code, signature)
		if err != nil {
			if isTicketUniqueConstraintError(err) {
				return db.RedeemTicket(eventID, code, signature)
			}
			return false, err
		}
		return false, nil
	case err != nil:
		return false, err
	}

	if storedSignature.Valid && storedSignature.String != "" && !strings.EqualFold(storedSignature.String, signature) {
		return true, ErrTicketSignatureMismatch
	}

	if !redeemedAt.Valid || strings.TrimSpace(redeemedAt.String) == "" {
		_, err = db.c.Exec(`UPDATE tickets SET signature = ?, redeemed_at = CURRENT_TIMESTAMP WHERE event_id = ? AND code = ?`, signature, eventID, code)
		if err != nil {
			return false, err
		}
		return false, nil
	}

	return true, nil
}

func isTicketUniqueConstraintError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed")
}

func (db *appdbimpl) ListEventPrizes(eventID int) ([]EventPrize, error) {
	rows, err := db.c.Query(`
SELECT p.id,
       p.event_id,
       p.name,
        p.position,
       p.winner_vote_id,
       IFNULL(p.winner_assigned_at, ''),
       IFNULL(v.ticket_code, ''),
       IFNULL(v.player_id, 0),
       IFNULL(pl.first_name, ''),
       IFNULL(pl.last_name, '')
FROM event_prizes p
LEFT JOIN votes v ON v.id = p.winner_vote_id
LEFT JOIN players pl ON pl.id = v.player_id
WHERE p.event_id = ?
ORDER BY p.position ASC, p.id ASC
`, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prizes []EventPrize
	for rows.Next() {
		var p EventPrize
		var winnerID sql.NullInt64
		var assignedAt sql.NullString
		var ticketCode string
		var playerID int
		var playerFirstName string
		var playerLastName string
		if err := rows.Scan(&p.ID, &p.EventID, &p.Name, &p.Position, &winnerID, &assignedAt, &ticketCode, &playerID, &playerFirstName, &playerLastName); err != nil {
			return nil, err
		}
		if winnerID.Valid {
			p.Winner = &EventPrizeWinner{
				VoteID:          int(winnerID.Int64),
				TicketCode:      ticketCode,
				PlayerID:        playerID,
				PlayerFirstName: playerFirstName,
				PlayerLastName:  playerLastName,
				AssignedAt:      assignedAt.String,
			}
		}
		prizes = append(prizes, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return prizes, nil
}

func (db *appdbimpl) getEventPrize(prizeID int) (EventPrize, error) {
	var p EventPrize
	var winnerID sql.NullInt64
	var assignedAt sql.NullString
	var ticketCode string
	var playerID int
	var playerFirstName string
	var playerLastName string

	err := db.c.QueryRow(`
SELECT p.id,
       p.event_id,
       p.name,
       p.position,
       p.winner_vote_id,
       IFNULL(p.winner_assigned_at, ''),
       IFNULL(v.ticket_code, ''),
       IFNULL(v.player_id, 0),
       IFNULL(pl.first_name, ''),
       IFNULL(pl.last_name, '')
FROM event_prizes p
LEFT JOIN votes v ON v.id = p.winner_vote_id
LEFT JOIN players pl ON pl.id = v.player_id
WHERE p.id = ?
`, prizeID).Scan(&p.ID, &p.EventID, &p.Name, &p.Position, &winnerID, &assignedAt, &ticketCode, &playerID, &playerFirstName, &playerLastName)
	if err != nil {
		return EventPrize{}, err
	}
	if winnerID.Valid {
		p.Winner = &EventPrizeWinner{
			VoteID:          int(winnerID.Int64),
			TicketCode:      ticketCode,
			PlayerID:        playerID,
			PlayerFirstName: playerFirstName,
			PlayerLastName:  playerLastName,
			AssignedAt:      assignedAt.String,
		}
	}
	return p, nil
}

func (db *appdbimpl) AssignPrizeWinner(eventID, prizeID, voteID int) (EventPrize, error) {
	tx, err := db.c.Begin()
	if err != nil {
		return EventPrize{}, err
	}
	defer tx.Rollback()

	var prizeEventID int
	var winnerID sql.NullInt64
	if err := tx.QueryRow(`SELECT event_id, winner_vote_id FROM event_prizes WHERE id = ?`, prizeID).Scan(&prizeEventID, &winnerID); err != nil {
		return EventPrize{}, err
	}
	if prizeEventID != eventID {
		return EventPrize{}, sql.ErrNoRows
	}
	if winnerID.Valid && winnerID.Int64 > 0 {
		return EventPrize{}, ErrPrizeAlreadyAssigned
	}

	var voteEventID int
	if err := tx.QueryRow(`SELECT event_id FROM votes WHERE id = ?`, voteID).Scan(&voteEventID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return EventPrize{}, ErrPrizeVoteMismatch
		}
		return EventPrize{}, err
	}
	if voteEventID != eventID {
		return EventPrize{}, ErrPrizeVoteMismatch
	}

	var alreadyAssigned int
	if err := tx.QueryRow(`SELECT COUNT(1) FROM event_prizes WHERE event_id = ? AND winner_vote_id = ?`, eventID, voteID).Scan(&alreadyAssigned); err != nil {
		return EventPrize{}, err
	}
	if alreadyAssigned > 0 {
		return EventPrize{}, ErrPrizeWinnerConflict
	}

	if _, err := tx.Exec(`UPDATE event_prizes SET winner_vote_id = ?, winner_assigned_at = CURRENT_TIMESTAMP WHERE id = ?`, voteID, prizeID); err != nil {
		return EventPrize{}, err
	}

	if err := tx.Commit(); err != nil {
		return EventPrize{}, err
	}

	return db.getEventPrize(prizeID)
}

func (db *appdbimpl) ClearPrizeWinner(eventID, prizeID int) error {
	res, err := db.c.Exec(`UPDATE event_prizes SET winner_vote_id = NULL, winner_assigned_at = NULL WHERE id = ? AND event_id = ?`, prizeID, eventID)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// GetEventResults returns aggregated vote results for an event
func (db *appdbimpl) GetEventResults(eventID int) ([]EventVoteResult, error) {
	var exists int
	if err := db.c.QueryRow(`SELECT COUNT(1) FROM events WHERE id = ?`, eventID).Scan(&exists); err != nil {
		return nil, err
	}
	if exists == 0 {
		return nil, sql.ErrNoRows
	}

	rows, err := db.c.Query(`
SELECT player_id, COUNT(*) AS votes, IFNULL(MAX(created_at), '') AS last_vote_at
FROM votes
WHERE event_id = ?
GROUP BY player_id
ORDER BY votes DESC, last_vote_at ASC, player_id ASC
`, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []EventVoteResult
	for rows.Next() {
		var res EventVoteResult
		if err := rows.Scan(&res.PlayerID, &res.Votes, &res.LastVoteAt); err != nil {
			return nil, err
		}
		results = append(results, res)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

func (db *appdbimpl) DeleteVote(id int) error {
	_, err := db.c.Exec(`DELETE FROM votes WHERE id=?`, id)
	return err
}

// Admin operations
func (db *appdbimpl) CreateAdmin(a Admin) (int, error) {
	res, err := db.c.Exec(`INSERT INTO admins (username, password_hash, role) VALUES (?, ?, ?)`, a.Username, a.PasswordHash, a.Role)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	return int(id), nil
}

func (db *appdbimpl) ListAdmins() ([]Admin, error) {
	rows, err := db.c.Query(`SELECT id, username, password_hash, role, created_at FROM admins`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var as []Admin
	for rows.Next() {
		var a Admin
		if err := rows.Scan(&a.ID, &a.Username, &a.PasswordHash, &a.Role, &a.CreatedAt); err != nil {
			return nil, err
		}
		as = append(as, a)
	}
	return as, nil
}

func (db *appdbimpl) UpdateAdmin(a Admin) error {
	_, err := db.c.Exec(`UPDATE admins SET username=?, password_hash=?, role=? WHERE id=?`, a.Username, a.PasswordHash, a.Role, a.ID)
	return err
}

func (db *appdbimpl) DeleteAdmin(id int) error {
	_, err := db.c.Exec(`DELETE FROM admins WHERE id=?`, id)
	return err
}

func (db *appdbimpl) GetAdminByUsername(username string) (Admin, error) {
	var admin Admin
	err := db.c.QueryRow(`SELECT id, username, password_hash, role, created_at FROM admins WHERE username = ?`, username).Scan(&admin.ID, &admin.Username, &admin.PasswordHash, &admin.Role, &admin.CreatedAt)
	if err != nil {
		return Admin{}, err
	}
	return admin, nil
}

func (db *appdbimpl) GetAdminByID(id int) (Admin, error) {
	var admin Admin
	err := db.c.QueryRow(`SELECT id, username, password_hash, role, created_at FROM admins WHERE id = ?`, id).Scan(&admin.ID, &admin.Username, &admin.PasswordHash, &admin.Role, &admin.CreatedAt)
	if err != nil {
		return Admin{}, err
	}
	return admin, nil
}

// Sponsor operations
func (db *appdbimpl) CreateSponsor(s Sponsor) (int, error) {
	sanitizedName := strings.TrimSpace(s.Name)
	if strings.TrimSpace(s.LogoData) == "" {
		return 0, ErrInvalidSponsorData
	}

	var total int
	if err := db.c.QueryRow(`SELECT COUNT(*) FROM sponsors`).Scan(&total); err != nil {
		return 0, err
	}
	if total >= maxSponsorSlots {
		return 0, ErrMaxSponsors
	}

	position := s.Position
	if position <= 0 || position > maxSponsorSlots {
		nextPos, err := db.nextSponsorPosition()
		if err != nil {
			return 0, err
		}
		position = nextPos
	}

	sanitizedLink := strings.TrimSpace(s.LinkURL)
	isActive := s.IsActive
	if position > total+1 {
		position = total + 1
	}

	res, err := db.c.Exec(`INSERT INTO sponsors (position, name, logo_data, link_url, is_active) VALUES (?, ?, ?, ?, ?)`, position, sanitizedName, s.LogoData, sanitizedLink, boolToInt(isActive))
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed: sponsors.position") {
			return 0, ErrInvalidSponsorPos
		}
		return 0, err
	}
	id, _ := res.LastInsertId()
	return int(id), nil
}

func (db *appdbimpl) UpdateSponsor(s Sponsor) error {
	if s.ID <= 0 {
		return sql.ErrNoRows
	}

	sanitizedName := strings.TrimSpace(s.Name)
	if strings.TrimSpace(s.LogoData) == "" {
		return ErrInvalidSponsorData
	}

	if s.Position <= 0 || s.Position > maxSponsorSlots {
		return ErrInvalidSponsorPos
	}

	sanitizedLink := strings.TrimSpace(s.LinkURL)

	res, err := db.c.Exec(`UPDATE sponsors SET position=?, name=?, logo_data=?, link_url=?, is_active=? WHERE id=?`, s.Position, sanitizedName, s.LogoData, sanitizedLink, boolToInt(s.IsActive), s.ID)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed: sponsors.position") {
			return ErrInvalidSponsorPos
		}
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (db *appdbimpl) DeleteSponsor(id int) error {
	res, err := db.c.Exec(`DELETE FROM sponsors WHERE id=?`, id)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return sql.ErrNoRows
	}
	return db.normalizeSponsorPositions()
}

func (db *appdbimpl) ListSponsors() ([]Sponsor, error) {
	return db.querySponsors(false)
}

func (db *appdbimpl) ListActiveSponsors() ([]Sponsor, error) {
	return db.querySponsors(true)
}

func (db *appdbimpl) RecordSponsorClick(eventID, sponsorID int) error {
	if eventID <= 0 || sponsorID <= 0 {
		return sql.ErrNoRows
	}
	_, err := db.c.Exec(`INSERT INTO sponsor_clicks (event_id, sponsor_id) VALUES (?, ?)`, eventID, sponsorID)
	return err
}

func (db *appdbimpl) GetSponsorClickStats(eventID int) ([]SponsorClickStat, error) {
	rows, err := db.c.Query(`
SELECT s.id,
       IFNULL(s.name, ''),
       IFNULL(s.link_url, ''),
       COUNT(c.id) AS clicks,
       IFNULL(s.position, 0)
FROM sponsor_clicks c
INNER JOIN sponsors s ON s.id = c.sponsor_id
WHERE c.event_id = ?
GROUP BY s.id, s.name, s.link_url, s.position
ORDER BY s.position ASC, s.id ASC
`, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []SponsorClickStat
	for rows.Next() {
		var stat SponsorClickStat
		var position int
		if err := rows.Scan(&stat.SponsorID, &stat.Name, &stat.LinkURL, &stat.Clicks, &position); err != nil {
			return nil, err
		}
		stats = append(stats, stat)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return stats, nil
}

func (db *appdbimpl) querySponsors(activeOnly bool) ([]Sponsor, error) {
	baseQuery := `SELECT id, position, name, logo_data, IFNULL(link_url, ''), is_active FROM sponsors`
	if activeOnly {
		baseQuery += ` WHERE is_active = 1`
	}
	baseQuery += ` ORDER BY position ASC, id ASC`

	rows, err := db.c.Query(baseQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sponsors []Sponsor
	for rows.Next() {
		var s Sponsor
		var isActive int
		if err := rows.Scan(&s.ID, &s.Position, &s.Name, &s.LogoData, &s.LinkURL, &isActive); err != nil {
			return nil, err
		}
		s.IsActive = isActive == 1
		sponsors = append(sponsors, s)
	}
	return sponsors, nil
}

func (db *appdbimpl) nextSponsorPosition() (int, error) {
	rows, err := db.c.Query(`SELECT position FROM sponsors ORDER BY position ASC`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	used := make(map[int]struct{})
	for rows.Next() {
		var pos int
		if err := rows.Scan(&pos); err != nil {
			return 0, err
		}
		used[pos] = struct{}{}
	}

	for i := 1; i <= maxSponsorSlots; i++ {
		if _, ok := used[i]; !ok {
			return i, nil
		}
	}

	return 0, ErrMaxSponsors
}

func (db *appdbimpl) normalizeSponsorPositions() error {
	rows, err := db.c.Query(`SELECT id FROM sponsors ORDER BY position ASC, id ASC`)
	if err != nil {
		return err
	}
	defer rows.Close()

	position := 1
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return err
		}
		if _, err := db.c.Exec(`UPDATE sponsors SET position=? WHERE id=?`, position, id); err != nil {
			return err
		}
		position++
	}
	return nil
}

func parseSQLiteTimestamp(value string) (time.Time, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return time.Time{}, fmt.Errorf("empty timestamp")
	}

	candidates := []string{trimmed}
	if !strings.Contains(trimmed, "T") {
		candidates = append(candidates, strings.Replace(trimmed, " ", "T", 1))
	}

	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05.000000000",
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02T15:04:05.000000000",
		"2006-01-02T15:04:05",
		"2006-01-02T15:04",
	}

	for _, candidate := range candidates {
		for _, layout := range layouts {
			if ts, err := time.ParseInLocation(layout, candidate, time.UTC); err == nil {
				return ts, nil
			}
		}
	}

	return time.Time{}, fmt.Errorf("unsupported timestamp format: %s", value)
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
