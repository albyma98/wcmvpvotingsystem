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
	rt.router.Post("/v1/op/{token}/bracket/decision", rt.wrapOp(rt.opBracketDecision))
	rt.router.Post("/v1/op/{token}/teams", rt.wrapOp(rt.opCreateTeam))
	rt.router.Put("/v1/op/{token}/teams/{id}", rt.wrapOp(rt.opUpdateTeam))
	rt.router.Put("/v1/op/{token}/teams/{id}/players", rt.wrapOp(rt.opSetTeamPlayers))
	rt.router.Delete("/v1/op/{token}/teams/{id}", rt.wrapOp(rt.opDeleteTeam))
	rt.router.Post("/v1/op/{token}/calendar", rt.wrapOp(rt.opCreateMatch))
	rt.router.Put("/v1/op/{token}/calendar/{id}", rt.wrapOp(rt.opUpdateMatch))
	rt.router.Delete("/v1/op/{token}/calendar/{id}", rt.wrapOp(rt.opDeleteMatch))
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
		// Formula set: 3 = al meglio dei 3 (2 su 3), 5 = al meglio dei 5 (3 su 5).
		// points_per_tie_win/loss = punti a chi vince/perde al tie-break (set decisivo).
		`ALTER TABLE events ADD COLUMN sets_best_of INTEGER NOT NULL DEFAULT 3`,
		`ALTER TABLE events ADD COLUMN points_per_tie_win INTEGER NOT NULL DEFAULT 2`,
		`ALTER TABLE events ADD COLUMN points_per_tie_loss INTEGER NOT NULL DEFAULT 1`,
		// allow_draws: 1 = i pareggi (set pari) contano in classifica (colonna N),
		// 0 = torneo senza pareggi (colonna N nascosta lato tifoso).
		`ALTER TABLE events ADD COLUMN allow_draws INTEGER NOT NULL DEFAULT 1`,
		// mvp_by_gender: 1 = votazione MVP separata uomo/donna (2 voti per device),
		// 0 = MVP unico indipendente dal sesso (1 voto per device).
		`ALTER TABLE events ADD COLUMN mvp_by_gender INTEGER NOT NULL DEFAULT 1`,
		`ALTER TABLE events ADD COLUMN bracket_qualifiers INTEGER NOT NULL DEFAULT 2`,
		`ALTER TABLE events ADD COLUMN bracket_third_place INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE events ADD COLUMN standings_legend_text TEXT NOT NULL DEFAULT 'Primi 2 di ogni girone alla fase finale · Ordinamento: punti, quoziente set, quoziente punti'`,
		`ALTER TABLE events ADD COLUMN fan_layout TEXT NOT NULL DEFAULT 'classic'`,
		`ALTER TABLE events ADD COLUMN bracket_auto_generate_at TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE events ADD COLUMN bracket_auto_generate_state INTEGER NOT NULL DEFAULT 0`,
		// Premi del torneo (JSON): 1°/2°/3° classificato + MVP uomo/donna scelti
		// da organizzatori e pubblico. Mostrati nella modale "Premi" (🏆) del layout Sunset.
		`ALTER TABLE events ADD COLUMN prizes_json TEXT`,
		`ALTER TABLE matches ADD COLUMN team_a_label TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE matches ADD COLUMN team_b_label TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE matches ADD COLUMN win_to_id TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE matches ADD COLUMN win_to_slot INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE matches ADD COLUMN lose_to_id TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE matches ADD COLUMN lose_to_slot INTEGER NOT NULL DEFAULT 0`,
		// is_anchor: la partita "inizio torneo". Il suo orario è il taglio del
		// ciclo cronologico: le partite con orario >= àncora restano nel giorno,
		// quelle con orario < àncora (es. dopo mezzanotte) scivolano in coda.
		// Al più una per evento (garantito da SetTAMatchAnchor).
		`ALTER TABLE matches ADD COLUMN is_anchor INTEGER NOT NULL DEFAULT 0`,
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
			role TEXT NOT NULL DEFAULT 'court',
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
	if _, err := s.db.Exec(`ALTER TABLE court_operators ADD COLUMN role TEXT NOT NULL DEFAULT 'court'`); err != nil &&
		!strings.Contains(err.Error(), "duplicate column") {
		return fmt.Errorf("court operators role: %w", err)
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
	ScoreA   int      `json:"scoreA"` // set vinti
	ScoreB   int      `json:"scoreB"`
	CurA     int      `json:"curA"` // punti del set in corso (per il live nel calendario)
	CurB     int      `json:"curB"`
	Sets     []string `json:"sets"`
	TeamA    string   `json:"teamA"`
	TeamB    string   `json:"teamB"`
}

func (s *Store) ListPublicMatches(ctx context.Context, slug string) ([]PublicMatch, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT m.id, m.court, m.scheduled_time, m.status, COALESCE(m.stage,''),
		       COALESCE(ta.group_name,''), m.set_label, m.score_a, m.score_b, m.cur_a, m.cur_b, m.sets_json,
		       COALESCE(NULLIF(ta.name,''), m.team_a_label, ''),
		       COALESCE(NULLIF(tb.name,''), m.team_b_label, '')
		FROM matches m
		JOIN events e ON e.id = m.event_id AND e.slug = ? AND e.type = 'tournament'
		LEFT JOIN tournament_teams ta ON ta.id = m.team_a_id
		LEFT JOIN tournament_teams tb ON tb.id = m.team_b_id
		ORDER BY
		  CASE WHEN m.scheduled_time >= COALESCE(
		         (SELECT a.scheduled_time FROM matches a
		            WHERE a.event_id = m.event_id AND a.is_anchor = 1 LIMIT 1), '')
		       THEN 0 ELSE 1 END,
		  m.scheduled_time, m.court`, slug)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]PublicMatch, 0, 24)
	for rows.Next() {
		var m PublicMatch
		var setsJSON string
		if err := rows.Scan(&m.ID, &m.Court, &m.Time, &m.Status, &m.Stage, &m.Group,
			&m.SetLabel, &m.ScoreA, &m.ScoreB, &m.CurA, &m.CurB, &setsJSON, &m.TeamA, &m.TeamB); err != nil {
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
	TeamID    int64  `json:"teamId"`
	Team      string `json:"team"`
	Short     string `json:"short,omitempty"`
	Played    int    `json:"played"`
	Wins      int    `json:"wins"`
	TieWins   int    `json:"tieWins,omitempty"` // vittorie al tie-break (set decisivo)
	Draws     int    `json:"draws"`
	Losses    int    `json:"losses"`
	TieLosses int    `json:"tieLosses,omitempty"` // sconfitte al tie-break (set decisivo)
	Points    int    `json:"points"`              // vedi ComputeStandings (tie-break separati da win/loss pieni)
	SetsW     int    `json:"setsWon"`
	SetsL     int    `json:"setsLost"`
	PointsW   int    `json:"pointsWon"`
	PointsL   int    `json:"pointsLost"`
}

type StandingsGroup struct {
	Group string        `json:"group"`
	Rows  []StandingRow `json:"rows"`
}

// ComputeStandings deriva le classifiche dalle sole partite CONCLUSE dei
// gironi (stage = ”). Ordinamento: punti classifica, quoziente set, quoziente
// punti. I punti per vittoria/sconfitta sono configurabili dal pannello admin.
// Ritorna anche allowDraws: se il torneo non ammette pareggi, le partite con set
// pari NON contano come pareggio e la colonna "N" è nascosta lato tifoso.
func (s *Store) ComputeStandings(ctx context.Context, slug string) ([]StandingsGroup, bool, error) {
	var perWin, perDraw, perLoss, perTieWin, perTieLoss, bestOf, allowDrawsInt int
	if err := s.db.QueryRowContext(ctx, `
		SELECT COALESCE(points_per_win,3), COALESCE(points_per_draw,1), COALESCE(points_per_loss,0),
		       COALESCE(points_per_tie_win,2), COALESCE(points_per_tie_loss,1), COALESCE(sets_best_of,3),
		       COALESCE(allow_draws,1)
		FROM events WHERE slug = ? AND type = 'tournament'`, slug).
		Scan(&perWin, &perDraw, &perLoss, &perTieWin, &perTieLoss, &bestOf, &allowDrawsInt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, false, errTANotFound
		}
		return nil, false, err
	}
	allowDraws := allowDrawsInt == 1
	// Set necessari per vincere: 2 su 3, 3 su 5. Il tie-break è il set decisivo:
	// vittoria "alla distanza" = vincitore a setsToWin, perdente a setsToWin-1.
	setsToWin := (bestOf + 1) / 2

	teams := map[int64]*StandingRow{}
	groupOf := map[int64]string{}
	rows, err := s.db.QueryContext(ctx, `
		SELECT tt.id, tt.name, tt.short_name, tt.group_name
		FROM tournament_teams tt
		JOIN events e ON e.id = tt.event_id AND e.slug = ? AND e.type = 'tournament'`, slug)
	if err != nil {
		return nil, false, err
	}
	for rows.Next() {
		var id int64
		var name, short, group string
		if err := rows.Scan(&id, &name, &short, &group); err != nil {
			rows.Close()
			return nil, false, err
		}
		teams[id] = &StandingRow{TeamID: id, Team: name, Short: short}
		groupOf[id] = group
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, false, err
	}

	mrows, err := s.db.QueryContext(ctx, `
		SELECT m.team_a_id, m.team_b_id, m.score_a, m.score_b, m.sets_json
		FROM matches m
		JOIN events e ON e.id = m.event_id AND e.slug = ? AND e.type = 'tournament'
		WHERE m.status = 'finished' AND COALESCE(m.stage,'') = ''`, slug)
	if err != nil {
		return nil, false, err
	}
	defer mrows.Close()
	for mrows.Next() {
		var aID, bID int64
		var sa, sb int
		var setsJSON string
		if err := mrows.Scan(&aID, &bID, &sa, &sb, &setsJSON); err != nil {
			return nil, false, err
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
		// Tie-break = vittoria al set decisivo: il vincitore arriva a setsToWin
		// e il perdente è a setsToWin-1 (2-1 nel 2 su 3, 3-2 nel 3 su 5).
		hi, lo := sa, sb
		if sb > sa {
			hi, lo = sb, sa
		}
		tieBreak := hi == setsToWin && lo == setsToWin-1
		switch {
		case sa > sb:
			a.Wins++
			b.Losses++
			if tieBreak {
				a.TieWins++
				b.TieLosses++
			}
		case sb > sa:
			b.Wins++
			a.Losses++
			if tieBreak {
				b.TieWins++
				a.TieLosses++
			}
		default: // sa == sb: pareggio, solo se il torneo li ammette
			if allowDraws {
				a.Draws++
				b.Draws++
			}
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
		return nil, false, err
	}

	byGroup := map[string][]StandingRow{}
	for id, row := range teams {
		// Vittorie/sconfitte al tie-break valgono perTieWin/perTieLoss; le "piene" perWin/perLoss.
		row.Points = (row.Wins-row.TieWins)*perWin + row.TieWins*perTieWin +
			row.Draws*perDraw +
			(row.Losses-row.TieLosses)*perLoss + row.TieLosses*perTieLoss
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
			rpa, rpb := ratio(a.PointsW, a.PointsL), ratio(b.PointsW, b.PointsL)
			if rpa != rpb {
				return rpa > rpb
			}
			// Ultimo criterio deterministico: evita che squadre perfettamente
			// pari si scambino per l'ordine casuale di iterazione delle mappe Go.
			nameA, nameB := strings.ToLower(a.Team), strings.ToLower(b.Team)
			if nameA != nameB {
				return nameA < nameB
			}
			return a.TeamID < b.TeamID
		})
		out = append(out, StandingsGroup{Group: g, Rows: rowsG})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Group < out[j].Group })
	return out, allowDraws, nil
}

func (rt *_router) HandleTournamentStandings(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	cacheKey := "standings:" + slug
	if cached, ok := rt.liveCache.Get(cacheKey); ok {
		writeJSON(w, http.StatusOK, cached)
		return
	}
	groups, allowDraws, err := rt.store.ComputeStandings(r.Context(), slug)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	qualifiers, legendText, err := rt.store.GetStandingsDisplaySettings(r.Context(), slug)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	payload := map[string]interface{}{
		"groups": groups, "allowDraws": allowDraws,
		"qualifiersPerGroup": qualifiers, "legendText": legendText,
	}
	rt.liveCache.Set(cacheKey, payload, 10*time.Second)
	writeJSON(w, http.StatusOK, payload)
}

func (s *Store) GetStandingsDisplaySettings(ctx context.Context, slug string) (int, string, error) {
	const defaultLegend = "Primi 2 di ogni girone alla fase finale · Ordinamento: punti, quoziente set, quoziente punti"
	var qualifiers int
	var legendText string
	err := s.db.QueryRowContext(ctx, `
		SELECT COALESCE(bracket_qualifiers,2), COALESCE(standings_legend_text, ?)
		FROM events WHERE slug = ? AND type = 'tournament'`,
		defaultLegend, slug).Scan(&qualifiers, &legendText)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, "", errTANotFound
	}
	if err != nil {
		return 0, "", err
	}
	if qualifiers < 1 {
		qualifiers = 1
	}
	if qualifiers > 8 {
		qualifiers = 8
	}
	return qualifiers, legendText, nil
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
	rt.tournamentHub.BroadcastEvent(int(eventID), "standings")
}

// invalidateTournamentScoreCaches aggiorna sempre live e calendario, ma
// invalida/notifica la classifica soltanto quando una partita viene conclusa
// o riaperta. I singoli punti e i set intermedi non cambiano la classifica.
func (rt *_router) invalidateTournamentScoreCaches(r *http.Request, eventID int64, standingsChanged bool) {
	if _, slug, err := rt.store.GetTASettings(r.Context(), eventID); err == nil && slug != "" {
		rt.liveCache.Delete(slug)
		rt.liveCache.Delete("matches:" + slug)
		if standingsChanged {
			rt.liveCache.Delete("standings:" + slug)
		}
	}
	eventType := "score"
	if standingsChanged {
		eventType = "standings"
	}
	rt.tournamentHub.BroadcastEvent(int(eventID), eventType)
}

func (rt *_router) invalidateTournamentBracketCaches(ctx context.Context, eventID int64) {
	if _, slug, err := rt.store.GetTASettings(ctx, eventID); err == nil && slug != "" {
		rt.liveCache.Delete(slug)
		rt.liveCache.Delete("matches:" + slug)
		rt.liveCache.Delete("standings:" + slug)
	}
	rt.tournamentHub.BroadcastEvent(int(eventID), "standings")
}

func (rt *_router) generatePendingBracket(ctx context.Context, eventID int64, requireDue bool) (bool, error) {
	claimed, err := rt.store.ClaimBracketAutoGeneration(ctx, eventID, requireDue)
	if err != nil || !claimed {
		return false, err
	}
	st, _, err := rt.store.GetTASettings(ctx, eventID)
	if err == nil {
		_, err = rt.store.GenerateBracket(ctx, eventID, st.BracketQualifiers, st.BracketThirdPlace)
	}
	rt.store.FinishClaimedBracketGeneration(ctx, eventID, err == nil)
	if err != nil {
		return false, err
	}
	rt.invalidateTournamentBracketCaches(ctx, eventID)
	return true, nil
}

func (rt *_router) armBracketAutoGeneration(eventID int64, prompt BracketAutoPrompt) {
	if !prompt.Pending {
		return
	}
	delay := time.Until(time.UnixMilli(prompt.DeadlineMs))
	if delay < 0 {
		delay = 0
	}
	time.AfterFunc(delay, func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		if _, err := rt.generatePendingBracket(ctx, eventID, true); err != nil {
			rt.baseLogger.WithError(err).WithField("eventID", eventID).Error("automatic bracket generation failed")
		}
	})
}

func (rt *_router) resumeBracketAutoGenerations() {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		prompts, err := rt.store.ListBracketAutoPrompts(ctx)
		if err != nil {
			rt.baseLogger.WithError(err).Error("cannot resume automatic bracket generations")
			return
		}
		for eventID, prompt := range prompts {
			rt.armBracketAutoGeneration(eventID, prompt)
		}
	}()
}

func (rt *_router) maybeScheduleBracketAutoGeneration(ctx context.Context, eventID int64) BracketAutoPrompt {
	prompt, created, err := rt.store.ScheduleBracketAutoGeneration(ctx, eventID)
	if err != nil {
		rt.baseLogger.WithError(err).WithField("eventID", eventID).Error("cannot schedule automatic bracket generation")
		return BracketAutoPrompt{}
	}
	if created {
		rt.armBracketAutoGeneration(eventID, prompt)
	}
	return prompt
}

// ============================ OPERATORI CAMPO =================================

type CourtOperator struct {
	ID     int64  `json:"id"`
	Court  string `json:"court"`
	Label  string `json:"label"`
	Role   string `json:"role"`
	Token  string `json:"token"`
	PIN    string `json:"pin"`
	Active bool   `json:"active"`
}

func (s *Store) ListOperators(ctx context.Context, eventID int64) ([]CourtOperator, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, court, label, COALESCE(role,'court'), token, pin, active
		FROM court_operators WHERE event_id = ? ORDER BY role, court, id`, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]CourtOperator, 0, 6)
	for rows.Next() {
		var o CourtOperator
		var act int
		if err := rows.Scan(&o.ID, &o.Court, &o.Label, &o.Role, &o.Token, &o.PIN, &act); err != nil {
			return nil, err
		}
		o.Active = act == 1
		out = append(out, o)
	}
	return out, rows.Err()
}

func (s *Store) CreateOperator(ctx context.Context, eventID int64, role, court, label string) (*CourtOperator, error) {
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
		INSERT INTO court_operators (event_id, court, label, role, token, pin) VALUES (?, ?, ?, ?, ?, ?)`,
		eventID, strings.TrimSpace(court), strings.TrimSpace(label), role, token, pin)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return &CourtOperator{ID: id, Court: court, Label: label, Role: role, Token: token, PIN: pin, Active: true}, nil
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
	Role    string
	PIN     string
	Active  bool
}

func (s *Store) GetOperatorByToken(ctx context.Context, token string) (*opInfo, error) {
	var o opInfo
	var act int
	err := s.db.QueryRowContext(ctx, `
		SELECT id, event_id, court, COALESCE(role,'court'), pin, active FROM court_operators WHERE token = ?`,
		token).Scan(&o.ID, &o.EventID, &o.Court, &o.Role, &o.PIN, &act)
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
		Role  string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, `{"error":"bad_input"}`, http.StatusBadRequest)
		return
	}
	role := strings.ToLower(strings.TrimSpace(body.Role))
	if role == "" {
		role = "court"
	}
	if role != "court" && role != "teams" && role != "calendar" && role != "mvp" ||
		role == "court" && strings.TrimSpace(body.Court) == "" {
		http.Error(w, `{"error":"bad_input"}`, http.StatusBadRequest)
		return
	}
	op, err := rt.store.CreateOperator(r.Context(), eventID, role, body.Court, body.Label)
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
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true, "court": op.Court, "role": op.Role})
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
	payload := map[string]interface{}{
		"tournament": settings.Name, "slug": slug, "court": op.Court, "role": op.Role,
	}
	switch op.Role {
	case "teams":
		teams, err := rt.store.ListTATeams(r.Context(), op.EventID)
		if err != nil {
			http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
			return
		}
		payload["teams"] = teams
	case "calendar":
		teams, teamErr := rt.store.ListTATeams(r.Context(), op.EventID)
		matches, matchErr := rt.store.ListTAMatches(r.Context(), op.EventID)
		if teamErr != nil || matchErr != nil {
			http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
			return
		}
		payload["teams"], payload["matches"], payload["courts"] = teams, matches, settings.Courts
	case "mvp":
		board, err := rt.store.GetMVPResults(r.Context(), op.EventID)
		if err != nil {
			http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
			return
		}
		payload["mvp"] = board
	default:
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
		prompt := rt.store.GetBracketAutoPrompt(r.Context(), op.EventID)
		if prompt.Pending {
			rt.armBracketAutoGeneration(op.EventID, prompt)
		}
		payload["matches"], payload["bracketPrompt"] = mine, prompt
	}
	writeJSON(w, http.StatusOK, payload)
}

func (rt *_router) opScore(w http.ResponseWriter, r *http.Request, op *opInfo) {
	if op.Role != "court" {
		http.Error(w, `{"error":"wrong_role"}`, http.StatusForbidden)
		return
	}
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
		if err.Error() == "set_tied" || err.Error() == "no_closed_set" ||
			err.Error() == "current_set_started" {
			http.Error(w, fmt.Sprintf(`{"error":%q}`, err.Error()), http.StatusBadRequest)
			return
		}
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	if body.Action == "reopen" {
		rt.store.ResetBracketAutoGeneration(r.Context(), op.EventID)
	}
	prompt := BracketAutoPrompt{}
	if body.Action == "finish" {
		prompt = rt.maybeScheduleBracketAutoGeneration(r.Context(), op.EventID)
	}
	rt.invalidateTournamentScoreCaches(r, op.EventID, body.Action == "finish" || body.Action == "reopen")
	state, _ := rt.store.ListTAMatches(r.Context(), op.EventID)
	mine := make([]TAMatch, 0, 8)
	for _, m := range state {
		if strings.EqualFold(m.Court, op.Court) {
			mine = append(mine, m)
		}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true, "matches": mine, "bracketPrompt": prompt})
}

func (rt *_router) opBracketDecision(w http.ResponseWriter, r *http.Request, op *opInfo) {
	if op.Role != "court" {
		http.Error(w, `{"error":"wrong_role"}`, http.StatusForbidden)
		return
	}
	var body struct {
		Action string `json:"action"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, `{"error":"bad_json"}`, http.StatusBadRequest)
		return
	}
	switch body.Action {
	case "decline":
		if err := rt.store.DeclineBracketAutoGeneration(r.Context(), op.EventID); err != nil {
			http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
			return
		}
		rt.tournamentHub.BroadcastEvent(int(op.EventID), "score")
		writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true, "generated": false})
	case "generate":
		generated, err := rt.generatePendingBracket(r.Context(), op.EventID, false)
		if err != nil {
			http.Error(w, `{"error":"generation_failed"}`, http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true, "generated": generated})
	default:
		http.Error(w, `{"error":"bad_action"}`, http.StatusBadRequest)
	}
}

func (rt *_router) opCreateTeam(w http.ResponseWriter, r *http.Request, op *opInfo) {
	if op.Role != "teams" {
		http.Error(w, `{"error":"wrong_role"}`, http.StatusForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxSponsorLogoBytes+65536)
	var team TATeam
	if err := json.NewDecoder(r.Body).Decode(&team); err != nil || strings.TrimSpace(team.Name) == "" {
		http.Error(w, `{"error":"bad_input"}`, http.StatusBadRequest)
		return
	}
	logo, err := sanitizeSponsorLogo(team.LogoURL)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":%q}`, err.Error()), http.StatusBadRequest)
		return
	}
	team.LogoURL = logo
	if _, err := rt.store.InsertTATeams(r.Context(), op.EventID, []TATeam{team}); err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	rt.invalidateTournamentBracketCaches(r.Context(), op.EventID)
	writeJSON(w, http.StatusCreated, map[string]interface{}{"ok": true})
}

func (rt *_router) opUpdateTeam(w http.ResponseWriter, r *http.Request, op *opInfo) {
	if op.Role != "teams" {
		http.Error(w, `{"error":"wrong_role"}`, http.StatusForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxSponsorLogoBytes+4096)
	teamID, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var team TATeam
	if err := json.NewDecoder(r.Body).Decode(&team); err != nil {
		http.Error(w, `{"error":"bad_json"}`, http.StatusBadRequest)
		return
	}
	logo, err := sanitizeSponsorLogo(team.LogoURL)
	if err == nil {
		err = rt.store.UpdateTATeam(r.Context(), op.EventID, teamID,
			team.Name, team.ShortName, team.City, logo, team.GroupName)
	}
	if err != nil {
		http.Error(w, `{"error":"update_failed"}`, http.StatusBadRequest)
		return
	}
	rt.invalidateTournamentBracketCaches(r.Context(), op.EventID)
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true})
}

func (rt *_router) opSetTeamPlayers(w http.ResponseWriter, r *http.Request, op *opInfo) {
	if op.Role != "teams" {
		http.Error(w, `{"error":"wrong_role"}`, http.StatusForbidden)
		return
	}
	teamID, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var body struct {
		Players []TAPlayer `json:"players"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil ||
		rt.store.ReplaceTAPlayers(r.Context(), op.EventID, teamID, body.Players) != nil {
		http.Error(w, `{"error":"update_failed"}`, http.StatusBadRequest)
		return
	}
	rt.invalidateTournamentBracketCaches(r.Context(), op.EventID)
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true})
}

func (rt *_router) opDeleteTeam(w http.ResponseWriter, r *http.Request, op *opInfo) {
	if op.Role != "teams" {
		http.Error(w, `{"error":"wrong_role"}`, http.StatusForbidden)
		return
	}
	teamID, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err := rt.store.DeleteTATeam(r.Context(), op.EventID, teamID); err != nil {
		status := http.StatusBadRequest
		if err.Error() == "team_in_use" {
			status = http.StatusConflict
		}
		http.Error(w, `{"error":"team_in_use"}`, status)
		return
	}
	rt.invalidateTournamentBracketCaches(r.Context(), op.EventID)
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true})
}

type opMatchInput struct {
	Court       string `json:"court"`
	Time        string `json:"time"`
	ScheduledAt string `json:"scheduledAt"`
	Stage       string `json:"stage"`
	TeamAID     int64  `json:"teamAId"`
	TeamBID     int64  `json:"teamBId"`
}

func decodeOpMatchInput(w http.ResponseWriter, r *http.Request) (opMatchInput, bool) {
	var body opMatchInput
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Court) == "" ||
		body.TeamAID == 0 || body.TeamBID == 0 || body.TeamAID == body.TeamBID {
		http.Error(w, `{"error":"bad_input"}`, http.StatusBadRequest)
		return body, false
	}
	return body, true
}

func (rt *_router) opCreateMatch(w http.ResponseWriter, r *http.Request, op *opInfo) {
	if op.Role != "calendar" {
		http.Error(w, `{"error":"wrong_role"}`, http.StatusForbidden)
		return
	}
	body, ok := decodeOpMatchInput(w, r)
	if !ok {
		return
	}
	id, err := rt.store.CreateTAMatch(r.Context(), op.EventID, body.Court, body.Time,
		body.ScheduledAt, body.Stage, body.TeamAID, body.TeamBID)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	rt.invalidateTournamentBracketCaches(r.Context(), op.EventID)
	writeJSON(w, http.StatusCreated, map[string]interface{}{"ok": true, "id": id})
}

func (rt *_router) opUpdateMatch(w http.ResponseWriter, r *http.Request, op *opInfo) {
	if op.Role != "calendar" {
		http.Error(w, `{"error":"wrong_role"}`, http.StatusForbidden)
		return
	}
	body, ok := decodeOpMatchInput(w, r)
	if !ok {
		return
	}
	if err := rt.store.UpdateTAMatch(r.Context(), op.EventID, chi.URLParam(r, "id"),
		body.Court, body.Time, body.ScheduledAt, body.Stage, body.TeamAID, body.TeamBID); err != nil {
		http.Error(w, `{"error":"update_failed"}`, http.StatusBadRequest)
		return
	}
	rt.invalidateTournamentBracketCaches(r.Context(), op.EventID)
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true})
}

func (rt *_router) opDeleteMatch(w http.ResponseWriter, r *http.Request, op *opInfo) {
	if op.Role != "calendar" {
		http.Error(w, `{"error":"wrong_role"}`, http.StatusForbidden)
		return
	}
	if err := rt.store.DeleteTAMatch(r.Context(), op.EventID, chi.URLParam(r, "id")); err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	rt.invalidateTournamentBracketCaches(r.Context(), op.EventID)
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true})
}

// Nota: reqcontext importato per coerenza di firma con gli altri file api.
var _ = func(ctx reqcontext.RequestContext) {}
