package api

// ============================================================================
// TOURNAMENT ADMIN — layer dati.
// Mondo PARALLELO alle società: tabelle dedicate (tournament_admins,
// tournament_teams, tournament_sponsors) che non toccano organizations,
// admins, teams o sponsors del flusso club. L'unico punto di contatto è
// events (type='tournament'), già esteso, invisibile alle query club che
// filtrano per organization_id.
// ============================================================================

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

var errTANotFound = errors.New("not found")

// EnsureTournamentAdminTables crea le tabelle del mondo torneo (idempotente).
// Chiamata al mount delle route: nessun tocco a database.go.
func (s *Store) EnsureTournamentAdminTables() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS tournament_admins (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			event_id INTEGER NOT NULL,
			username TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			created_at TEXT NOT NULL DEFAULT (datetime('now'))
		);`,
		`CREATE TABLE IF NOT EXISTS tournament_admin_sessions (
			token TEXT PRIMARY KEY,
			admin_id INTEGER NOT NULL,
			event_id INTEGER NOT NULL,
			expires_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS tournament_teams (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			event_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			short_name TEXT NOT NULL DEFAULT '',
			city TEXT NOT NULL DEFAULT '',
			logo_url TEXT NOT NULL DEFAULT '',
			group_name TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL DEFAULT (datetime('now'))
		);`,
		`CREATE TABLE IF NOT EXISTS tournament_sponsors (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			event_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			logo_url TEXT NOT NULL DEFAULT '',
			url TEXT NOT NULL DEFAULT '',
			tier TEXT NOT NULL DEFAULT 'partner',
			brand_color TEXT NOT NULL DEFAULT '',
			position INTEGER NOT NULL DEFAULT 0,
			active INTEGER NOT NULL DEFAULT 1
		);`,
	}
	for _, q := range stmts {
		if _, err := s.db.Exec(q); err != nil {
			return fmt.Errorf("tournament admin tables: %w", err)
		}
	}
	// Punteggio del set corrente (il resto di matches esiste già).
	// ALTER idempotente: SQLite non ha IF NOT EXISTS su ADD COLUMN.
	for _, alter := range []string{
		`ALTER TABLE matches ADD COLUMN cur_a INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE matches ADD COLUMN cur_b INTEGER NOT NULL DEFAULT 0`,
	} {
		if _, err := s.db.Exec(alter); err != nil && !strings.Contains(err.Error(), "duplicate column") {
			return fmt.Errorf("matches alter: %w", err)
		}
	}
	return nil
}

// Delete invalida una chiave della cache live (dopo ogni scrittura di score
// il polling dei tifosi deve vedere il dato fresco, non il TTL residuo).
func (c *TTLCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.data, key)
}

// --- Creazione torneo (dal master) ------------------------------------------

type TournamentCreateInput struct {
	Name      string `json:"name"`
	Format    string `json:"format"`
	DateLabel string `json:"dateLabel"`
	Location  string `json:"location"`
	Slug      string `json:"slug"`
}

type TournamentSummary struct {
	EventID       int64  `json:"eventId"`
	Slug          string `json:"slug"`
	Name          string `json:"name"`
	Format        string `json:"format"`
	DateLabel     string `json:"dateLabel"`
	Location      string `json:"location"`
	StatusLabel   string `json:"statusLabel"`
	PhaseLabel    string `json:"phaseLabel"`
	AdminUsername string `json:"adminUsername"`
	TeamsCount    int    `json:"teamsCount"`
	MatchesCount  int    `json:"matchesCount"`
}

var defaultTournamentTiles = []Tile{
	{ID: "calendar", Icon: "calendar", Label: "Calendario", Sub: "Tutte le partite", Color: "#35357F", Route: "/calendar"},
	{ID: "standings", Icon: "chart", Label: "Classifiche", Sub: "Gironi e ranking", Color: "#5B2333", Route: "/standings"},
	{ID: "bracket", Icon: "bracket", Label: "Tabellone", Sub: "Fase finale", Color: "#0E5F4C", Route: "/bracket"},
	{ID: "mvp", Icon: "star", Label: "Vota MVP", Sub: "Vota il migliore", Color: "#A8730F", Route: "/mvp"},
	{ID: "prizes", Icon: "trophy", Label: "Premi", Sub: "Cosa si vince", Color: "#6B5A12", Route: "/prizes"},
	{ID: "gallery", Icon: "gallery", Label: "Gallery", Sub: "Foto del torneo", Color: "#8F2B44", Route: "/gallery"},
	{ID: "rules", Icon: "doc", Label: "Regolamento", Sub: "Info e regole", Color: "#3A4A63", Route: "/rules"},
	{ID: "event", Icon: "info", Label: "Info Evento", Sub: "Mappa e servizi", Color: "#5B2E86", Route: "/event"},
}

// CreateTournament crea l'evento contenitore + tiles default + admin dedicato.
// Ritorna l'ID evento. passwordHash è già bcrypt-ato dal chiamante.
func (s *Store) CreateTournament(ctx context.Context, in TournamentCreateInput, adminUsername, passwordHash string) (int64, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()

	// events: le colonne club NOT NULL (team1/2, start_datetime) vengono
	// riempite con valori neutri — il mondo club le ignora (type filtra).
	res, err := tx.ExecContext(ctx, `
		INSERT INTO events (organization_id, team1_id, team2_id, start_datetime, location,
		                    slug, name, format, date_label, status_label, phase_label, type)
		VALUES (0, 0, 0, '', ?, ?, ?, ?, ?, 'TORNEO IN ARRIVO', 'ISCRIZIONI', 'tournament')`,
		in.Location, in.Slug, in.Name, in.Format, in.DateLabel)
	if err != nil {
		return 0, err
	}
	eventID, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	for i, t := range defaultTournamentTiles {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO event_tiles (id, event_id, icon, label, sub, color, route, position, enabled)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, 1)`,
			fmt.Sprintf("%d-%s", eventID, t.ID), eventID, t.Icon, t.Label, t.Sub, t.Color, t.Route, i); err != nil {
			return 0, err
		}
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO tournament_admins (event_id, username, password_hash) VALUES (?, ?, ?)`,
		eventID, adminUsername, passwordHash); err != nil {
		return 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return eventID, nil
}

func (s *Store) SlugExists(ctx context.Context, slug string) (bool, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM events WHERE slug = ?`, slug).Scan(&n)
	return n > 0, err
}

func (s *Store) ListTournaments(ctx context.Context) ([]TournamentSummary, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT e.id, e.slug, e.name, COALESCE(e.format,''), COALESCE(e.date_label,''),
		       COALESCE(e.location,''), COALESCE(e.status_label,''), COALESCE(e.phase_label,''),
		       COALESCE(a.username,''),
		       (SELECT COUNT(1) FROM tournament_teams tt WHERE tt.event_id = e.id),
		       (SELECT COUNT(1) FROM matches m WHERE m.event_id = e.id)
		FROM events e
		LEFT JOIN tournament_admins a ON a.event_id = e.id
		WHERE e.type = 'tournament'
		ORDER BY e.id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]TournamentSummary, 0, 8)
	for rows.Next() {
		var t TournamentSummary
		if err := rows.Scan(&t.EventID, &t.Slug, &t.Name, &t.Format, &t.DateLabel,
			&t.Location, &t.StatusLabel, &t.PhaseLabel, &t.AdminUsername,
			&t.TeamsCount, &t.MatchesCount); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// --- Auth admin torneo --------------------------------------------------------

type taAdmin struct {
	ID           int64
	EventID      int64
	Username     string
	PasswordHash string
}

func (s *Store) GetTAAdminByUsername(ctx context.Context, username string) (*taAdmin, error) {
	var a taAdmin
	err := s.db.QueryRowContext(ctx, `
		SELECT id, event_id, username, password_hash FROM tournament_admins WHERE username = ?`,
		username).Scan(&a.ID, &a.EventID, &a.Username, &a.PasswordHash)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errTANotFound
	}
	return &a, err
}

func (s *Store) EventIDBySlug(ctx context.Context, slug string) (int64, error) {
	var id int64
	err := s.db.QueryRowContext(ctx, `
		SELECT id FROM events WHERE slug = ? AND type = 'tournament'`, slug).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, errTANotFound
	}
	return id, err
}

func (s *Store) CreateTASession(ctx context.Context, adminID, eventID int64, ttl time.Duration) (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	token := base64.RawURLEncoding.EncodeToString(buf)
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO tournament_admin_sessions (token, admin_id, event_id, expires_at)
		VALUES (?, ?, ?, ?)`,
		token, adminID, eventID, time.Now().Add(ttl).UTC().Format(time.RFC3339))
	return token, err
}

func (s *Store) GetTASession(ctx context.Context, token string) (adminID, eventID int64, ok bool) {
	var expires string
	err := s.db.QueryRowContext(ctx, `
		SELECT admin_id, event_id, expires_at FROM tournament_admin_sessions WHERE token = ?`,
		token).Scan(&adminID, &eventID, &expires)
	if err != nil {
		return 0, 0, false
	}
	exp, err := time.Parse(time.RFC3339, expires)
	if err != nil || time.Now().After(exp) {
		_, _ = s.db.ExecContext(ctx, `DELETE FROM tournament_admin_sessions WHERE token = ?`, token)
		return 0, 0, false
	}
	return adminID, eventID, true
}

func (s *Store) DeleteTASession(ctx context.Context, token string) {
	_, _ = s.db.ExecContext(ctx, `DELETE FROM tournament_admin_sessions WHERE token = ?`, token)
}

// --- Squadre -------------------------------------------------------------------

type TATeam struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	ShortName string `json:"shortName"`
	City      string `json:"city"`
	GroupName string `json:"groupName"`
}

func (s *Store) ListTATeams(ctx context.Context, eventID int64) ([]TATeam, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, short_name, city, group_name
		FROM tournament_teams WHERE event_id = ? ORDER BY group_name, name COLLATE NOCASE`, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]TATeam, 0, 16)
	for rows.Next() {
		var t TATeam
		if err := rows.Scan(&t.ID, &t.Name, &t.ShortName, &t.City, &t.GroupName); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *Store) InsertTATeams(ctx context.Context, eventID int64, teams []TATeam) (int, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()
	n := 0
	for _, t := range teams {
		name := strings.TrimSpace(t.Name)
		if name == "" {
			continue
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO tournament_teams (event_id, name, short_name, city, group_name)
			VALUES (?, ?, ?, ?, ?)`,
			eventID, name, strings.TrimSpace(t.ShortName), strings.TrimSpace(t.City), strings.TrimSpace(t.GroupName)); err != nil {
			return 0, err
		}
		n++
	}
	return n, tx.Commit()
}

func (s *Store) DeleteTATeam(ctx context.Context, eventID, teamID int64) error {
	// Rifiuta se la squadra è usata in una partita: integrità prima di tutto.
	var used int
	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(1) FROM matches WHERE event_id = ? AND (team_a_id = ? OR team_b_id = ?)`,
		eventID, teamID, teamID).Scan(&used); err != nil {
		return err
	}
	if used > 0 {
		return fmt.Errorf("team_in_use")
	}
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM tournament_teams WHERE id = ? AND event_id = ?`, teamID, eventID)
	return err
}

// --- Partite e scoring -----------------------------------------------------------

type TAMatch struct {
	ID        string   `json:"id"`
	Court     string   `json:"court"`
	Time      string   `json:"time"`
	Status    string   `json:"status"` // scheduled | live | finished
	SetLabel  string   `json:"setLabel"`
	ScoreA    int      `json:"scoreA"`
	ScoreB    int      `json:"scoreB"`
	CurA      int      `json:"curA"`
	CurB      int      `json:"curB"`
	Sets      []string `json:"sets"`
	TeamAID   int64    `json:"teamAId"`
	TeamBID   int64    `json:"teamBId"`
	TeamAName string   `json:"teamAName"`
	TeamBName string   `json:"teamBName"`
}

func (s *Store) ListTAMatches(ctx context.Context, eventID int64) ([]TAMatch, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT m.id, m.court, m.scheduled_time, m.status, m.set_label,
		       m.score_a, m.score_b, m.cur_a, m.cur_b, m.sets_json,
		       m.team_a_id, m.team_b_id, ta.name, tb.name
		FROM matches m
		JOIN tournament_teams ta ON ta.id = m.team_a_id
		JOIN tournament_teams tb ON tb.id = m.team_b_id
		WHERE m.event_id = ?
		ORDER BY CASE m.status WHEN 'live' THEN 0 WHEN 'scheduled' THEN 1 ELSE 2 END,
		         m.scheduled_at, m.court`, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]TAMatch, 0, 16)
	for rows.Next() {
		var m TAMatch
		var setsJSON string
		if err := rows.Scan(&m.ID, &m.Court, &m.Time, &m.Status, &m.SetLabel,
			&m.ScoreA, &m.ScoreB, &m.CurA, &m.CurB, &setsJSON,
			&m.TeamAID, &m.TeamBID, &m.TeamAName, &m.TeamBName); err != nil {
			return nil, err
		}
		m.Sets = decodeSets(setsJSON)
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *Store) CreateTAMatch(ctx context.Context, eventID int64, court, timeLabel, scheduledAt string, teamA, teamB int64) (string, error) {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	id := fmt.Sprintf("m%d-%s", eventID, base64.RawURLEncoding.EncodeToString(buf))
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO matches (id, event_id, court, scheduled_time, scheduled_at, team_a_id, team_b_id, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, 'scheduled')`,
		id, eventID, court, timeLabel, scheduledAt, teamA, teamB)
	return id, err
}

func (s *Store) DeleteTAMatch(ctx context.Context, eventID int64, matchID string) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM matches WHERE id = ? AND event_id = ?`, matchID, eventID)
	return err
}

// ApplyScoreAction esegue un'azione della console scoring in transazione.
// Azioni: start, point_a, point_b, undo_a, undo_b, close_set, finish, reopen.
func (s *Store) ApplyScoreAction(ctx context.Context, eventID int64, matchID, action string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var status, setLabel, setsJSON string
	var scoreA, scoreB, curA, curB int
	err = tx.QueryRowContext(ctx, `
		SELECT status, set_label, sets_json, score_a, score_b, cur_a, cur_b
		FROM matches WHERE id = ? AND event_id = ?`, matchID, eventID).
		Scan(&status, &setLabel, &setsJSON, &scoreA, &scoreB, &curA, &curB)
	if errors.Is(err, sql.ErrNoRows) {
		return errTANotFound
	}
	if err != nil {
		return err
	}
	sets := decodeSets(setsJSON)

	switch action {
	case "start":
		status, setLabel, curA, curB = "live", "1° SET", 0, 0
	case "point_a":
		curA++
	case "point_b":
		curB++
	case "undo_a":
		if curA > 0 {
			curA--
		}
	case "undo_b":
		if curB > 0 {
			curB--
		}
	case "close_set":
		if curA == curB {
			return fmt.Errorf("set_tied") // un set di volley non può chiudersi in parità
		}
		sets = append(sets, fmt.Sprintf("%d-%d", curA, curB))
		if curA > curB {
			scoreA++
		} else {
			scoreB++
		}
		curA, curB = 0, 0
		setLabel = fmt.Sprintf("%d° SET", len(sets)+1)
	case "finish":
		status = "finished"
	case "reopen":
		status = "live"
	default:
		return fmt.Errorf("unknown_action")
	}

	encoded, err := json.Marshal(sets)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE matches SET status=?, set_label=?, sets_json=?, score_a=?, score_b=?, cur_a=?, cur_b=?
		WHERE id = ? AND event_id = ?`,
		status, setLabel, string(encoded), scoreA, scoreB, curA, curB, matchID, eventID); err != nil {
		return err
	}
	return tx.Commit()
}

// --- Sponsor ----------------------------------------------------------------------

type TASponsor struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	LogoURL    string `json:"logoUrl"`
	URL        string `json:"url"`
	Tier       string `json:"tier"`
	BrandColor string `json:"brandColor"`
}

func (s *Store) ListTASponsors(ctx context.Context, eventID int64) ([]TASponsor, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, logo_url, url, tier, brand_color
		FROM tournament_sponsors WHERE event_id = ? AND active = 1
		ORDER BY CASE tier WHEN 'main' THEN 0 ELSE 1 END, position, id`, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]TASponsor, 0, 8)
	for rows.Next() {
		var sp TASponsor
		if err := rows.Scan(&sp.ID, &sp.Name, &sp.LogoURL, &sp.URL, &sp.Tier, &sp.BrandColor); err != nil {
			return nil, err
		}
		out = append(out, sp)
	}
	return out, rows.Err()
}

func (s *Store) CreateTASponsor(ctx context.Context, eventID int64, sp TASponsor) (int64, error) {
	if sp.Tier != "main" {
		sp.Tier = "partner"
	}
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO tournament_sponsors (event_id, name, logo_url, url, tier, brand_color)
		VALUES (?, ?, ?, ?, ?, ?)`,
		eventID, strings.TrimSpace(sp.Name), sp.LogoURL, sp.URL, sp.Tier, sp.BrandColor)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) DeleteTASponsor(ctx context.Context, eventID, sponsorID int64) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM tournament_sponsors WHERE id = ? AND event_id = ?`, sponsorID, eventID)
	return err
}

// --- Impostazioni evento -------------------------------------------------------------

type TASettings struct {
	Name        string `json:"name"`
	Format      string `json:"format"`
	DateLabel   string `json:"dateLabel"`
	Location    string `json:"location"`
	StatusLabel string `json:"statusLabel"`
	PhaseLabel  string `json:"phaseLabel"`
}

func (s *Store) GetTASettings(ctx context.Context, eventID int64) (*TASettings, string, error) {
	var st TASettings
	var slug string
	err := s.db.QueryRowContext(ctx, `
		SELECT COALESCE(name,''), COALESCE(format,''), COALESCE(date_label,''),
		       COALESCE(location,''), COALESCE(status_label,''), COALESCE(phase_label,''), COALESCE(slug,'')
		FROM events WHERE id = ?`, eventID).
		Scan(&st.Name, &st.Format, &st.DateLabel, &st.Location, &st.StatusLabel, &st.PhaseLabel, &slug)
	return &st, slug, err
}

func (s *Store) UpdateTASettings(ctx context.Context, eventID int64, st TASettings) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE events SET name=?, format=?, date_label=?, location=?, status_label=?, phase_label=?
		WHERE id = ? AND type = 'tournament'`,
		st.Name, st.Format, st.DateLabel, st.Location, st.StatusLabel, st.PhaseLabel, eventID)
	return err
}
