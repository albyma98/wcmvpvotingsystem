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
		`CREATE TABLE IF NOT EXISTS tournament_shop_products (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			event_id INTEGER NOT NULL,
			image_url TEXT NOT NULL,
			title TEXT NOT NULL DEFAULT '',
			description TEXT NOT NULL DEFAULT '',
			price_cents INTEGER NOT NULL,
			extras_json TEXT NOT NULL DEFAULT '[]',
			position INTEGER NOT NULL DEFAULT 0,
			active INTEGER NOT NULL DEFAULT 1,
			created_at TEXT NOT NULL DEFAULT (datetime('now'))
		);`,
		`CREATE TABLE IF NOT EXISTS tournament_shop_reservations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			event_id INTEGER NOT NULL,
			product_id INTEGER NOT NULL,
			product_title TEXT NOT NULL DEFAULT '',
			product_image_url TEXT NOT NULL DEFAULT '',
			base_price_cents INTEGER NOT NULL,
			selected_extras_json TEXT NOT NULL DEFAULT '[]',
			total_price_cents INTEGER NOT NULL,
			first_name TEXT NOT NULL,
			last_name TEXT NOT NULL,
			phone TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'new',
			created_at TEXT NOT NULL DEFAULT (datetime('now'))
		);`,
		// Rosa giocatori per squadra: Nome/Cognome facoltativi (max 8 lato UI),
		// serviranno a popolare la votazione MVP del pubblico. Nessuna FK (schema
		// minimale coerente col resto del mondo torneo); la pulizia è esplicita
		// in DeleteTATeam/DeleteTournament.
		`CREATE TABLE IF NOT EXISTS tournament_players (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			event_id INTEGER NOT NULL,
			team_id INTEGER NOT NULL,
			first_name TEXT NOT NULL DEFAULT '',
			last_name TEXT NOT NULL DEFAULT '',
			gender TEXT NOT NULL DEFAULT 'male',
			position INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL DEFAULT (datetime('now'))
		);`,
	}
	for _, q := range stmts {
		if _, err := s.db.Exec(q); err != nil {
			return fmt.Errorf("tournament admin tables: %w", err)
		}
	}

	// Reconcile: alcune installazioni hanno tournament_teams/tournament_sponsors
	// ereditate da un design precedente (colonna `tournament_id NOT NULL`, quando
	// i tornei erano una tabella a parte). Con il modello attuale (torneo = event)
	// gli INSERT usano event_id e falliscono con "NOT NULL constraint failed:
	// ...tournament_id". Se rileviamo lo schema vecchio, ricostruiamo la tabella
	// allo schema canonico preservando i dati compatibili.
	if err := s.reconcileTournamentTable("tournament_teams",
		`CREATE TABLE tournament_teams (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			event_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			short_name TEXT NOT NULL DEFAULT '',
			city TEXT NOT NULL DEFAULT '',
			logo_url TEXT NOT NULL DEFAULT '',
			group_name TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL DEFAULT (datetime('now'))
		)`,
		[]string{"id", "event_id", "name", "short_name", "city", "logo_url", "group_name", "created_at"}); err != nil {
		return err
	}
	if err := s.reconcileTournamentTable("tournament_sponsors",
		`CREATE TABLE tournament_sponsors (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			event_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			logo_url TEXT NOT NULL DEFAULT '',
			url TEXT NOT NULL DEFAULT '',
			tier TEXT NOT NULL DEFAULT 'partner',
			brand_color TEXT NOT NULL DEFAULT '',
			position INTEGER NOT NULL DEFAULT 0,
			active INTEGER NOT NULL DEFAULT 1
		)`,
		[]string{"id", "event_id", "name", "logo_url", "url", "tier", "brand_color", "position", "active"}); err != nil {
		return err
	}
	// I DB predatanti il modello "torneo = event" hanno una `tournament_players`
	// legacy (colonna `tournament_id NOT NULL`, niente `event_id`/`first_name`/
	// `last_name`) dalla vecchia feature tornei poi ripristinata: CREATE IF NOT
	// EXISTS la lascerebbe com'è → gli INSERT/SELECT sui nuovi campi fallirebbero
	// (500 su aggiungi-squadra e su lista squadre). Reconcile allo schema canonico.
	if err := s.reconcileTournamentTable("tournament_players",
		`CREATE TABLE tournament_players (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			event_id INTEGER NOT NULL,
			team_id INTEGER NOT NULL,
			first_name TEXT NOT NULL DEFAULT '',
			last_name TEXT NOT NULL DEFAULT '',
			gender TEXT NOT NULL DEFAULT 'male',
			position INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL DEFAULT (datetime('now'))
		)`,
		[]string{"id", "event_id", "team_id", "first_name", "last_name", "gender", "position", "created_at"}); err != nil {
		return err
	}

	// ALTER idempotenti (SQLite non ha IF NOT EXISTS su ADD COLUMN): garantiscono
	// che tutte le colonne esistano anche su DB creati da versioni precedenti
	// (difesa contro drift dello schema → evita "no such column" mascherato da 500).
	for _, alter := range []string{
		// Default 1 preserva il comportamento dei tornei esistenti; i nuovi
		// vengono creati esplicitamente con le azioni dei tifosi disabilitate.
		`ALTER TABLE events ADD COLUMN tournament_started INTEGER NOT NULL DEFAULT 1`,
		`ALTER TABLE events ADD COLUMN organizer_logo_url TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE matches ADD COLUMN cur_a INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE matches ADD COLUMN cur_b INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE tournament_teams ADD COLUMN short_name TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE tournament_teams ADD COLUMN city TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE tournament_teams ADD COLUMN logo_url TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE tournament_teams ADD COLUMN group_name TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE tournament_players ADD COLUMN team_id INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE tournament_players ADD COLUMN first_name TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE tournament_players ADD COLUMN last_name TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE tournament_players ADD COLUMN gender TEXT NOT NULL DEFAULT 'male'`,
		`ALTER TABLE tournament_players ADD COLUMN position INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE tournament_sponsors ADD COLUMN logo_url TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE tournament_sponsors ADD COLUMN url TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE tournament_sponsors ADD COLUMN tier TEXT NOT NULL DEFAULT 'partner'`,
		`ALTER TABLE tournament_sponsors ADD COLUMN brand_color TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE tournament_sponsors ADD COLUMN position INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE tournament_sponsors ADD COLUMN active INTEGER NOT NULL DEFAULT 1`,
	} {
		if _, err := s.db.Exec(alter); err != nil && !strings.Contains(err.Error(), "duplicate column") {
			return fmt.Errorf("tournament column ensure (%s): %w", alter, err)
		}
	}
	return nil
}

// tableColumns ritorna l'insieme dei nomi colonna di una tabella (vuoto se assente).
func (s *Store) tableColumns(table string) (map[string]bool, error) {
	rows, err := s.db.Query(`PRAGMA table_info(` + table + `)`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	cols := map[string]bool{}
	for rows.Next() {
		var cid, notnull, pk int
		var name, ctype string
		var dflt interface{}
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return nil, err
		}
		cols[name] = true
	}
	return cols, rows.Err()
}

// reconcileTournamentTable ricostruisce `table` allo schema canonico quando
// rileva lo schema legacy (colonna `tournament_id`) o l'assenza di `event_id`.
// Preserva i dati copiando solo le colonne in comune. Operazioni FK-safe
// (nessuna tabella referenzia queste): rename → create → insert → drop.
func (s *Store) reconcileTournamentTable(table, createSQL string, canonical []string) error {
	cols, err := s.tableColumns(table)
	if err != nil {
		return fmt.Errorf("reconcile %s (table_info): %w", table, err)
	}
	if len(cols) == 0 {
		return nil // assente: la crea il CREATE IF NOT EXISTS
	}
	if !cols["tournament_id"] && cols["event_id"] {
		return nil // già schema canonico (event-based)
	}

	shared := make([]string, 0, len(canonical))
	for _, c := range canonical {
		if cols[c] {
			shared = append(shared, c)
		}
	}
	stmts := []string{
		fmt.Sprintf(`ALTER TABLE %s RENAME TO %s_old`, table, table),
		createSQL,
	}
	if cols["event_id"] && len(shared) > 0 {
		list := strings.Join(shared, ", ")
		stmts = append(stmts, fmt.Sprintf(`INSERT INTO %s (%s) SELECT %s FROM %s_old`, table, list, list, table))
	}
	stmts = append(stmts, fmt.Sprintf(`DROP TABLE %s_old`, table))
	for _, q := range stmts {
		if _, err := s.db.Exec(q); err != nil {
			return fmt.Errorf("reconcile %s: %w", table, err)
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

	// events ha FK NOT NULL su organization_id/team1_id/team2_id (schema club) e
	// le FK sono attive (PRAGMA foreign_keys=ON). Un torneo non ha né org né due
	// squadre fisse: usiamo righe sentinella CONDIVISE (una sola org + una sola
	// team di sistema, riusate da tutti i tornei) così l'INSERT soddisfa le FK
	// senza rifare la tabella events (referenziata da altre 19 FK) e senza
	// toccare i dati/logica del club. I valori non sono usati dalle query torneo
	// (che filtrano per slug + type='tournament').
	var sysOrgID, sysTeamID int64
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO organizations (name, slug, is_active)
		SELECT 'Tornei (sistema)', '__tournament_sys__', 0
		WHERE NOT EXISTS (SELECT 1 FROM organizations WHERE slug = '__tournament_sys__')`); err != nil {
		return 0, err
	}
	if err := tx.QueryRowContext(ctx,
		`SELECT id FROM organizations WHERE slug = '__tournament_sys__' LIMIT 1`).Scan(&sysOrgID); err != nil {
		return 0, err
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO teams (name, championship)
		SELECT '—', '__tournament_sys__'
		WHERE NOT EXISTS (SELECT 1 FROM teams WHERE championship = '__tournament_sys__')`); err != nil {
		return 0, err
	}
	if err := tx.QueryRowContext(ctx,
		`SELECT id FROM teams WHERE championship = '__tournament_sys__' LIMIT 1`).Scan(&sysTeamID); err != nil {
		return 0, err
	}

	res, err := tx.ExecContext(ctx, `
		INSERT INTO events (organization_id, team1_id, team2_id, start_datetime, location,
		                    slug, name, format, date_label, status_label, phase_label, type,
		                    tournament_started)
		VALUES (?, ?, ?, '', ?, ?, ?, ?, ?, 'TORNEO IN ARRIVO', 'ISCRIZIONI', 'tournament', 0)`,
		sysOrgID, sysTeamID, sysTeamID, in.Location, in.Slug, in.Name, in.Format, in.DateLabel)
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

// SetTAAdminPassword imposta (o crea, se mancante) le credenziali admin di un
// torneo e ritorna lo username effettivo. passwordHash è già bcrypt-ato dal
// chiamante. Cambiando la password invalidiamo le sessioni attive del torneo:
// un reset dal master deve sloggare chi era entrato con la vecchia password.
func (s *Store) SetTAAdminPassword(ctx context.Context, eventID int64, passwordHash string) (string, error) {
	var slug string
	err := s.db.QueryRowContext(ctx,
		`SELECT slug FROM events WHERE id = ? AND type = 'tournament'`, eventID).Scan(&slug)
	if errors.Is(err, sql.ErrNoRows) {
		return "", errTANotFound
	}
	if err != nil {
		return "", err
	}

	var username string
	err = s.db.QueryRowContext(ctx,
		`SELECT username FROM tournament_admins WHERE event_id = ?`, eventID).Scan(&username)
	if errors.Is(err, sql.ErrNoRows) {
		// Torneo senza admin (installazione legacy): lo creiamo ora.
		username = "ta-" + slug
		if _, err := s.db.ExecContext(ctx,
			`INSERT INTO tournament_admins (event_id, username, password_hash) VALUES (?, ?, ?)`,
			eventID, username, passwordHash); err != nil {
			return "", err
		}
		return username, nil
	}
	if err != nil {
		return "", err
	}
	if _, err := s.db.ExecContext(ctx,
		`UPDATE tournament_admins SET password_hash = ? WHERE event_id = ?`, passwordHash, eventID); err != nil {
		return "", err
	}
	_, _ = s.db.ExecContext(ctx, `DELETE FROM tournament_admin_sessions WHERE event_id = ?`, eventID)
	return username, nil
}

// DeleteTournament rimuove un torneo e TUTTI i suoi dati in transazione: admin,
// sessioni, operatori di campo, squadre, sponsor, partite, tiles e la riga event.
// Le sentinelle di sistema (org/team condivise) restano: sono riusate dagli
// altri tornei. Ordine figli→padre per rispettare eventuali FK (PRAGMA ON).
func (s *Store) DeleteTournament(ctx context.Context, eventID int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var n int
	if err := tx.QueryRowContext(ctx,
		`SELECT COUNT(1) FROM events WHERE id = ? AND type = 'tournament'`, eventID).Scan(&n); err != nil {
		return err
	}
	if n == 0 {
		return errTANotFound
	}

	stmts := []string{
		`DELETE FROM court_operator_sessions WHERE operator_id IN (SELECT id FROM court_operators WHERE event_id = ?)`,
		`DELETE FROM court_operators WHERE event_id = ?`,
		`DELETE FROM tournament_admin_sessions WHERE event_id = ?`,
		`DELETE FROM tournament_admins WHERE event_id = ?`,
		`DELETE FROM tournament_shop_reservations WHERE event_id = ?`,
		`DELETE FROM tournament_shop_products WHERE event_id = ?`,
		`DELETE FROM tournament_sponsors WHERE event_id = ?`,
		`DELETE FROM matches WHERE event_id = ?`,
		`DELETE FROM tournament_players WHERE event_id = ?`,
		`DELETE FROM tournament_teams WHERE event_id = ?`,
		`DELETE FROM event_tiles WHERE event_id = ?`,
		`DELETE FROM events WHERE id = ? AND type = 'tournament'`,
	}
	for _, q := range stmts {
		if _, err := tx.ExecContext(ctx, q, eventID); err != nil {
			return fmt.Errorf("delete tournament (%s): %w", q, err)
		}
	}
	return tx.Commit()
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

func (s *Store) IsTournamentStarted(ctx context.Context, eventID int64) (bool, error) {
	var started int
	err := s.db.QueryRowContext(ctx, `
		SELECT COALESCE(tournament_started,1)
		FROM events WHERE id = ? AND type = 'tournament'`, eventID).Scan(&started)
	if errors.Is(err, sql.ErrNoRows) {
		return false, errTANotFound
	}
	return started == 1, err
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
	ID        int64      `json:"id"`
	Name      string     `json:"name"`
	ShortName string     `json:"shortName"`
	City      string     `json:"city"`
	GroupName string     `json:"groupName"`
	Players   []TAPlayer `json:"players"`
}

// TAPlayer è un giocatore in rosa (Nome/Cognome facoltativi).
// Gender ("male"/"female") serve alla votazione MVP separata uomo/donna.
type TAPlayer struct {
	ID        int64  `json:"id"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Gender    string `json:"gender"`
}

// normalizeTAGender riduce l'input alle due categorie ammesse; default "male".
func normalizeTAGender(g string) string {
	switch strings.ToLower(strings.TrimSpace(g)) {
	case "female", "f", "femmina", "donna":
		return "female"
	default:
		return "male"
	}
}

// maxPlayersPerTeam limita la rosa lato server (la UI espone 8 coppie).
const maxPlayersPerTeam = 8

func (s *Store) ListTATeams(ctx context.Context, eventID int64) ([]TATeam, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, short_name, city, group_name
		FROM tournament_teams WHERE event_id = ? ORDER BY group_name, name COLLATE NOCASE`, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]TATeam, 0, 16)
	idx := map[int64]int{}
	for rows.Next() {
		var t TATeam
		if err := rows.Scan(&t.ID, &t.Name, &t.ShortName, &t.City, &t.GroupName); err != nil {
			return nil, err
		}
		t.Players = []TAPlayer{}
		idx[t.ID] = len(out)
		out = append(out, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Rosa: una sola query per l'intero evento, raggruppata in memoria.
	prows, err := s.db.QueryContext(ctx, `
		SELECT id, team_id, first_name, last_name, gender
		FROM tournament_players WHERE event_id = ?
		ORDER BY team_id, position, id`, eventID)
	if err != nil {
		return nil, err
	}
	defer prows.Close()
	for prows.Next() {
		var p TAPlayer
		var teamID int64
		if err := prows.Scan(&p.ID, &teamID, &p.FirstName, &p.LastName, &p.Gender); err != nil {
			return nil, err
		}
		if i, ok := idx[teamID]; ok {
			out[i].Players = append(out[i].Players, p)
		}
	}
	return out, prows.Err()
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
		res, err := tx.ExecContext(ctx, `
			INSERT INTO tournament_teams (event_id, name, short_name, city, group_name)
			VALUES (?, ?, ?, ?, ?)`,
			eventID, name, strings.TrimSpace(t.ShortName), strings.TrimSpace(t.City), strings.TrimSpace(t.GroupName))
		if err != nil {
			return 0, err
		}
		teamID, err := res.LastInsertId()
		if err != nil {
			return 0, err
		}
		if err := insertTAPlayers(ctx, tx, eventID, teamID, t.Players); err != nil {
			return 0, err
		}
		n++
	}
	return n, tx.Commit()
}

// insertTAPlayers inserisce la rosa (scarta le coppie interamente vuote,
// tronca a maxPlayersPerTeam). Condivisa da InsertTATeams e ReplaceTAPlayers.
func insertTAPlayers(ctx context.Context, tx *sql.Tx, eventID, teamID int64, players []TAPlayer) error {
	pos := 0
	for _, p := range players {
		first := strings.TrimSpace(p.FirstName)
		last := strings.TrimSpace(p.LastName)
		if first == "" && last == "" {
			continue // coppia vuota: si salta, non è un giocatore
		}
		if pos >= maxPlayersPerTeam {
			break
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO tournament_players (event_id, team_id, first_name, last_name, gender, position)
			VALUES (?, ?, ?, ?, ?, ?)`,
			eventID, teamID, first, last, normalizeTAGender(p.Gender), pos); err != nil {
			return err
		}
		pos++
	}
	return nil
}

// ReplaceTAPlayers sostituisce l'intera rosa di una squadra (delete + insert in tx).
// Verifica che la squadra appartenga all'evento (scoping hard).
func (s *Store) ReplaceTAPlayers(ctx context.Context, eventID, teamID int64, players []TAPlayer) error {
	var owned int
	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(1) FROM tournament_teams WHERE id = ? AND event_id = ?`,
		teamID, eventID).Scan(&owned); err != nil {
		return err
	}
	if owned == 0 {
		return errTANotFound
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `
		DELETE FROM tournament_players WHERE event_id = ? AND team_id = ?`, eventID, teamID); err != nil {
		return err
	}
	if err := insertTAPlayers(ctx, tx, eventID, teamID, players); err != nil {
		return err
	}
	return tx.Commit()
}

// UpdateTATeam modifica gli attributi editabili di una squadra (nome, sigla,
// città, girone). La rosa NON è toccata (si modifica con ReplaceTAPlayers).
// errTANotFound se la squadra non appartiene all'evento; "name_required" se vuoto.
func (s *Store) UpdateTATeam(ctx context.Context, eventID, teamID int64, name, shortName, city, groupName string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("name_required")
	}
	res, err := s.db.ExecContext(ctx, `
		UPDATE tournament_teams SET name = ?, short_name = ?, city = ?, group_name = ?
		WHERE id = ? AND event_id = ?`,
		name, strings.TrimSpace(shortName), strings.TrimSpace(city), strings.TrimSpace(groupName), teamID, eventID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return errTANotFound
	}
	return nil
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
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `
		DELETE FROM tournament_players WHERE event_id = ? AND team_id = ?`, eventID, teamID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
		DELETE FROM tournament_teams WHERE id = ? AND event_id = ?`, teamID, eventID); err != nil {
		return err
	}
	return tx.Commit()
}

// --- Partite e scoring -----------------------------------------------------------

type TAMatch struct {
	ID        string   `json:"id"`
	Court     string   `json:"court"`
	Stage     string   `json:"stage"`
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
	IsAnchor  bool     `json:"isAnchor"` // partita "inizio torneo" (taglio del ciclo cronologico)
}

func (s *Store) ListTAMatches(ctx context.Context, eventID int64) ([]TAMatch, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT m.id, m.court, COALESCE(m.stage,''), m.scheduled_time, m.status, m.set_label,
		       m.score_a, m.score_b, m.cur_a, m.cur_b, m.sets_json,
		       m.team_a_id, m.team_b_id,
		       COALESCE(NULLIF(ta.name,''), m.team_a_label, ''),
		       COALESCE(NULLIF(tb.name,''), m.team_b_label, ''),
		       m.is_anchor
		FROM matches m
		LEFT JOIN tournament_teams ta ON ta.id = m.team_a_id
		LEFT JOIN tournament_teams tb ON tb.id = m.team_b_id
		WHERE m.event_id = ?
		ORDER BY CASE m.status WHEN 'live' THEN 0 WHEN 'scheduled' THEN 1 ELSE 2 END,
		         CASE WHEN m.scheduled_time >= COALESCE(
		                (SELECT a.scheduled_time FROM matches a
		                   WHERE a.event_id = m.event_id AND a.is_anchor = 1 LIMIT 1), '')
		              THEN 0 ELSE 1 END,
		         m.scheduled_time, m.court`, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]TAMatch, 0, 16)
	for rows.Next() {
		var m TAMatch
		var setsJSON string
		if err := rows.Scan(&m.ID, &m.Court, &m.Stage, &m.Time, &m.Status, &m.SetLabel,
			&m.ScoreA, &m.ScoreB, &m.CurA, &m.CurB, &setsJSON,
			&m.TeamAID, &m.TeamBID, &m.TeamAName, &m.TeamBName, &m.IsAnchor); err != nil {
			return nil, err
		}
		m.Sets = decodeSets(setsJSON)
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *Store) CreateTAMatch(ctx context.Context, eventID int64, court, timeLabel, scheduledAt, stage string, teamA, teamB int64) (string, error) {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	id := fmt.Sprintf("m%d-%s", eventID, base64.RawURLEncoding.EncodeToString(buf))
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO matches (id, event_id, court, scheduled_time, scheduled_at, stage, team_a_id, team_b_id, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'scheduled')`,
		id, eventID, court, timeLabel, scheduledAt, strings.TrimSpace(stage), teamA, teamB)
	return id, err
}

// UpdateTAMatch modifica gli attributi editabili di una partita (campo, orario,
// squadre, fase). Punteggi/stato restano invariati. errTANotFound se assente.
func (s *Store) UpdateTAMatch(ctx context.Context, eventID int64, matchID, court, timeLabel, scheduledAt, stage string, teamA, teamB int64) error {
	res, err := s.db.ExecContext(ctx, `
		UPDATE matches SET court = ?, scheduled_time = ?, scheduled_at = ?, stage = ?, team_a_id = ?, team_b_id = ?
		WHERE id = ? AND event_id = ?`,
		court, timeLabel, scheduledAt, strings.TrimSpace(stage), teamA, teamB, matchID, eventID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return errTANotFound
	}
	return nil
}

// SetTAMatchAnchor imposta (on=true) o rimuove (on=false) la partita "inizio
// torneo". L'àncora è unica per evento: impostarne una azzera tutte le altre
// nella stessa transazione. errTANotFound se la partita non è dell'evento.
func (s *Store) SetTAMatchAnchor(ctx context.Context, eventID int64, matchID string, on bool) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx,
		`UPDATE matches SET is_anchor = 0 WHERE event_id = ?`, eventID); err != nil {
		return err
	}
	anchor := 0
	if on {
		anchor = 1
	}
	res, err := tx.ExecContext(ctx,
		`UPDATE matches SET is_anchor = ? WHERE id = ? AND event_id = ?`, anchor, matchID, eventID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return errTANotFound
	}
	return tx.Commit()
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

	var status, setLabel, setsJSON, winToID, loseToID string
	var scoreA, scoreB, curA, curB int
	var teamAID, teamBID int64
	var winToSlot, loseToSlot int
	err = tx.QueryRowContext(ctx, `
		SELECT status, set_label, sets_json, score_a, score_b, cur_a, cur_b,
		       team_a_id, team_b_id, win_to_id, win_to_slot, lose_to_id, lose_to_slot
		FROM matches WHERE id = ? AND event_id = ?`, matchID, eventID).
		Scan(&status, &setLabel, &setsJSON, &scoreA, &scoreB, &curA, &curB,
			&teamAID, &teamBID, &winToID, &winToSlot, &loseToID, &loseToSlot)
	if errors.Is(err, sql.ErrNoRows) {
		return errTANotFound
	}
	if err != nil {
		return err
	}
	sets := decodeSets(setsJSON)

	switch action {
	case "start":
		// Una partita del tabellone non parte finché entrambe le squadre non
		// sono note (arrivano dai turni precedenti via auto-avanzamento).
		if teamAID == 0 || teamBID == 0 {
			return fmt.Errorf("teams_not_ready")
		}
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

	// Auto-avanzamento: alla chiusura di una partita del tabellone, il vincitore
	// (e, se c'è finalina, il perdente) riempie lo slot del turno successivo.
	if action == "finish" && scoreA != scoreB {
		winner, loser := teamAID, teamBID
		if scoreB > scoreA {
			winner, loser = teamBID, teamAID
		}
		if winToID != "" && winner != 0 {
			if err := advanceTeam(ctx, tx, eventID, winToID, winToSlot, winner); err != nil {
				return err
			}
		}
		if loseToID != "" && loser != 0 {
			if err := advanceTeam(ctx, tx, eventID, loseToID, loseToSlot, loser); err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}

// advanceTeam scrive la squadra nello slot (0=A, 1=B) della partita target e
// azzera l'etichetta segnaposto corrispondente. Nomi colonna fissi (non input).
func advanceTeam(ctx context.Context, tx *sql.Tx, eventID int64, targetID string, slot int, teamID int64) error {
	col, labelCol := "team_a_id", "team_a_label"
	if slot == 1 {
		col, labelCol = "team_b_id", "team_b_label"
	}
	_, err := tx.ExecContext(ctx,
		`UPDATE matches SET `+col+` = ?, `+labelCol+` = '' WHERE id = ? AND event_id = ?`,
		teamID, targetID, eventID)
	return err
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

// --- Shop -------------------------------------------------------------------------

func decodeTournamentShopExtras(raw string) []TournamentShopExtra {
	if strings.TrimSpace(raw) == "" {
		return []TournamentShopExtra{}
	}
	var extras []TournamentShopExtra
	if err := json.Unmarshal([]byte(raw), &extras); err != nil || extras == nil {
		return []TournamentShopExtra{}
	}
	return extras
}

func (s *Store) ListTAShopProducts(ctx context.Context, eventID int64) ([]TournamentShopProduct, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, image_url, title, description, price_cents, extras_json
		FROM tournament_shop_products
		WHERE event_id = ? AND active = 1
		ORDER BY position, id`, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]TournamentShopProduct, 0, 8)
	for rows.Next() {
		var product TournamentShopProduct
		var extrasJSON string
		if err := rows.Scan(&product.ID, &product.ImageURL, &product.Title, &product.Description,
			&product.PriceCents, &extrasJSON); err != nil {
			return nil, err
		}
		product.Extras = decodeTournamentShopExtras(extrasJSON)
		out = append(out, product)
	}
	return out, rows.Err()
}

func (s *Store) CreateTAShopProduct(ctx context.Context, eventID int64, product TournamentShopProduct) (int64, error) {
	extrasJSON, err := json.Marshal(product.Extras)
	if err != nil {
		return 0, err
	}
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO tournament_shop_products
			(event_id, image_url, title, description, price_cents, extras_json)
		VALUES (?, ?, ?, ?, ?, ?)`,
		eventID, product.ImageURL, product.Title, product.Description, product.PriceCents, string(extrasJSON))
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) DeleteTAShopProduct(ctx context.Context, eventID, productID int64) error {
	res, err := s.db.ExecContext(ctx, `
		DELETE FROM tournament_shop_products WHERE id = ? AND event_id = ?`, productID, eventID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return errTANotFound
	}
	return nil
}

type TournamentShopReservation struct {
	ID              int64                 `json:"id"`
	ProductID       int64                 `json:"productId"`
	ProductTitle    string                `json:"productTitle"`
	ProductImageURL string                `json:"productImageUrl"`
	BasePriceCents  int                   `json:"basePriceCents"`
	SelectedExtras  []TournamentShopExtra `json:"selectedExtras"`
	TotalPriceCents int                   `json:"totalPriceCents"`
	FirstName       string                `json:"firstName"`
	LastName        string                `json:"lastName"`
	Phone           string                `json:"phone"`
	Status          string                `json:"status"`
	CreatedAt       string                `json:"createdAt"`
}

func (s *Store) CreateTournamentShopReservation(
	ctx context.Context,
	slug string,
	productID int64,
	extraTitles []string,
	firstName, lastName, phone string,
) (int64, error) {
	var eventID int64
	var title, imageURL, extrasJSON string
	var basePrice int
	err := s.db.QueryRowContext(ctx, `
		SELECT e.id, p.title, p.image_url, p.price_cents, p.extras_json
		FROM tournament_shop_products p
		JOIN events e ON e.id = p.event_id
		WHERE e.slug = ? AND e.type = 'tournament' AND p.id = ? AND p.active = 1`,
		slug, productID).Scan(&eventID, &title, &imageURL, &basePrice, &extrasJSON)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, errTANotFound
	}
	if err != nil {
		return 0, err
	}

	available := decodeTournamentShopExtras(extrasJSON)
	wanted := make(map[string]bool, len(extraTitles))
	for _, extraTitle := range extraTitles {
		clean := strings.TrimSpace(extraTitle)
		if clean != "" {
			wanted[clean] = true
		}
	}
	selected := make([]TournamentShopExtra, 0, len(wanted))
	total := basePrice
	for _, extra := range available {
		if wanted[extra.Title] {
			selected = append(selected, extra)
			total += extra.PriceCents
			delete(wanted, extra.Title)
		}
	}
	if len(wanted) != 0 {
		return 0, fmt.Errorf("invalid_extra")
	}
	selectedJSON, err := json.Marshal(selected)
	if err != nil {
		return 0, err
	}
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO tournament_shop_reservations
			(event_id, product_id, product_title, product_image_url, base_price_cents,
			 selected_extras_json, total_price_cents, first_name, last_name, phone)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		eventID, productID, title, imageURL, basePrice, string(selectedJSON), total,
		firstName, lastName, phone)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) ListTAShopReservations(ctx context.Context, eventID int64) ([]TournamentShopReservation, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, product_id, product_title, product_image_url, base_price_cents,
		       selected_extras_json, total_price_cents, first_name, last_name,
		       phone, status, created_at
		FROM tournament_shop_reservations
		WHERE event_id = ?
		ORDER BY id DESC`, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]TournamentShopReservation, 0, 16)
	for rows.Next() {
		var reservation TournamentShopReservation
		var extrasJSON string
		if err := rows.Scan(
			&reservation.ID, &reservation.ProductID, &reservation.ProductTitle,
			&reservation.ProductImageURL, &reservation.BasePriceCents, &extrasJSON,
			&reservation.TotalPriceCents, &reservation.FirstName, &reservation.LastName,
			&reservation.Phone, &reservation.Status, &reservation.CreatedAt,
		); err != nil {
			return nil, err
		}
		reservation.SelectedExtras = decodeTournamentShopExtras(extrasJSON)
		out = append(out, reservation)
	}
	return out, rows.Err()
}

// --- Impostazioni evento -------------------------------------------------------------

type TASettings struct {
	Name        string `json:"name"`
	Format      string `json:"format"`
	DateLabel   string `json:"dateLabel"`
	Location    string `json:"location"`
	StatusLabel string `json:"statusLabel"`
	PhaseLabel  string `json:"phaseLabel"`
	// Intestazione della home tifosi: immagine (data-URL o URL) mostrata al posto
	// del titolo testuale. Vuota = si usa il nome del torneo.
	Logo string `json:"logoUrl"`
	// Logo dell'organizzatore mostrato nella tile in basso a destra del layout Sunset.
	OrganizerLogo string `json:"organizerLogoUrl"`
	PointsPerWin  int    `json:"pointsPerWin"`
	PointsPerDraw int    `json:"pointsPerDraw"`
	PointsPerLoss int    `json:"pointsPerLoss"`
	// Formula set: 3 (al meglio dei 3 = 2 su 3) | 5 (al meglio dei 5 = 3 su 5).
	SetsBestOf int `json:"setsBestOf"`
	// Punti a chi vince/perde al tie-break (set decisivo): tipicamente
	// PointsPerLoss < PointsPerTieLoss < PointsPerTieWin < PointsPerWin.
	PointsPerTieWin  int `json:"pointsPerTieWin"`
	PointsPerTieLoss int `json:"pointsPerTieLoss"`
	// Se false, i pareggi non contano in classifica (colonna N nascosta ai tifosi).
	AllowDraws bool `json:"allowDraws"`
	// Modalità votazione MVP del pubblico: true = MVP uomo + MVP donna (2 voti per
	// device), false = MVP unico indipendente dal sesso (1 voto per device).
	MvpByGender bool `json:"mvpByGender"`
	// Se false, i tifosi possono consultare il torneo ma non votare o caricare foto.
	TournamentStarted bool `json:"tournamentStarted"`
	// Fase finale: quante squadre passano per girone + finalina 3°/4° posto.
	BracketQualifiers int  `json:"bracketQualifiers"`
	BracketThirdPlace bool `json:"bracketThirdPlace"`
	// Testo informativo mostrato sotto la classifica pubblica.
	StandingsLegendText string `json:"standingsLegendText"`
	// Grafica della home tifosi: 'classic' | 'sunset'.
	FanLayout string `json:"fanLayout"`
	// Premi del torneo mostrati nella modale "Premi" del layout Sunset.
	Prizes TournamentPrizes `json:"prizes"`
}

// TournamentPrizes raccoglie i premi per posizione/categoria (testo libero, es.
// "Buono 200€"). Campo vuoto = categoria non mostrata ai tifosi.
type TournamentPrizes struct {
	First           string `json:"first"`           // 1° classificato
	Second          string `json:"second"`          // 2° classificato
	Third           string `json:"third"`           // 3° classificato
	OrgMvpMale      string `json:"orgMvpMale"`      // MVP maschile (scelto dagli organizzatori)
	OrgMvpFemale    string `json:"orgMvpFemale"`    // MVP femminile (organizzatori)
	PublicMvpMale   string `json:"publicMvpMale"`   // MVP maschile (voto del pubblico)
	PublicMvpFemale string `json:"publicMvpFemale"` // MVP femminile (voto del pubblico)
}

// decodeTournamentPrizes converte il JSON salvato in struct (vuoto se assente).
func decodeTournamentPrizes(raw string) TournamentPrizes {
	var p TournamentPrizes
	raw = strings.TrimSpace(raw)
	if raw != "" {
		_ = json.Unmarshal([]byte(raw), &p)
	}
	return p
}

// sanitizeTournamentPrizes ripulisce ogni campo (trim + tetto lunghezza) per
// evitare payload abnormi nel testo dei premi.
func sanitizeTournamentPrizes(p TournamentPrizes) TournamentPrizes {
	clip := func(s string) string {
		s = strings.TrimSpace(s)
		if len(s) > 200 {
			s = s[:200]
		}
		return s
	}
	return TournamentPrizes{
		First:           clip(p.First),
		Second:          clip(p.Second),
		Third:           clip(p.Third),
		OrgMvpMale:      clip(p.OrgMvpMale),
		OrgMvpFemale:    clip(p.OrgMvpFemale),
		PublicMvpMale:   clip(p.PublicMvpMale),
		PublicMvpFemale: clip(p.PublicMvpFemale),
	}
}

func (s *Store) GetTASettings(ctx context.Context, eventID int64) (*TASettings, string, error) {
	var st TASettings
	var slug, prizesJSON string
	var thirdPlace, allowDraws, mvpByGender, tournamentStarted int
	err := s.db.QueryRowContext(ctx, `
		SELECT COALESCE(name,''), COALESCE(format,''), COALESCE(date_label,''),
		       COALESCE(location,''), COALESCE(status_label,''), COALESCE(phase_label,''),
		       COALESCE(logo_url,''), COALESCE(organizer_logo_url,''),
		       COALESCE(points_per_win,3), COALESCE(points_per_draw,1), COALESCE(points_per_loss,0),
		       COALESCE(sets_best_of,3), COALESCE(points_per_tie_win,2), COALESCE(points_per_tie_loss,1),
		       COALESCE(allow_draws,1), COALESCE(mvp_by_gender,1), COALESCE(tournament_started,1),
		       COALESCE(bracket_qualifiers,2), COALESCE(bracket_third_place,0),
		       COALESCE(standings_legend_text,'Primi 2 di ogni girone alla fase finale · Ordinamento: punti, quoziente set, quoziente punti'),
		       COALESCE(fan_layout,'classic'), COALESCE(prizes_json,''), COALESCE(slug,'')
		FROM events WHERE id = ?`, eventID).
		Scan(&st.Name, &st.Format, &st.DateLabel, &st.Location, &st.StatusLabel, &st.PhaseLabel,
			&st.Logo, &st.OrganizerLogo,
			&st.PointsPerWin, &st.PointsPerDraw, &st.PointsPerLoss,
			&st.SetsBestOf, &st.PointsPerTieWin, &st.PointsPerTieLoss,
			&allowDraws, &mvpByGender, &tournamentStarted,
			&st.BracketQualifiers, &thirdPlace, &st.StandingsLegendText,
			&st.FanLayout, &prizesJSON, &slug)
	st.BracketThirdPlace = thirdPlace == 1
	st.AllowDraws = allowDraws == 1
	st.MvpByGender = mvpByGender == 1
	st.TournamentStarted = tournamentStarted == 1
	st.Prizes = decodeTournamentPrizes(prizesJSON)
	return &st, slug, err
}

func (s *Store) UpdateTASettings(ctx context.Context, eventID int64, st TASettings) error {
	thirdPlace := 0
	if st.BracketThirdPlace {
		thirdPlace = 1
	}
	allowDraws := 0
	if st.AllowDraws {
		allowDraws = 1
	}
	mvpByGender := 0
	if st.MvpByGender {
		mvpByGender = 1
	}
	tournamentStarted := 0
	if st.TournamentStarted {
		tournamentStarted = 1
	}
	prizesJSON, _ := json.Marshal(st.Prizes)
	_, err := s.db.ExecContext(ctx, `
		UPDATE events SET name=?, format=?, date_label=?, location=?, status_label=?, phase_label=?,
		                  logo_url=?, organizer_logo_url=?,
		                  points_per_win=?, points_per_draw=?, points_per_loss=?,
		                  sets_best_of=?, points_per_tie_win=?, points_per_tie_loss=?, allow_draws=?, mvp_by_gender=?,
		                  tournament_started=?,
		                  bracket_qualifiers=?, bracket_third_place=?, standings_legend_text=?,
		                  fan_layout=?, prizes_json=?
		WHERE id = ? AND type = 'tournament'`,
		st.Name, st.Format, st.DateLabel, st.Location, st.StatusLabel, st.PhaseLabel,
		st.Logo, st.OrganizerLogo,
		st.PointsPerWin, st.PointsPerDraw, st.PointsPerLoss,
		st.SetsBestOf, st.PointsPerTieWin, st.PointsPerTieLoss, allowDraws, mvpByGender,
		tournamentStarted,
		st.BracketQualifiers, thirdPlace, st.StandingsLegendText,
		st.FanLayout, string(prizesJSON), eventID)
	return err
}
