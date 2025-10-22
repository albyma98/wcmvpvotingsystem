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
	"strings"
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
	ID            int    `json:"id"`
	Team1ID       int    `json:"team1_id"`
	Team1Name     string `json:"team1_name,omitempty"`
	Team2ID       int    `json:"team2_id"`
	Team2Name     string `json:"team2_name,omitempty"`
	StartDateTime string `json:"start_datetime"`
	Location      string `json:"location"`
	IsActive      bool   `json:"is_active"`
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

type TicketScan struct {
	ID              int    `json:"id"`
	VoteID          int    `json:"vote_id"`
	TicketCode      string `json:"ticket_code"`
	TicketSignature string `json:"ticket_signature"`
	ScanCount       int    `json:"scan_count"`
	FirstScannedAt  string `json:"first_scanned_at"`
	LastScannedAt   string `json:"last_scanned_at"`
}

type EventTicket struct {
	VoteID          int    `json:"vote_id"`
	TicketCode      string `json:"ticket_code"`
	PlayerID        int    `json:"player_id"`
	PlayerFirstName string `json:"player_first_name"`
	PlayerLastName  string `json:"player_last_name"`
	CreatedAt       string `json:"created_at"`
}

type Admin struct {
	ID           int    `json:"id"`
	Username     string `json:"username"`
	PasswordHash string `json:"password_hash"`
	Role         string `json:"role"`
	CreatedAt    string `json:"created_at"`
}

// AppDatabase is the high level interface for the DB
type AppDatabase interface {
	GetName() (string, error)
	SetName(name string) error
	AddVote(eventID, playerID int, code, signature, deviceID string) error
	GetVoteByCodeAndSignature(code, signature string) (Vote, error)
	RecordTicketScan(voteID int, code, signature string) (bool, error)
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
	GetActiveEvent() (Event, error)
	ListVotes() ([]Vote, error)
	ListEventTickets(eventID int) ([]EventTicket, error)
	DeleteVote(id int) error
	CreateAdmin(a Admin) (int, error)
	ListAdmins() ([]Admin, error)
	UpdateAdmin(a Admin) error
	DeleteAdmin(id int) error
	GetAdminByUsername(username string) (Admin, error)
	Ping() error
}

type appdbimpl struct {
	c *sql.DB
}

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

	if _, err = db.Exec(`ALTER TABLE events ADD COLUMN is_active INTEGER NOT NULL DEFAULT 0`); err != nil {
		if !strings.Contains(err.Error(), "duplicate column name") {
			return nil, fmt.Errorf("error ensuring events active column: %w", err)
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

	// Create ticket_scans table if not exists
	err = db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='ticket_scans';`).Scan(&tableName)
	if errors.Is(err, sql.ErrNoRows) {
		sqlStmt := `CREATE TABLE ticket_scans (id INTEGER PRIMARY KEY AUTOINCREMENT, vote_id INTEGER NOT NULL UNIQUE, ticket_code TEXT NOT NULL, ticket_signature TEXT NOT NULL, scan_count INTEGER NOT NULL DEFAULT 0, first_scanned_at TEXT DEFAULT CURRENT_TIMESTAMP, last_scanned_at TEXT DEFAULT CURRENT_TIMESTAMP, FOREIGN KEY (vote_id) REFERENCES votes(id) ON DELETE CASCADE);`
		_, err = db.Exec(sqlStmt)
		if err != nil {
			return nil, fmt.Errorf("error creating ticket_scans table: %w", err)
		}
		_, err = db.Exec(`CREATE UNIQUE INDEX idx_ticket_scans_code ON ticket_scans(ticket_code);`)
		if err != nil {
			return nil, fmt.Errorf("error creating ticket_scans code index: %w", err)
		}
	}
	_, err = db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_ticket_scans_code ON ticket_scans(ticket_code);`)
	if err != nil {
		return nil, fmt.Errorf("error ensuring ticket_scans code index: %w", err)
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

// GetVoteByCodeAndSignature returns the vote matching the provided code and signature
func (db *appdbimpl) GetVoteByCodeAndSignature(code, signature string) (Vote, error) {
	var v Vote
	err := db.c.QueryRow(`SELECT id, event_id, player_id, ticket_code, ticket_signature, device_id, created_at FROM votes WHERE ticket_code = ? AND ticket_signature = ?`, code, signature).
		Scan(&v.ID, &v.EventID, &v.PlayerID, &v.TicketCode, &v.TicketSignature, &v.DeviceID, &v.CreatedAt)
	if err != nil {
		return Vote{}, err
	}
	return v, nil
}

// RecordTicketScan increments scan count for a vote and reports whether it was previously scanned
func (db *appdbimpl) RecordTicketScan(voteID int, code, signature string) (bool, error) {
	tx, err := db.c.Begin()
	if err != nil {
		return false, err
	}
	defer tx.Rollback()

	var scanCount int
	err = tx.QueryRow(`SELECT scan_count FROM ticket_scans WHERE vote_id = ?`, voteID).Scan(&scanCount)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		_, err = tx.Exec(`INSERT INTO ticket_scans (vote_id, ticket_code, ticket_signature, scan_count) VALUES (?, ?, ?, 1)`, voteID, code, signature)
		if err != nil {
			return false, err
		}
		if err := tx.Commit(); err != nil {
			return false, err
		}
		return false, nil
	case err != nil:
		return false, err
	default:
		_, err = tx.Exec(`UPDATE ticket_scans SET scan_count = scan_count + 1, last_scanned_at = CURRENT_TIMESTAMP WHERE vote_id = ?`, voteID)
		if err != nil {
			return false, err
		}
		if err := tx.Commit(); err != nil {
			return false, err
		}
		return true, nil
	}
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
func (db *appdbimpl) CreateEvent(e Event) (int, error) {
	res, err := db.c.Exec(`INSERT INTO events (team1_id, team2_id, start_datetime, location) VALUES (?, ?, ?, ?)`, e.Team1ID, e.Team2ID, e.StartDateTime, e.Location)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	return int(id), nil
}

func (db *appdbimpl) ListEvents() ([]Event, error) {
	rows, err := db.c.Query(`SELECT id, team1_id, team2_id, start_datetime, location, is_active FROM events`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var es []Event
	for rows.Next() {
		var e Event
		var isActive int
		if err := rows.Scan(&e.ID, &e.Team1ID, &e.Team2ID, &e.StartDateTime, &e.Location, &isActive); err != nil {
			return nil, err
		}
		e.IsActive = isActive == 1
		es = append(es, e)
	}
	return es, nil
}

func (db *appdbimpl) UpdateEvent(e Event) error {
	_, err := db.c.Exec(`UPDATE events SET team1_id=?, team2_id=?, start_datetime=?, location=? WHERE id=?`, e.Team1ID, e.Team2ID, e.StartDateTime, e.Location, e.ID)
	return err
}

func (db *appdbimpl) DeleteEvent(id int) error {
	_, err := db.c.Exec(`DELETE FROM events WHERE id=?`, id)
	return err
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

	res, err := tx.Exec(`UPDATE events SET is_active = 1 WHERE id = ?`, eventID)
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

func (db *appdbimpl) ClearActiveEvent() error {
	_, err := db.c.Exec(`UPDATE events SET is_active = 0`)
	return err
}

func (db *appdbimpl) GetActiveEvent() (Event, error) {
	var e Event
	var isActive int
	err := db.c.QueryRow(`
SELECT e.id, e.team1_id, IFNULL(t1.name, ''), e.team2_id, IFNULL(t2.name, ''), e.start_datetime, e.location, e.is_active
FROM events e
LEFT JOIN teams t1 ON t1.id = e.team1_id
LEFT JOIN teams t2 ON t2.id = e.team2_id
WHERE e.is_active = 1
LIMIT 1
`).Scan(&e.ID, &e.Team1ID, &e.Team1Name, &e.Team2ID, &e.Team2Name, &e.StartDateTime, &e.Location, &isActive)
	if err != nil {
		return Event{}, err
	}
	e.IsActive = isActive == 1
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
SELECT v.id, v.ticket_code, v.player_id, IFNULL(p.first_name, ''), IFNULL(p.last_name, ''), v.created_at
FROM votes v
LEFT JOIN players p ON p.id = v.player_id
WHERE v.event_id = ?
ORDER BY v.created_at ASC
`, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tickets []EventTicket
	for rows.Next() {
		var t EventTicket
		if err := rows.Scan(&t.VoteID, &t.TicketCode, &t.PlayerID, &t.PlayerFirstName, &t.PlayerLastName, &t.CreatedAt); err != nil {
			return nil, err
		}
		tickets = append(tickets, t)
	}
	return tickets, nil
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
