package database

import (
	"database/sql"
	"errors"
	"math"
	"sync"
	"testing"
	"time"
)

func crossoverWinningTrace() []CrossoverSample {
	return []CrossoverSample{{0, 0}, {100, .8}, {400, .8}, {500, -.8}}
}

func TestCrossoverSessionStatsAndImmediateNextRound(t *testing.T) {
	db, now := basketballTestDB(t)
	session, err := db.CreateCrossoverSession(42, 7, 0, "device", now)
	if err != nil || session.NextDuel != 0 || session.RemainingMS != 25000 || session.Wins != 0 || session.QualityTotal != 0 {
		t.Fatalf("start %+v: %v", session, err)
	}
	resumed, err := db.CreateCrossoverSession(42, 7, 0, "device", now.Add(time.Second))
	if err != nil || resumed.SessionID != session.SessionID || resumed.RemainingMS != 24000 {
		t.Fatalf("resume %+v: %v", resumed, err)
	}
	for duel := 0; duel < 3; duel++ {
		result, err := db.ResolveCrossoverDuel(42, 7, 0, "device", session.SessionID, duel, duel*1600, crossoverWinningTrace(), now.Add(time.Duration(duel+1)*1600*time.Millisecond))
		if err != nil || !result.Won || !result.Clean || result.Quality != 94 || result.CoinsEarned != 1 || result.Wallet != duel+1 || result.CrossoverEarned != duel+1 || result.Wins != duel+1 || result.Streak != duel+1 || result.BestStreak != duel+1 || result.QualityTotal != 94*(duel+1) || result.NextDuel != duel+1 {
			t.Fatalf("duel %d %+v: %v", duel, result, err)
		}
		if duel < 2 {
			resumed, err = db.CreateCrossoverSession(42, 7, 0, "device", now.Add(time.Duration(duel+1)*1600*time.Millisecond))
			if err != nil || resumed.SessionID != session.SessionID || resumed.NextDuel != duel+1 || resumed.Wins != duel+1 || resumed.QualityTotal != 94*(duel+1) {
				t.Fatalf("restored stats %+v: %v", resumed, err)
			}
		}
	}
	next, err := db.CreateCrossoverSession(42, 7, 0, "device", now.Add(5*time.Second))
	if err != nil || next.SessionID == session.SessionID || next.NextDuel != 0 || next.Wins != 0 || next.QualityTotal != 0 || next.RemainingMS != 25000 {
		t.Fatalf("immediate replay %+v: %v", next, err)
	}
	if _, err := db.ResolveCrossoverDuel(42, 7, 0, "device", session.SessionID, 3, 5000, crossoverWinningTrace(), now.Add(7*time.Second)); !errors.Is(err, ErrCrossoverGesture) {
		t.Fatalf("fourth duel %v", err)
	}
	rewards, err := db.ListFanJourneyRewards(42, 7, 0, "device")
	if err != nil || rewards["crossover"] != 3 {
		t.Fatalf("journey %+v: %v", rewards, err)
	}
}

func TestCrossoverConcurrentRetriesRemainIdempotent(t *testing.T) {
	db, now := basketballTestDB(t)
	session, err := db.CreateCrossoverSession(42, 7, 0, "device", now)
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	results := make(chan CrossoverDuelResult, 2)
	errorsCh := make(chan error, 2)
	for n := 0; n < 2; n++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := db.ResolveCrossoverDuel(42, 7, 0, "device", session.SessionID, 0, 0, crossoverWinningTrace(), now.Add(1600*time.Millisecond))
			results <- result
			errorsCh <- err
		}()
	}
	wg.Wait()
	close(results)
	close(errorsCh)
	for err := range errorsCh {
		if err != nil {
			t.Fatal(err)
		}
	}
	credited := 0
	for result := range results {
		credited += result.CoinsEarned
		if !result.Won || result.Wallet != 1 || result.CrossoverEarned != 1 || result.Wins != 1 || result.NextDuel != 1 {
			t.Fatalf("duplicate %+v", result)
		}
	}
	if credited != 1 {
		t.Fatalf("duplicate credited %d", credited)
	}
	if _, err := db.c.Exec(`UPDATE events SET is_active=0,is_concluded=1 WHERE id=42`); err != nil {
		t.Fatal(err)
	}
	// An existing receipt is authoritative even if a later retry has stale
	// timing/input and the match has been closed in the meantime.
	retry, err := db.ResolveCrossoverDuel(42, 7, 0, "device", session.SessionID, 0, -1, nil, now.Add(time.Hour))
	if err != nil || !retry.Won || retry.CoinsEarned != 0 || retry.Wallet != 1 || retry.Wins != 1 || retry.RemainingMS != 0 {
		t.Fatalf("late retry %+v: %v", retry, err)
	}
}

func TestCrossoverFailedDuelAndStreakRestore(t *testing.T) {
	db, now := basketballTestDB(t)
	session, err := db.CreateCrossoverSession(42, 7, 0, "device", now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.ResolveCrossoverDuel(42, 7, 0, "device", session.SessionID, 0, 0, crossoverWinningTrace(), now.Add(1600*time.Millisecond)); err != nil {
		t.Fatal(err)
	}
	miss := []CrossoverSample{{0, 0}, {100, 0}}
	result, err := db.ResolveCrossoverDuel(42, 7, 0, "device", session.SessionID, 1, 1600, miss, now.Add(3200*time.Millisecond))
	if err != nil || result.Won || result.Reason != "no_feint" || result.Quality != 0 || result.CoinsEarned != 0 || result.Wallet != 1 || result.Wins != 1 || result.Streak != 0 || result.BestStreak != 1 || result.QualityTotal != 94 {
		t.Fatalf("failed duel %+v: %v", result, err)
	}
	retry, err := db.ResolveCrossoverDuel(42, 7, 0, "device", session.SessionID, 1, 1600, crossoverWinningTrace(), now.Add(4*time.Second))
	if err != nil || retry.Won || retry.Reason != "no_feint" || retry.CoinsEarned != 0 {
		t.Fatalf("immutable miss %+v: %v", retry, err)
	}
	resumed, err := db.CreateCrossoverSession(42, 7, 0, "device", now.Add(4*time.Second))
	if err != nil || resumed.NextDuel != 2 || resumed.Wins != 1 || resumed.Streak != 0 || resumed.BestStreak != 1 || resumed.QualityTotal != 94 {
		t.Fatalf("resumed %+v: %v", resumed, err)
	}
}

func TestCrossoverScopeClockAndGrace(t *testing.T) {
	db, now := basketballTestDB(t)
	session, err := db.CreateCrossoverSession(42, 7, 0, "device", now)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		event, org int
		device     string
	}{{42, 8, "device"}, {43, 7, "device"}, {42, 7, "other"}} {
		if _, err := db.ResolveCrossoverDuel(tc.event, tc.org, 0, tc.device, session.SessionID, 0, 0, crossoverWinningTrace(), now.Add(2*time.Second)); !errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("scope %+v: %v", tc, err)
		}
	}
	if _, err := db.ResolveCrossoverDuel(42, 7, 0, "device", session.SessionID, 1, 0, crossoverWinningTrace(), now.Add(2*time.Second)); !errors.Is(err, ErrCrossoverOrder) {
		t.Fatalf("out of order %v", err)
	}
	if _, err := db.ResolveCrossoverDuel(42, 7, 0, "device", session.SessionID, 0, 0, crossoverWinningTrace(), now.Add(time.Second)); !errors.Is(err, ErrCrossoverTooEarly) {
		t.Fatalf("early %v", err)
	}
	for _, start := range []int{-1, 25000, 5000} {
		if _, err := db.ResolveCrossoverDuel(42, 7, 0, "device", session.SessionID, 0, start, crossoverWinningTrace(), now.Add(1600*time.Millisecond)); !errors.Is(err, ErrCrossoverGesture) {
			t.Fatalf("invalid time %d: %v", start, err)
		}
	}
	last, err := db.ResolveCrossoverDuel(42, 7, 0, "device", session.SessionID, 0, 24500, crossoverWinningTrace(), now.Add(26*time.Second))
	if err != nil || !last.Won || last.RemainingMS != 0 {
		t.Fatalf("last duel grace %+v: %v", last, err)
	}
	if _, err := db.ResolveCrossoverDuel(42, 7, 0, "device", session.SessionID, 1, 24500, crossoverWinningTrace(), now.Add(28*time.Second)); !errors.Is(err, ErrCrossoverExpired) {
		t.Fatalf("one new grace result only %v", err)
	}
	newRound, err := db.CreateCrossoverSession(42, 7, 0, "device", now.Add(30*time.Second))
	if err != nil || newRound.SessionID == session.SessionID {
		t.Fatalf("expired session renewal %+v: %v", newRound, err)
	}
	if _, err := db.ResolveCrossoverDuel(42, 7, 0, "device", newRound.SessionID, 0, 0, crossoverWinningTrace(), now.Add(71*time.Second)); !errors.Is(err, ErrCrossoverExpired) {
		t.Fatalf("outside grace %v", err)
	}
}

func TestCrossoverRegisteredProgressAndGuestTransfer(t *testing.T) {
	db, now := basketballTestDB(t)
	session, err := db.CreateCrossoverSession(42, 7, 0, "device", now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.ResolveCrossoverDuel(42, 7, 0, "device", session.SessionID, 0, 0, crossoverWinningTrace(), now.Add(1600*time.Millisecond)); err != nil {
		t.Fatal(err)
	}
	profile, err := db.RegisterFan(FanRegisterInput{OrganizationID: 7, EventID: 42, DeviceID: "device", SessionToken: "session", Nickname: "Fan", Phone: "123", AcceptedTerms: true, GuestCoins: 1})
	if err != nil || profile.Wallet != 1 {
		t.Fatalf("register %+v: %v", profile, err)
	}
	fanID := profile.Profile.ID
	rewards, err := db.ListFanJourneyRewards(42, 7, fanID, "new-device")
	if err != nil || rewards["crossover"] != 1 {
		t.Fatalf("portable progress %+v: %v", rewards, err)
	}
	if _, err := db.ResolveCrossoverDuel(42, 7, 0, "device", session.SessionID, 0, 0, crossoverWinningTrace(), now.Add(2*time.Second)); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("guest cannot replay owned duel %v", err)
	}
	resumed, err := db.CreateCrossoverSession(42, 7, fanID, "device", now.Add(2*time.Second))
	if err != nil || resumed.SessionID != session.SessionID || resumed.Wins != 1 {
		t.Fatalf("registered resume %+v: %v", resumed, err)
	}
	if err := db.AddFanWalletCoins(fanID, 20); err != nil {
		t.Fatal(err)
	}
	other, err := db.CreateCrossoverSession(42, 7, fanID, "new-device", now.Add(3*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	result, err := db.ResolveCrossoverDuel(42, 7, fanID, "new-device", other.SessionID, 0, 0, crossoverWinningTrace(), now.Add(4600*time.Millisecond))
	if err != nil || result.Wallet != 22 || result.CrossoverEarned != 2 || result.CoinsEarned != 1 {
		t.Fatalf("wallet increment %+v: %v", result, err)
	}
	if _, err := db.CreateCrossoverSession(44, 8, fanID, "device", now); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("cross-org fan %v", err)
	}
	otherRewards, err := db.ListFanJourneyRewards(43, 7, fanID, "new-device")
	if err != nil || otherRewards["crossover"] != 0 {
		t.Fatalf("other event %+v: %v", otherRewards, err)
	}
}

func TestCrossoverTraceValidation(t *testing.T) {
	tooMany := make([]CrossoverSample, 129)
	for index := range tooMany {
		tooMany[index] = CrossoverSample{TMS: index, X: 0}
	}
	invalid := [][]CrossoverSample{nil, {{0, 0}}, {{1, 0}, {100, .8}}, {{0, .1}, {100, .8}}, {{0, 0}, {100, .8}, {100, -.8}}, {{0, 0}, {-1, .8}}, {{0, 0}, {6001, .8}}, {{0, 0}, {100, 1.01}}, {{0, 0}, {100, math.NaN()}}, {{0, 0}, {100, math.Inf(1)}}, tooMany}
	for index, samples := range invalid {
		if _, err := evaluateCrossoverDuel(0, samples); !errors.Is(err, ErrCrossoverGesture) {
			t.Fatalf("invalid trace %d: %v", index, err)
		}
	}
	for _, duel := range []int{-1, 3} {
		if _, err := evaluateCrossoverDuel(duel, crossoverWinningTrace()); !errors.Is(err, ErrCrossoverGesture) {
			t.Fatalf("invalid duel %d: %v", duel, err)
		}
	}
}

func TestCrossoverGoldenPhysicsMatchesJavaScript(t *testing.T) {
	fast := []CrossoverSample{{0, 0}, {120, .6}, {320, .6}, {460, -.8}}
	for duel, quality := range []int{89, 88, 88} {
		result, err := evaluateCrossoverDuel(duel, fast)
		if err != nil || !result.won || result.clean || result.quality != quality || result.reason != "beat" {
			t.Fatalf("golden fast duel%d %+v: %v", duel, result, err)
		}
	}
	for _, tc := range []struct {
		samples []CrossoverSample
		reason  string
	}{
		{[]CrossoverSample{{0, 0}, {100, .7}, {180, -.8}}, "no_feint"},
		{[]CrossoverSample{{0, 0}, {100, .6}, {500, .6}}, "no_crossover"},
		{[]CrossoverSample{{0, 0}, {100, .6}, {400, .6}, {600, 0}, {2000, -.8}}, "too_slow"},
		{[]CrossoverSample{{0, 0}, {100, .6}, {400, .6}, {1100, -.8}}, "blocked"},
	} {
		result, err := evaluateCrossoverDuel(0, tc.samples)
		if err != nil || result.won || result.clean || result.quality != 0 || result.reason != tc.reason {
			t.Fatalf("golden %s %+v: %v", tc.reason, result, err)
		}
	}
	corrected := []CrossoverSample{{0, 0}, {100, .6}, {400, .6}, {1100, -.8}, {1450, -.8}, {1590, .8}}
	if result, err := evaluateCrossoverDuel(0, corrected); err != nil || !result.won || result.reason != "beat" {
		t.Fatalf("later correction %+v: %v", result, err)
	}
	for duel := 0; duel < 3; duel++ {
		mirror := crossoverWinningTrace()
		for index := range mirror {
			mirror[index].X = -mirror[index].X
		}
		if result, err := evaluateCrossoverDuel(duel, mirror); err != nil || !result.won || result.quality != 94 || !result.clean {
			t.Fatalf("mirrored trace %+v: %v", result, err)
		}
	}
}
