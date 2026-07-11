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
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

func registerTournamentMVPRoutes(rt *_router) {
	rt.router.Get("/v1/tournaments/{slug}/mvp", rt.HandleTournamentMVP)
	rt.router.Post("/v1/tournaments/{slug}/mvp/vote", rt.HandleTournamentMVPVote)
	// Admin torneo: monitoraggio risultati votazione (scoping via wrapTA).
	rt.router.Get("/v1/ta/{slug}/mvp", rt.wrapTA(rt.taMVPResults))
}

// EnsureTournamentMVPTables crea/riconcilia la tabella dei voti MVP (idempotente).
// I DB con la vecchia feature "Tornei" avevano già una `tournament_mvp_votes`
// legacy (colonna `tournament_id`, niente `event_id`): stesso trattamento di
// teams/sponsors/players → reconcile allo schema canonico.
func (s *Store) EnsureTournamentMVPTables() error {
	// Schema canonico: un voto per (evento, device, GENERE) → il tifoso può
	// votare un uomo e una donna, entrambi modificabili (upsert per slot-genere).
	const createSQL = `CREATE TABLE tournament_mvp_votes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		event_id INTEGER NOT NULL,
		player_id INTEGER NOT NULL,
		gender TEXT NOT NULL DEFAULT 'male',
		device_id TEXT NOT NULL,
		created_at TEXT NOT NULL DEFAULT (datetime('now')),
		UNIQUE(event_id, device_id, gender)
	)`
	if _, err := s.db.Exec(`CREATE TABLE IF NOT EXISTS tournament_mvp_votes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		event_id INTEGER NOT NULL,
		player_id INTEGER NOT NULL,
		gender TEXT NOT NULL DEFAULT 'male',
		device_id TEXT NOT NULL,
		created_at TEXT NOT NULL DEFAULT (datetime('now')),
		UNIQUE(event_id, device_id, gender)
	)`); err != nil {
		return err
	}
	// Schema legacy "tournament_id" → canonico (event-based), copiando le colonne comuni.
	if err := s.reconcileTournamentTable("tournament_mvp_votes", createSQL,
		[]string{"id", "event_id", "player_id", "gender", "device_id", "created_at"}); err != nil {
		return err
	}
	// Colonna gender sui DB pre-genere.
	if _, err := s.db.Exec(`ALTER TABLE tournament_mvp_votes ADD COLUMN gender TEXT NOT NULL DEFAULT 'male'`); err != nil &&
		!strings.Contains(err.Error(), "duplicate column") {
		return err
	}
	// Backfill del genere dal giocatore votato.
	if _, err := s.db.Exec(`UPDATE tournament_mvp_votes SET gender = COALESCE((SELECT tp.gender FROM tournament_players tp WHERE tp.id = tournament_mvp_votes.player_id), 'male') WHERE gender IS NULL OR gender = ''`); err != nil {
		return err
	}
	// Migrazione unicità (event,device) → (event,device,gender): il vincolo è un
	// UNIQUE di tabella (auto-index non droppabile), quindi si ricostruisce la
	// tabella preservando i dati. Idempotente: si esegue solo se manca già.
	has, err := s.mvpVotesHasGenderUnique()
	if err != nil {
		return err
	}
	if !has {
		for _, q := range []string{
			`ALTER TABLE tournament_mvp_votes RENAME TO tournament_mvp_votes_old`,
			createSQL,
			`INSERT INTO tournament_mvp_votes (id, event_id, player_id, gender, device_id, created_at)
			 SELECT id, event_id, player_id, gender, device_id, created_at FROM tournament_mvp_votes_old`,
			`DROP TABLE tournament_mvp_votes_old`,
		} {
			if _, err := s.db.Exec(q); err != nil {
				return fmt.Errorf("mvp votes gender migration: %w", err)
			}
		}
	}
	return nil
}

// mvpVotesHasGenderUnique ritorna true se esiste un indice UNIQUE che copre
// esattamente (event_id, device_id, gender) — cioè lo schema già migrato.
func (s *Store) mvpVotesHasGenderUnique() (bool, error) {
	rows, err := s.db.Query(`PRAGMA index_list(tournament_mvp_votes)`)
	if err != nil {
		return false, err
	}
	var uniqueIdx []string
	for rows.Next() {
		var seq, unique, partial int
		var name, origin string
		if err := rows.Scan(&seq, &name, &unique, &origin, &partial); err != nil {
			rows.Close()
			return false, err
		}
		if unique == 1 {
			uniqueIdx = append(uniqueIdx, name)
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return false, err
	}
	rows.Close()

	for _, name := range uniqueIdx {
		cols, err := s.indexColumns(name)
		if err != nil {
			return false, err
		}
		if len(cols) == 3 && cols["event_id"] && cols["device_id"] && cols["gender"] {
			return true, nil
		}
	}
	return false, nil
}

// indexColumns ritorna l'insieme dei nomi colonna coperti da un indice.
func (s *Store) indexColumns(name string) (map[string]bool, error) {
	rows, err := s.db.Query(`PRAGMA index_info(` + name + `)`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	cols := map[string]bool{}
	for rows.Next() {
		var seqno, cid int
		var col sql.NullString
		if err := rows.Scan(&seqno, &cid, &col); err != nil {
			return nil, err
		}
		if col.Valid {
			cols[col.String] = true
		}
	}
	return cols, rows.Err()
}

// --- Store ------------------------------------------------------------------

// MVPCandidate = un giocatore votabile con il suo conteggio voti corrente.
// Gender ("male"/"female") consente all'admin di distinguere MVP uomo e donna.
type MVPCandidate struct {
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	Gender string `json:"gender"`
	Votes  int    `json:"votes"`
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
	// Voti di questo device: uno per slot-genere (0 se non ancora espresso).
	MyVoteMale   int64 `json:"myVoteMale"`
	MyVoteFemale int64 `json:"myVoteFemale"`
}

// GetMVPBoard ritorna la board completa per lo slug (vista tifoso, con MyVote).
// Solo le squadre con almeno un giocatore in rosa compaiono (le uniche votabili).
func (s *Store) GetMVPBoard(ctx context.Context, slug, deviceID string) (*MVPBoard, error) {
	eventID, err := s.EventIDBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	return s.mvpBoardByEvent(ctx, eventID, deviceID)
}

// GetMVPResults ritorna la board per l'admin (per eventID, senza voto-device).
func (s *Store) GetMVPResults(ctx context.Context, eventID int64) (*MVPBoard, error) {
	return s.mvpBoardByEvent(ctx, eventID, "")
}

// mvpBoardByEvent costruisce la board dai voti dell'evento. Se deviceID != ""
// popola anche MyVote (giocatore votato da quel device).
func (s *Store) mvpBoardByEvent(ctx context.Context, eventID int64, deviceID string) (*MVPBoard, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT tp.id, tp.team_id, tt.name, tt.short_name, tt.group_name,
		       tp.first_name, tp.last_name, tp.gender,
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
		var teamName, shortName, groupName, first, last, gender string
		var votes int
		if err := rows.Scan(&pid, &teamID, &teamName, &shortName, &groupName, &first, &last, &gender, &votes); err != nil {
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
			MVPCandidate{ID: pid, Name: name, Gender: normalizeTAGender(gender), Votes: votes})
		board.TotalVotes += votes
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if deviceID != "" {
		vrows, err := s.db.QueryContext(ctx, `
			SELECT gender, player_id FROM tournament_mvp_votes WHERE event_id = ? AND device_id = ?`,
			eventID, deviceID)
		if err != nil {
			return nil, err
		}
		defer vrows.Close()
		for vrows.Next() {
			var g string
			var pid int64
			if err := vrows.Scan(&g, &pid); err != nil {
				return nil, err
			}
			if normalizeTAGender(g) == "female" {
				board.MyVoteFemale = pid
			} else {
				board.MyVoteMale = pid
			}
		}
		if err := vrows.Err(); err != nil {
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
	// Il giocatore deve appartenere a questo torneo (anti-manomissione); il suo
	// genere determina lo slot di voto (uomo/donna).
	var gender string
	err = s.db.QueryRowContext(ctx, `
		SELECT gender FROM tournament_players WHERE id = ? AND event_id = ?`,
		playerID, eventID).Scan(&gender)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, errTANotFound
	}
	if err != nil {
		return 0, err
	}
	gender = normalizeTAGender(gender)
	// Upsert: un voto per (device, genere), modificabile.
	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO tournament_mvp_votes (event_id, player_id, gender, device_id)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(event_id, device_id, gender) DO UPDATE SET
			player_id = excluded.player_id, created_at = datetime('now')`,
		eventID, playerID, gender, deviceID); err != nil {
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
		writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true})
		return
	}
	writeJSON(w, http.StatusOK, board)
}

// taMVPResults: monitoraggio admin dell'andamento della votazione MVP.
func (rt *_router) taMVPResults(w http.ResponseWriter, r *http.Request, eventID int64) {
	board, err := rt.store.GetMVPResults(r.Context(), eventID)
	if err != nil {
		rt.baseLogger.WithError(err).WithField("eventID", eventID).Error("mvp results")
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, board)
}
