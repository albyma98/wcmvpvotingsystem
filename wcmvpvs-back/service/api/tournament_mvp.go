package api

// ============================================================================
// TOURNAMENT MVP — votazione del pubblico (tifosi).
// Il tifoso apre la tile "Vota MVP" e vota UN giocatore fra tutte le squadre
// partecipanti. Voto 1-per-device (X-Device-ID), modificabile (upsert): il
// device può cambiare la sua preferenza, il conteggio riflette sempre lo stato
// corrente. Le candidature sono la rosa (tournament_players) inserita in admin.
// Endpoint pubblici, nessuna auth: GET candidati+conteggi, POST voto.
// ============================================================================

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

func registerTournamentMVPRoutes(rt *_router) {
	rt.router.Get("/v1/tournaments/{slug}/mvp", rt.HandleTournamentMVP)
	rt.router.Post("/v1/tournaments/{slug}/mvp/vote", rt.HandleTournamentMVPVote)
}

// EnsureTournamentMVPTables crea/riconcilia la tabella dei voti MVP (idempotente).
// I DB con la vecchia feature "Tornei" avevano già una `tournament_mvp_votes`
// legacy (colonna `tournament_id`, niente `event_id`): stesso trattamento di
// teams/sponsors/players → reconcile allo schema canonico.
func (s *Store) EnsureTournamentMVPTables() error {
	const createSQL = `CREATE TABLE tournament_mvp_votes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		event_id INTEGER NOT NULL,
		player_id INTEGER NOT NULL,
		device_id TEXT NOT NULL,
		created_at TEXT NOT NULL DEFAULT (datetime('now')),
		UNIQUE(event_id, device_id)
	)`
	if _, err := s.db.Exec(`CREATE TABLE IF NOT EXISTS tournament_mvp_votes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		event_id INTEGER NOT NULL,
		player_id INTEGER NOT NULL,
		device_id TEXT NOT NULL,
		created_at TEXT NOT NULL DEFAULT (datetime('now')),
		UNIQUE(event_id, device_id)
	)`); err != nil {
		return err
	}
	return s.reconcileTournamentTable("tournament_mvp_votes", createSQL,
		[]string{"id", "event_id", "player_id", "device_id", "created_at"})
}

// --- Store ------------------------------------------------------------------

// MVPCandidate = un giocatore votabile con il suo conteggio voti corrente.
type MVPCandidate struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Votes int    `json:"votes"`
}

// MVPTeam = una squadra partecipante con la sua rosa votabile.
type MVPTeam struct {
	ID         int64          `json:"id"`
	Name       string         `json:"name"`
	ShortName  string         `json:"shortName,omitempty"`
	GroupName  string         `json:"groupName,omitempty"`
	Candidates []MVPCandidate `json:"candidates"`
}

// MVPBoard = payload pubblico della votazione: squadre con rosa+conteggi,
// totale voti e (se noto) il giocatore già votato da questo device.
type MVPBoard struct {
	Teams      []MVPTeam `json:"teams"`
	TotalVotes int       `json:"totalVotes"`
	MyVote     int64     `json:"myVote"` // playerId votato da questo device, 0 se nessuno
}

// GetMVPBoard ritorna la board completa per lo slug. Solo le squadre con almeno
// un giocatore in rosa compaiono (sono le uniche votabili).
func (s *Store) GetMVPBoard(ctx context.Context, slug, deviceID string) (*MVPBoard, error) {
	eventID, err := s.EventIDBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT tp.id, tp.team_id, tt.name, tt.short_name, tt.group_name,
		       tp.first_name, tp.last_name,
		       (SELECT COUNT(1) FROM tournament_mvp_votes v WHERE v.player_id = tp.id) AS votes
		FROM tournament_players tp
		JOIN tournament_teams tt ON tt.id = tp.team_id
		WHERE tp.event_id = ?
		ORDER BY tt.group_name, tt.name COLLATE NOCASE, tp.position, tp.id`, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	board := &MVPBoard{Teams: []MVPTeam{}}
	idx := map[int64]int{} // team_id -> indice in Teams
	for rows.Next() {
		var pid, teamID int64
		var teamName, shortName, groupName, first, last string
		var votes int
		if err := rows.Scan(&pid, &teamID, &teamName, &shortName, &groupName, &first, &last, &votes); err != nil {
			return nil, err
		}
		name := strings.TrimSpace(first + " " + last)
		if name == "" {
			continue // rosa senza nome: niente da votare
		}
		i, ok := idx[teamID]
		if !ok {
			i = len(board.Teams)
			idx[teamID] = i
			board.Teams = append(board.Teams, MVPTeam{
				ID: teamID, Name: teamName, ShortName: shortName, GroupName: groupName,
				Candidates: []MVPCandidate{},
			})
		}
		board.Teams[i].Candidates = append(board.Teams[i].Candidates,
			MVPCandidate{ID: pid, Name: name, Votes: votes})
		board.TotalVotes += votes
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if deviceID != "" {
		var voted int64
		err := s.db.QueryRowContext(ctx, `
			SELECT player_id FROM tournament_mvp_votes WHERE event_id = ? AND device_id = ?`,
			eventID, deviceID).Scan(&voted)
		if err == nil {
			board.MyVote = voted
		} else if !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
	}
	return board, nil
}

// CastMVPVote registra (o aggiorna) il voto di un device per un giocatore.
// Ritorna errTANotFound se il giocatore non appartiene all'evento.
func (s *Store) CastMVPVote(ctx context.Context, slug string, playerID int64, deviceID string) (int64, error) {
	eventID, err := s.EventIDBySlug(ctx, slug)
	if err != nil {
		return 0, err
	}
	// Il giocatore deve appartenere a questo torneo (anti-manomissione).
	var owned int
	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(1) FROM tournament_players WHERE id = ? AND event_id = ?`,
		playerID, eventID).Scan(&owned); err != nil {
		return 0, err
	}
	if owned == 0 {
		return 0, errTANotFound
	}
	// Upsert: un voto per device, modificabile.
	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO tournament_mvp_votes (event_id, player_id, device_id)
		VALUES (?, ?, ?)
		ON CONFLICT(event_id, device_id) DO UPDATE SET
			player_id = excluded.player_id, created_at = datetime('now')`,
		eventID, playerID, deviceID); err != nil {
		return 0, err
	}
	return eventID, nil
}

// --- Handlers ---------------------------------------------------------------

func (rt *_router) HandleTournamentMVP(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	board, err := rt.store.GetMVPBoard(r.Context(), slug, rt.deviceIDFromRequest(r))
	if err != nil {
		if errors.Is(err, errTANotFound) {
			http.Error(w, `{"error":"tournament_not_found"}`, http.StatusNotFound)
			return
		}
		rt.baseLogger.WithError(err).Error("mvp board")
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, board)
}

func (rt *_router) HandleTournamentMVPVote(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	deviceID := rt.deviceIDFromRequest(r)
	if deviceID == "" {
		http.Error(w, `{"error":"device_required"}`, http.StatusBadRequest)
		return
	}
	var body struct {
		PlayerID int64 `json:"playerId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.PlayerID == 0 {
		http.Error(w, `{"error":"bad_request"}`, http.StatusBadRequest)
		return
	}
	eventID, err := rt.store.CastMVPVote(r.Context(), slug, body.PlayerID, deviceID)
	if err != nil {
		if errors.Is(err, errTANotFound) {
			http.Error(w, `{"error":"not_found"}`, http.StatusNotFound)
			return
		}
		rt.baseLogger.WithError(err).Error("mvp vote")
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	// Push ai client connessi: i conteggi si aggiornano live.
	rt.tournamentHub.Broadcast(int(eventID))

	// Rispondi con la board aggiornata (evita un secondo GET dal client).
	board, err := rt.store.GetMVPBoard(r.Context(), slug, deviceID)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true, "myVote": body.PlayerID})
		return
	}
	writeJSON(w, http.StatusOK, board)
}
