package api

// ============================================================================
// TOURNAMENT BRACKET — generazione automatica della fase finale dai gironi.
// A gironi conclusi, prende le prime `qualifiers` di ogni girone, le semina con
// lo schema standard (teste di serie separate, stesso girone lontano) e crea
// TUTTO il tabellone: il primo turno con squadre reali, i turni successivi con
// segnaposto ("Vincente Q1") e link di avanzamento. Alla chiusura di ogni
// partita il vincitore riempie da solo lo slot successivo (vedi ApplyScoreAction).
// ============================================================================

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
)

// seedPositions ritorna l'ordine standard dei seed (1-based) nei bracket slot
// per n potenza di 2. Es: n=8 -> [1,8,4,5,2,7,3,6]; le coppie consecutive sono
// gli accoppiamenti del primo turno (1-8, 4-5, 2-7, 3-6).
func seedPositions(n int) []int {
	order := []int{1}
	for len(order) < n {
		m := len(order) * 2
		next := make([]int, 0, m)
		for _, s := range order {
			next = append(next, s, m+1-s)
		}
		order = next
	}
	return order
}

func isPowerOfTwo(n int) bool { return n >= 2 && n&(n-1) == 0 }

func roundStage(matchesInRound int) string {
	switch matchesInRound {
	case 8:
		return "OTTAVI"
	case 4:
		return "QUARTI"
	case 2:
		return "SEMIFINALE"
	case 1:
		return "FINALE"
	default:
		return fmt.Sprintf("TURNO DA %d", matchesInRound*2)
	}
}

func roundCode(matchesInRound int) string {
	switch matchesInRound {
	case 8:
		return "O"
	case 4:
		return "Q"
	case 2:
		return "S"
	default:
		return "F"
	}
}

func newBracketMatchID(eventID int64) (string, error) {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return fmt.Sprintf("m%d-%s", eventID, base64.RawURLEncoding.EncodeToString(buf)), nil
}

type bracketMatch struct {
	id                   string
	stage                string
	teamA, teamB         int64
	labelA, labelB       string
	winToID, loseToID    string
	winToSlot, loseToSlot int
}

// GenerateBracket (ri)genera il tabellone dai gironi. Ritorna il numero di
// partite create. Errori "attesi" (stringa-codice) da mostrare all'utente:
// groups_not_finished, no_group_matches, no_groups, not_enough_teams,
// not_power_of_two:<Q>.
func (s *Store) GenerateBracket(ctx context.Context, eventID int64, qualifiers int, thirdPlace bool) (int, error) {
	if qualifiers < 1 {
		qualifiers = 1
	}
	_, slug, err := s.GetTASettings(ctx, eventID)
	if err != nil {
		return 0, err
	}

	// 1) tutte le partite dei gironi (stage='') devono essere concluse
	var pending int
	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(1) FROM matches
		WHERE event_id = ? AND COALESCE(stage,'') = '' AND status != 'finished'`, eventID).Scan(&pending); err != nil {
		return 0, err
	}
	if pending > 0 {
		return 0, fmt.Errorf("groups_not_finished")
	}
	var played int
	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(1) FROM matches
		WHERE event_id = ? AND COALESCE(stage,'') = '' AND status = 'finished'`, eventID).Scan(&played); err != nil {
		return 0, err
	}
	if played == 0 {
		return 0, fmt.Errorf("no_group_matches")
	}

	// 2) classifiche per girone (già ordinate)
	groups, _, err := s.ComputeStandings(ctx, slug)
	if err != nil {
		return 0, err
	}
	if len(groups) == 0 {
		return 0, fmt.Errorf("no_groups")
	}
	for _, g := range groups {
		if len(g.Rows) < qualifiers {
			return 0, fmt.Errorf("not_enough_teams")
		}
	}

	// 3) seed list per (rank, girone): [1°gir.A, 1°gir.B, ..., 2°gir.A, ...]
	seeds := make([]int64, 0, len(groups)*qualifiers)
	for rank := 0; rank < qualifiers; rank++ {
		for _, g := range groups {
			seeds = append(seeds, g.Rows[rank].TeamID)
		}
	}
	Q := len(seeds)
	if !isPowerOfTwo(Q) {
		return 0, fmt.Errorf("not_power_of_two:%d", Q)
	}

	// 4) turni: rounds[0] primo turno (Q/2 match) ... ultimo = finale (1 match)
	numRounds := 0
	for x := Q; x >= 2; x /= 2 {
		numRounds++
	}
	rounds := make([][]*bracketMatch, numRounds)
	for r := 0; r < numRounds; r++ {
		cnt := Q / (1 << (r + 1))
		rounds[r] = make([]*bracketMatch, cnt)
		for i := 0; i < cnt; i++ {
			id, err := newBracketMatchID(eventID)
			if err != nil {
				return 0, err
			}
			rounds[r][i] = &bracketMatch{id: id, stage: roundStage(cnt)}
		}
	}
	positions := seedPositions(Q)
	for i := range rounds[0] {
		rounds[0][i].teamA = seeds[positions[2*i]-1]
		rounds[0][i].teamB = seeds[positions[2*i+1]-1]
	}
	for r := 1; r < numRounds; r++ {
		prevCode := roundCode(len(rounds[r-1]))
		for i := range rounds[r] {
			m := rounds[r][i]
			feedA, feedB := rounds[r-1][2*i], rounds[r-1][2*i+1]
			m.labelA = fmt.Sprintf("Vincente %s%d", prevCode, 2*i+1)
			m.labelB = fmt.Sprintf("Vincente %s%d", prevCode, 2*i+2)
			feedA.winToID, feedA.winToSlot = m.id, 0
			feedB.winToID, feedB.winToSlot = m.id, 1
		}
	}

	all := make([]*bracketMatch, 0, Q)
	for r := 0; r < numRounds; r++ {
		all = append(all, rounds[r]...)
	}

	// 5) finalina 3°/4° posto: solo se richiesta e c'è un turno di semifinale
	if thirdPlace {
		for r := 0; r < numRounds; r++ {
			if len(rounds[r]) == 2 {
				id, err := newBracketMatchID(eventID)
				if err != nil {
					return 0, err
				}
				third := &bracketMatch{
					id: id, stage: "FINALE 3° POSTO",
					labelA: "Perdente S1", labelB: "Perdente S2",
				}
				rounds[r][0].loseToID, rounds[r][0].loseToSlot = third.id, 0
				rounds[r][1].loseToID, rounds[r][1].loseToSlot = third.id, 1
				all = append(all, third)
				break
			}
		}
	}

	// 6) scrittura in tx: rimpiazza il tabellone esistente
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx,
		`DELETE FROM matches WHERE event_id = ? AND COALESCE(stage,'') != ''`, eventID); err != nil {
		return 0, err
	}
	for _, m := range all {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO matches (id, event_id, court, scheduled_time, scheduled_at, stage, status,
			                     team_a_id, team_b_id, team_a_label, team_b_label,
			                     win_to_id, win_to_slot, lose_to_id, lose_to_slot)
			VALUES (?, ?, '', '', '', ?, 'scheduled', ?, ?, ?, ?, ?, ?, ?, ?)`,
			m.id, eventID, m.stage, m.teamA, m.teamB, m.labelA, m.labelB,
			m.winToID, m.winToSlot, m.loseToID, m.loseToSlot); err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return len(all), nil
}

// taGenerateBracket: handler admin. Legge qualificate/finalina dalle impostazioni
// (persistite) e (ri)genera il tabellone.
func (rt *_router) taGenerateBracket(w http.ResponseWriter, r *http.Request, eventID int64) {
	st, _, err := rt.store.GetTASettings(r.Context(), eventID)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	count, err := rt.store.GenerateBracket(r.Context(), eventID, st.BracketQualifiers, st.BracketThirdPlace)
	if err != nil {
		switch msg := err.Error(); {
		case msg == "groups_not_finished" || msg == "no_group_matches" || msg == "no_groups" ||
			msg == "not_enough_teams" || len(msg) > 17 && msg[:17] == "not_power_of_two:":
			http.Error(w, fmt.Sprintf(`{"error":%q}`, msg), http.StatusBadRequest)
			return
		default:
			rt.baseLogger.WithError(err).WithField("eventID", eventID).Error("cannot generate bracket")
			http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
			return
		}
	}
	rt.invalidateTournamentCaches(r, eventID)
	writeJSON(w, http.StatusOK, map[string]interface{}{"created": count})
}
