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
	"sort"
	"time"
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
	id                    string
	stage                 string
	court, timeLabel      string
	scheduledAt           string
	teamA, teamB          int64
	labelA, labelB        string
	winToID, loseToID     string
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
	settings, slug, err := s.GetTASettings(ctx, eventID)
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

	// Ordine temporale dei turni: tutte le gare del primo turno, poi semifinali,
	// finalina e finale. I collegamenti del tabellone restano invariati.
	stageRank := func(stage string) int {
		switch stage {
		case "OTTAVI":
			return 0
		case "QUARTI":
			return 1
		case "SEMIFINALE":
			return 2
		case "FINALE 3° POSTO":
			return 3
		case "FINALE":
			return 4
		default:
			return 5
		}
	}
	sort.SliceStable(all, func(i, j int) bool {
		return stageRank(all[i].stage) < stageRank(all[j].stage)
	})

	// La prima gara della fase finale parte 30 minuti dopo l'ultima gara dei
	// gironi; le successive avanzano di mezz'ora. Con più campi li assegniamo
	// ciclicamente nell'ordine configurato.
	var lastTimeLabel, lastScheduledAt string
	rows, err := s.db.QueryContext(ctx, `
		SELECT scheduled_time, COALESCE(scheduled_at,'')
		FROM matches
		WHERE event_id = ? AND COALESCE(stage,'') = ''
		ORDER BY
		  CASE WHEN scheduled_time >= COALESCE(
		         (SELECT a.scheduled_time FROM matches a
		            WHERE a.event_id = ? AND a.is_anchor = 1 LIMIT 1), '')
		       THEN 0 ELSE 1 END,
		  scheduled_time`, eventID, eventID)
	if err != nil {
		return 0, err
	}
	for rows.Next() {
		if err := rows.Scan(&lastTimeLabel, &lastScheduledAt); err != nil {
			rows.Close()
			return 0, err
		}
	}
	if err := rows.Close(); err != nil {
		return 0, err
	}
	baseClock, clockErr := time.Parse("15:04", lastTimeLabel)
	var baseScheduled time.Time
	var scheduledLayout string
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04"} {
		if parsed, parseErr := time.Parse(layout, lastScheduledAt); parseErr == nil {
			baseScheduled, scheduledLayout = parsed, layout
			break
		}
	}
	courts := settings.Courts
	if len(courts) == 0 {
		courts = []string{"CAMPO 1"}
	}
	for i, match := range all {
		offset := time.Duration(i+1) * 30 * time.Minute
		match.court = courts[i%len(courts)]
		if clockErr == nil {
			match.timeLabel = baseClock.Add(offset).Format("15:04")
		}
		if scheduledLayout != "" {
			match.scheduledAt = baseScheduled.Add(offset).Format(scheduledLayout)
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
			VALUES (?, ?, ?, ?, ?, ?, 'scheduled', ?, ?, ?, ?, ?, ?, ?, ?)`,
			m.id, eventID, m.court, m.timeLabel, m.scheduledAt, m.stage, m.teamA, m.teamB, m.labelA, m.labelB,
			m.winToID, m.winToSlot, m.loseToID, m.loseToSlot); err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return len(all), nil
}

type BracketAutoPrompt struct {
	Pending    bool  `json:"pending"`
	DeadlineMs int64 `json:"deadlineMs,omitempty"`
}

func (s *Store) ListBracketAutoPrompts(ctx context.Context) (map[int64]BracketAutoPrompt, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, bracket_auto_generate_at
		FROM events
		WHERE type = 'tournament' AND bracket_auto_generate_state = 0
		  AND bracket_auto_generate_at != ''`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int64]BracketAutoPrompt{}
	for rows.Next() {
		var eventID int64
		var raw string
		if err := rows.Scan(&eventID, &raw); err != nil {
			return nil, err
		}
		if at, err := time.Parse(time.RFC3339Nano, raw); err == nil {
			out[eventID] = BracketAutoPrompt{Pending: true, DeadlineMs: at.UnixMilli()}
		}
	}
	return out, rows.Err()
}

func (s *Store) GetBracketAutoPrompt(ctx context.Context, eventID int64) BracketAutoPrompt {
	var deadline string
	var state int
	if err := s.db.QueryRowContext(ctx, `
		SELECT COALESCE(bracket_auto_generate_at,''), COALESCE(bracket_auto_generate_state,0)
		FROM events WHERE id = ?`, eventID).Scan(&deadline, &state); err != nil || deadline == "" || state != 0 {
		return BracketAutoPrompt{}
	}
	at, err := time.Parse(time.RFC3339Nano, deadline)
	if err != nil {
		return BracketAutoPrompt{}
	}
	return BracketAutoPrompt{Pending: true, DeadlineMs: at.UnixMilli()}
}

// ScheduleBracketAutoGeneration apre una finestra di 30 secondi soltanto
// quando tutti i gironi sono conclusi e non esiste ancora una fase finale.
func (s *Store) ScheduleBracketAutoGeneration(ctx context.Context, eventID int64) (BracketAutoPrompt, bool, error) {
	var groupTotal, groupPending, finals int
	if err := s.db.QueryRowContext(ctx, `
		SELECT
		  COALESCE(SUM(CASE WHEN COALESCE(stage,'') = '' THEN 1 ELSE 0 END),0),
		  COALESCE(SUM(CASE WHEN COALESCE(stage,'') = '' AND status != 'finished' THEN 1 ELSE 0 END),0),
		  COALESCE(SUM(CASE WHEN COALESCE(stage,'') != '' THEN 1 ELSE 0 END),0)
		FROM matches WHERE event_id = ?`, eventID).Scan(&groupTotal, &groupPending, &finals); err != nil {
		return BracketAutoPrompt{}, false, err
	}
	if groupTotal == 0 || groupPending != 0 || finals != 0 {
		return BracketAutoPrompt{}, false, nil
	}
	existing := s.GetBracketAutoPrompt(ctx, eventID)
	if existing.Pending {
		return existing, false, nil
	}
	deadline := time.Now().UTC().Add(30 * time.Second)
	res, err := s.db.ExecContext(ctx, `
		UPDATE events SET bracket_auto_generate_at = ?, bracket_auto_generate_state = 0
		WHERE id = ? AND COALESCE(bracket_auto_generate_at,'') = ''
		  AND COALESCE(bracket_auto_generate_state,0) = 0`,
		deadline.Format(time.RFC3339Nano), eventID)
	if err != nil {
		return BracketAutoPrompt{}, false, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return s.GetBracketAutoPrompt(ctx, eventID), false, nil
	}
	return BracketAutoPrompt{Pending: true, DeadlineMs: deadline.UnixMilli()}, true, nil
}

func (s *Store) DeclineBracketAutoGeneration(ctx context.Context, eventID int64) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE events SET bracket_auto_generate_at = '', bracket_auto_generate_state = 1
		WHERE id = ? AND bracket_auto_generate_state = 0`, eventID)
	return err
}

func (s *Store) ResetBracketAutoGeneration(ctx context.Context, eventID int64) {
	_, _ = s.db.ExecContext(ctx, `
		UPDATE events SET bracket_auto_generate_at = '', bracket_auto_generate_state = 0 WHERE id = ?`, eventID)
}

// ClaimBracketAutoGeneration impedisce che timer e operatori generino due
// volte lo stesso tabellone. Lo stato 2 indica "in generazione".
func (s *Store) ClaimBracketAutoGeneration(ctx context.Context, eventID int64, requireDue bool) (bool, error) {
	prompt := s.GetBracketAutoPrompt(ctx, eventID)
	if !prompt.Pending || (requireDue && time.Now().UnixMilli() < prompt.DeadlineMs) {
		return false, nil
	}
	res, err := s.db.ExecContext(ctx, `
		UPDATE events SET bracket_auto_generate_state = 2
		WHERE id = ? AND bracket_auto_generate_state = 0 AND bracket_auto_generate_at != ''`, eventID)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n == 1, nil
}

func (s *Store) FinishClaimedBracketGeneration(ctx context.Context, eventID int64, success bool) {
	if success {
		s.ResetBracketAutoGeneration(ctx, eventID)
		return
	}
	_, _ = s.db.ExecContext(ctx, `
		UPDATE events SET bracket_auto_generate_state = 0 WHERE id = ? AND bracket_auto_generate_state = 2`, eventID)
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
	rt.store.ResetBracketAutoGeneration(r.Context(), eventID)
	rt.invalidateTournamentCaches(r, eventID)
	writeJSON(w, http.StatusOK, map[string]interface{}{"created": count})
}
