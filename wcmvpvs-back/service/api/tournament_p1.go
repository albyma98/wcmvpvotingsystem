package api

// ============================================================================
// TOURNAMENT P1 — endpoint pubblici (calendario, classifiche derivate,
// tabellone via stage) + operatori campo (magic link + PIN).
//
// Le classifiche NON hanno tabella: si derivano dalle partite concluse.
// Un solo scrittore (la console), zero divergenze possibili.
// Gli operatori campo non sono admin: token revocabile scopato su UN campo
// di UN evento, PIN a 6 cifre, zero account.
// ============================================================================

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
	"github.com/go-chi/chi/v5"
)

const opCookieName = "op_session"
const opSessionTTL = 48 * time.Hour

func registerTournamentP1Routes(rt *_router) {
	// Pubblici (tifosi)
	rt.router.Get("/v1/tournaments/{slug}/matches", rt.HandleTournamentMatches)
	rt.router.Get("/v1/tournaments/{slug}/standings", rt.HandleTournamentStandings)
	rt.router.Get("/v1/tournaments/{slug}/stream", rt.getTournamentStream) // SSE: push live

	// Gestione operatori (admin torneo)
	rt.router.Get("/v1/ta/{slug}/operators", rt.wrapTA(rt.taListOperators))
	rt.router.Post("/v1/ta/{slug}/operators", rt.wrapTA(rt.taCreateOperator))
	rt.router.Delete("/v1/ta/{slug}/operators/{id}", rt.wrapTA(rt.taDeleteOperator))

	// Console operatore (magic link)
	rt.router.Post("/v1/op/{token}/login", rt.opLogin)
	rt.router.Get("/v1/op/{token}/state", rt.wrapOp(rt.opState))
	rt.router.Post("/v1/op/{token}/matches/{id}/score", rt.wrapOp(rt.opScore))
}

// EnsureTournamentP1Tables: colonna stage + tabelle operatori (idempotente).
func (s *Store) EnsureTournamentP1Tables() error {
	if _, err := s.db.Exec(`ALTER TABLE matches ADD COLUMN stage TEXT NOT NULL DEFAULT ''`); err != nil &&
		!strings.Contains(err.Error(), "duplicate column") {
		return fmt.Errorf("matches stage: %w", err)
	}
	// Config gironi/bracket sull'evento + colonne bracket sulle partite (segnaposto
	// "Vincente QF1" quando la squadra non è ancora nota, e link di avanzamento
	// vincitore/perdente al turno successivo). Tutte idempotenti.
	for _, alter := range []string{
		`ALTER TABLE events ADD COLUMN points_per_win INTEGER NOT NULL DEFAULT 3`,
		`ALTER TABLE events ADD COLUMN points_per_draw INTEGER NOT NULL DEFAULT 1`,
		`ALTER TABLE events ADD COLUMN points_per_loss INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE events ADD COLUMN bracket_qualifiers INTEGER NOT NULL DEFAULT 2`,
		`ALTER TABLE events ADD COLUMN bracket_third_place INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE events ADD COLUMN fan_layout TEXT NOT NULL DEFAULT 'classic'`,
		`ALTER TABLE matches ADD COLUMN team_a_label TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE matches ADD COLUMN team_b_label TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE matches ADD COLUMN win_to_id TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE matches ADD COLUMN win_to_slot INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE matches ADD COLUMN lose_to_id TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE matches ADD COLUMN lose_to_slot INTEGER NOT NULL DEFAULT 0`,
	} {
		if _, err := s.db.Exec(alter); err != nil && !strings.Contains(err.Error(), "duplicate column") {
			return fmt.Errorf("events/matches bracket ensure (%s): %w", alter, err)
		}
	}
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS court_operators (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			event_id INTEGER NOT NULL,
			court TEXT NOT NULL,
			label TEXT NOT NULL DEFAULT '',
			token TEXT NOT NULL UNIQUE,
			pin TEXT NOT NULL,
			active INTEGER NOT NULL DEFAULT 1,
			created_at TEXT NOT NULL DEFAULT (datetime('now'))
		);`,
		`CREATE TABLE IF NOT EXISTS court_operator_sessions (
			token TEXT PRIMARY KEY,
			operator_id INTEGER NOT NULL,
			expires_at TEXT NOT NULL
		);`,
	}
	for _, q := range stmts {
		if _, err := s.db.Exec(q); err != nil {
			return fmt.Errorf("court operators tables: %w", err)
		}
	}
	return nil
}

// ============================ PUBBLICO: PARTITE ==============================

type PublicMatch struct {
	ID       string   `json:"id"`
	Court    string   `json:"court"`
	Time     string   `json:"time"`
	Status   string   `json:"status"`
	Stage    string   `json:"stage,omitempty"`
	Group    string   `json:"group,omitempty"`
	SetLabel string   `json:"setLabel,omitempty"`
	ScoreA   int      `json:"scoreA"`
	ScoreB   int      `json:"scoreB"`
	Sets     []string `json:"sets"`
	TeamA    string   `json:"teamA"`
	TeamB    string   `json:"teamB"`
}

func (s *Store) ListPublicMatches(ctx context.Context, slug string) ([]PublicMatch, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT m.id, m.court, m.scheduled_time, m.status, COALESCE(m.stage,''),
		       COALESCE(ta.group_name,''), m.set_label, m.score_a, m.score_b, m.sets_json,
		       COALESCE(NULLIF(ta.name,''), m.team_a_label, ''),
		       COALESCE(NULLIF(tb.name,''), m.team_b_label, '')
		FROM matches m
		JOIN events e ON e.id = m.event_id AND e.slug = ? AND e.type = 'tournament'
		LEFT JOIN tournament_teams ta ON ta.id = m.team_a_id
		LEFT JOIN tournament_teams tb ON tb.id = m.team_b_id
		ORDER BY m.scheduled_at, m.court`, slug)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]PublicMatch, 0, 24)
	for rows.Next() {
		var m PublicMatch
		var setsJSON string
		if err := rows.Scan(&m.ID, &m.Court, &m.Time, &m.Status, &m.Stage, &m.Group,
			&m.SetLabel, &m.ScoreA, &m.ScoreB, &setsJSON, &m.TeamA, &m.TeamB); err != nil {
			return nil, err
		}
		m.Sets = decodeSets(setsJSON)
		out = append(out, m)
	}
	return out, rows.Err()
}

func (rt *_router) HandleTournamentMatches(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	cacheKey := "matches:" + slug
	if cached, ok := rt.liveCache.Get(cacheKey); ok {
		writeJSON(w, http.StatusOK, cached)
		return
	}
	matches, err := rt.store.ListPublicMatches(r.Context(), slug)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	payload := map[string]interface{}{"matches": matches}
	rt.liveCache.Set(cacheKey, payload, 10*time.Second)
	writeJSON(w, http.StatusOK, payload)
}

// ============================ PUBBLICO: CLASSIFICHE ===========================

type StandingRow struct {
	TeamID  int64  `json:"teamId"`
	Team    string `json:"team"`
	Short   string `json:"short,omitempty"`
	Played  int    `json:"played"`
	Wins    int    `json:"wins"`
	Draws   int    `json:"draws"`
	Losses  int    `json:"losses"`
	Points  int    `json:"points"` // punti classifica: wins*perWin + draws*perDraw + losses*perLoss
	SetsW   int    `json:"setsWon"`
	SetsL   int    `json:"setsLost"`
	PointsW int    `json:"pointsWon"`
	PointsL int    `json:"pointsLost"`
}

type StandingsGroup struct {
	Group string        `json:"group"`
	Rows  []StandingRow `json:"rows"`
}

// ComputeStandings deriva le classifiche dalle sole partite CONCLUSE dei
// gironi (stage = ''). Ordinamento: punti classifica, quoziente set, quoziente
// punti. I punti per vittoria/sconfitta sono configurabili dal pannello admin.
func (s *Store) ComputeStandings(ctx context.Context, slug string) ([]StandingsGroup, error) {
	var perWin, perDraw, perLoss int
	if err := s.db.QueryRowContext(ctx, `
		SELECT COALESCE(points_per_win,3), COALESCE(points_per_draw,1), COALESCE(points_per_loss,0)
		FROM events WHERE slug = ? AND type = 'tournament'`, slug).Scan(&perWin, &perDraw, &perLoss); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errTANotFound
		}
		return nil, err
	}

	teams := map[int64]*StandingRow{}
	groupOf := map[int64]string{}
	rows, err := s.db.QueryContext(ctx, `
		SELECT tt.id, tt.name, tt.short_name, tt.group_name
		FROM tournament_teams tt
		JOIN events e ON e.id = tt.event_id AND e.slug = ? AND e.type = 'tournament'`, slug)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var id int64
		var name, short, group string
		if err := rows.Scan(&id, &name, &short, &group); err != nil {
			rows.Close()
			return nil, err
		}
		teams[id] = &StandingRow{TeamID: id, Team: name, Short: short}
		groupOf[id] = group
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	mrows, err := s.db.QueryContext(ctx, `
		SELECT m.team_a_id, m.team_b_id, m.score_a, m.score_b, m.sets_json
		FROM matches m
		JOIN events e ON e.id = m.event_id AND e.slug = ? AND e.type = 'tournament'
		WHERE m.status = 'finished' AND COALESCE(m.stage,'') = ''`, slug)
	if err != nil {
		return nil, err
	}
	defer mrows.Close()
	for mrows.Next() {
		var aID, bID int64
		var sa, sb int
		var setsJSON string
		if err := mrows.Scan(&aID, &bID, &sa, &sb, &setsJSON); err != nil {
			return nil, err
		}
		a, b := teams[aID], teams[bID]
		if a == nil || b == nil {
			continue
		}
		a.Played++
		b.Played++
		a.SetsW += sa
		a.SetsL += sb
		b.SetsW += sb
		b.SetsL += sa
		switch {
		case sa > sb:
			a.Wins++
			b.Losses++
		case sb > sa:
			b.Wins++
			a.Losses++
		default: // sa == sb: pareggio (dove il formato lo prevede)
			a.Draws++
			b.Draws++
		}
		for _, set := range decodeSets(setsJSON) {
			parts := strings.SplitN(set, "-", 2)
			if len(parts) != 2 {
				continue
			}
			pa, _ := strconv.Atoi(parts[0])
			pb, _ := strconv.Atoi(parts[1])
			a.PointsW += pa
			a.PointsL += pb
			b.PointsW += pb
			b.PointsL += pa
		}
	}
	if err := mrows.Err(); err != nil {
		return nil, err
	}

	byGroup := map[string][]StandingRow{}
	for id, row := range teams {
		row.Points = row.Wins*perWin + row.Draws*perDraw + row.Losses*perLoss
		g := groupOf[id]
		byGroup[g] = append(byGroup[g], *row)
	}
	ratio := func(w, l int) float64 {
		if l == 0 {
			return float64(w) * 1000 // imbattuto: sopra tutti a parità di vittorie
		}
		return float64(w) / float64(l)
	}
	out := make([]StandingsGroup, 0, len(byGroup))
	for g, rowsG := range byGroup {
		sort.SliceStable(rowsG, func(i, j int) bool {
			a, b := rowsG[i], rowsG[j]
			if a.Points != b.Points {
				return a.Points > b.Points
			}
			ra, rb := ratio(a.SetsW, a.SetsL), ratio(b.SetsW, b.SetsL)
			if ra != rb {
				return ra > rb
			}
			return ratio(a.PointsW, a.PointsL) > ratio(b.PointsW, b.PointsL)
		})
		out = append(out, StandingsGroup{Group: g, Rows: rowsG})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Group < out[j].Group })
	return out, nil
}

func (rt *_router) HandleTournamentStandings(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	cacheKey := "standings:" + slug
	if cached, ok := rt.liveCache.Get(cacheKey); ok {
		writeJSON(w, http.StatusOK, cached)
		return
	}
	groups, err := rt.store.ComputeStandings(r.Context(), slug)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	payload := map[string]interface{}{"groups": groups}
	rt.liveCache.Set(cacheKey, payload, 10*time.Second)
	writeJSON(w, http.StatusOK, payload)
}

// invalidateTournamentCaches: dopo ogni scrittura la vista tifosi si aggiorna.
// Oltre a bucare la cache TTL, notifica via SSE tutti i client connessi allo
// stream dell'evento (push real-time al posto del polling).
func (rt *_router) invalidateTournamentCaches(r *http.Request, eventID int64) {
	if _, slug, err := rt.store.GetTASettings(r.Context(), eventID); err == nil && slug != "" {
		rt.liveCache.Delete(slug)
		rt.liveCache.Delete("matches:" + slug)
		rt.liveCache.Delete("standings:" + slug)
	}
	rt.tournamentHub.Broadcast(int(eventID))
}

// ============================ OPERATORI CAMPO =================================

type CourtOperator struct {
	ID     int64  `json:"id"`
	Court  string `json:"court"`
	Label  string `json:"label"`
	Token  string `json:"token"`
	PIN    string `json:"pin"`
	Active bool   `json:"active"`
}

func (s *Store) ListOperators(ctx context.Context, eventID int64) ([]CourtOperator, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, court, label, token, pin, active
		FROM court_operators WHERE event_id = ? ORDER BY court, id`, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]CourtOperator, 0, 6)
	for rows.Next() {
		var o CourtOperator
		var act int
		if err := rows.Scan(&o.ID, &o.Court, &o.Label, &o.Token, &o.PIN, &act); err != nil {
			return nil, err
		}
		o.Active = act == 1
		out = append(out, o)
	}
	return out, rows.Err()
}

func (s *Store) CreateOperator(ctx context.Context, eventID int64, court, label string) (*CourtOperator, error) {
	buf := make([]byte, 12)
	if _, err := rand.Read(buf); err != nil {
		return nil, err
	}
	token := base64.RawURLEncoding.EncodeToString(buf)
	pinN, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return nil, err
	}
	pin := fmt.Sprintf("%06d", pinN.Int64())
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO court_operators (event_id, court, label, token, pin) VALUES (?, ?, ?, ?, ?)`,
		eventID, strings.TrimSpace(court), strings.TrimSpace(label), token, pin)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return &CourtOperator{ID: id, Court: court, Label: label, Token: token, PIN: pin, Active: true}, nil
}

func (s *Store) DeleteOperator(ctx context.Context, eventID, opID int64) error {
	if _, err := s.db.ExecContext(ctx, `
		DELETE FROM court_operators WHERE id = ? AND event_id = ?`, opID, eventID); err != nil {
		return err
	}
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM court_operator_sessions WHERE operator_id = ?`, opID)
	return err
}

type opInfo struct {
	ID      int64
	EventID int64
	Court   string
	PIN     string
	Active  bool
}

func (s *Store) GetOperatorByToken(ctx context.Context, token string) (*opInfo, error) {
	var o opInfo
	var act int
	err := s.db.QueryRowContext(ctx, `
		SELECT id, event_id, court, pin, active FROM court_operators WHERE token = ?`,
		token).Scan(&o.ID, &o.EventID, &o.Court, &o.PIN, &act)
	if err != nil {
		return nil, err
	}
	o.Active = act == 1
	return &o, nil
}

func (s *Store) CreateOpSession(ctx context.Context, operatorID int64) (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	token := base64.RawURLEncoding.EncodeToString(buf)
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO court_operator_sessions (token, operator_id, expires_at) VALUES (?, ?, ?)`,
		token, operatorID, time.Now().Add(opSessionTTL).UTC().Format(time.RFC3339))
	return token, err
}

func (s *Store) GetOpSession(ctx context.Context, token string) (operatorID int64, ok bool) {
	var expires string
	if err := s.db.QueryRowContext(ctx, `
		SELECT operator_id, expires_at FROM court_operator_sessions WHERE token = ?`,
		token).Scan(&operatorID, &expires); err != nil {
		return 0, false
	}
	exp, err := time.Parse(time.RFC3339, expires)
	if err != nil || time.Now().After(exp) {
		_, _ = s.db.ExecContext(ctx, `DELETE FROM court_operator_sessions WHERE token = ?`, token)
		return 0, false
	}
	return operatorID, true
}

// --- handler admin (gestione operatori) ---

func (rt *_router) taListOperators(w http.ResponseWriter, r *http.Request, eventID int64) {
	ops, err := rt.store.ListOperators(r.Context(), eventID)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"operators": ops})
}

func (rt *_router) taCreateOperator(w http.ResponseWriter, r *http.Request, eventID int64) {
	var body struct {
		Court string `json:"court"`
		Label string `json:"label"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Court) == "" {
		http.Error(w, `{"error":"bad_input"}`, http.StatusBadRequest)
		return
	}
	op, err := rt.store.CreateOperator(r.Context(), eventID, body.Court, body.Label)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"operator": op,
		"path":     "/op/" + op.Token,
	})
}

func (rt *_router) taDeleteOperator(w http.ResponseWriter, r *http.Request, eventID int64) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err := rt.store.DeleteOperator(r.Context(), eventID, id); err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true})
}

// --- console operatore ---

func (rt *_router) opLogin(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	var body struct {
		PIN string `json:"pin"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, `{"error":"bad_json"}`, http.StatusBadRequest)
		return
	}
	op, err := rt.store.GetOperatorByToken(r.Context(), token)
	if err != nil || !op.Active || op.PIN != strings.TrimSpace(body.PIN) {
		http.Error(w, `{"error":"invalid_pin"}`, http.StatusUnauthorized)
		return
	}
	sess, err := rt.store.CreateOpSession(r.Context(), op.ID)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	secure := r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
	http.SetCookie(w, &http.Cookie{
		Name: opCookieName, Value: sess, Path: "/",
		MaxAge: int(opSessionTTL.Seconds()), HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode,
	})
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true, "court": op.Court})
}

type opHandler func(w http.ResponseWriter, r *http.Request, op *opInfo)

// wrapOp: sessione valida + sessione appartenente al token nel path +
// operatore ancora attivo (la revoca è istantanea).
func (rt *_router) wrapOp(fn opHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(opCookieName)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		operatorID, ok := rt.store.GetOpSession(r.Context(), c.Value)
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		op, err := rt.store.GetOperatorByToken(r.Context(), chi.URLParam(r, "token"))
		if err != nil || !op.Active || op.ID != operatorID {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		fn(w, r, op)
	}
}

func (rt *_router) opState(w http.ResponseWriter, r *http.Request, op *opInfo) {
	settings, slug, err := rt.store.GetTASettings(r.Context(), op.EventID)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	matches, err := rt.store.ListTAMatches(r.Context(), op.EventID)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	mine := make([]TAMatch, 0, 8)
	for _, m := range matches {
		if strings.EqualFold(m.Court, op.Court) {
			mine = append(mine, m)
		}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"tournament": settings.Name, "slug": slug, "court": op.Court, "matches": mine,
	})
}

func (rt *_router) opScore(w http.ResponseWriter, r *http.Request, op *opInfo) {
	matchID := chi.URLParam(r, "id")
	var body struct {
		Action string `json:"action"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, `{"error":"bad_json"}`, http.StatusBadRequest)
		return
	}
	// L'operatore tocca SOLO le partite del suo campo.
	matches, err := rt.store.ListTAMatches(r.Context(), op.EventID)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	found := false
	for _, m := range matches {
		if m.ID == matchID && strings.EqualFold(m.Court, op.Court) {
			found = true
			break
		}
	}
	if !found {
		http.Error(w, `{"error":"not_your_court"}`, http.StatusForbidden)
		return
	}
	if err := rt.store.ApplyScoreAction(r.Context(), op.EventID, matchID, body.Action); err != nil {
		if err.Error() == "set_tied" {
			http.Error(w, `{"error":"set_tied"}`, http.StatusBadRequest)
			return
		}
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	rt.invalidateTournamentCaches(r, op.EventID)
	state, _ := rt.store.ListTAMatches(r.Context(), op.EventID)
	mine := make([]TAMatch, 0, 8)
	for _, m := range state {
		if strings.EqualFold(m.Court, op.Court) {
			mine = append(mine, m)
		}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true, "matches": mine})
}

// Nota: reqcontext importato per coerenza di firma con gli altri file api.
var _ = func(ctx reqcontext.RequestContext) {}
