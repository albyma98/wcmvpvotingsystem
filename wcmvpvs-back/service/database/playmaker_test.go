package database

import (
	"database/sql"
	"errors"
	"math"
	"sync"
	"testing"
	"time"
)

func TestPlaymakerSessionClockAndResume(t *testing.T) {
	db, now := basketballTestDB(t)
	session, err := db.CreatePlaymakerSession(42, 7, 0, "device", now)
	if err != nil || session.Attempt != 0 || session.RemainingMS != 30000 {
		t.Fatalf("start %+v: %v", session, err)
	}
	started, err := time.Parse(time.RFC3339Nano, session.StartedAt)
	if err != nil || !started.Equal(now) {
		t.Fatalf("start time %s: %v", session.StartedAt, err)
	}
	ends, err := time.Parse(time.RFC3339Nano, session.EndsAt)
	if err != nil || ends.Sub(started) != 30*time.Second {
		t.Fatalf("end time %s: %v", session.EndsAt, err)
	}
	resumed, err := db.CreatePlaymakerSession(42, 7, 0, "device", now.Add(15*time.Second))
	if err != nil || resumed.SessionID != session.SessionID || resumed.RemainingMS != 15000 {
		t.Fatalf("resume %+v: %v", resumed, err)
	}
	other, err := db.CreatePlaymakerSession(42, 7, 0, "other", now)
	if err != nil || other.SessionID == session.SessionID {
		t.Fatalf("independent device %+v: %v", other, err)
	}
	newRound, err := db.CreatePlaymakerSession(42, 7, 0, "device", now.Add(30*time.Second))
	if err != nil || newRound.SessionID == session.SessionID || newRound.Attempt != 0 || newRound.RemainingMS != 30000 {
		t.Fatalf("new round %+v: %v", newRound, err)
	}
}

func TestPlaymakerAssistAtomicAndConcurrentRetries(t *testing.T) {
	db, now := basketballTestDB(t)
	session, err := db.CreatePlaymakerSession(42, 7, 0, "device", now)
	if err != nil {
		t.Fatal(err)
	}
	// Two copies of the same request race; only one can increase the wallet.
	var wg sync.WaitGroup
	results := make(chan PlaymakerPassResult, 2)
	errorsCh := make(chan error, 2)
	for n := 0; n < 2; n++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := db.ResolvePlaymakerPass(42, 7, 0, "device", session.SessionID, 0, 0, -98, -218, now.Add(1400*time.Millisecond))
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
		if !result.Assisted || !result.Clean || result.Reason != "assist" || result.Wallet != 1 || result.PlaymakerEarned != 1 || result.NextAttempt != 1 {
			t.Fatalf("assist %+v", result)
		}
	}
	if credited != 1 {
		t.Fatalf("duplicate credit %d", credited)
	}
	resumed, err := db.CreatePlaymakerSession(42, 7, 0, "device", now.Add(2*time.Second))
	if err != nil || resumed.Attempt != 1 {
		t.Fatalf("cursor %+v: %v", resumed, err)
	}
	// Receipts remain retryable after both the event and the grace window close.
	if _, err := db.c.Exec(`UPDATE events SET is_active=0,is_concluded=1 WHERE id=42`); err != nil {
		t.Fatal(err)
	}
	retry, err := db.ResolvePlaymakerPass(42, 7, 0, "device", session.SessionID, 0, 999999, math.NaN(), 0, now.Add(time.Hour))
	if err != nil || !retry.Assisted || retry.CoinsEarned != 0 || retry.Wallet != 1 || retry.PlaymakerEarned != 1 || retry.RemainingMS != 0 {
		t.Fatalf("late retry %+v: %v", retry, err)
	}
	rewards, err := db.ListFanJourneyRewards(42, 7, 0, "device")
	if err != nil || rewards["playmaker"] != 1 {
		t.Fatalf("progress %+v: %v", rewards, err)
	}
}

func TestPlaymakerMissCannotBecomeAnAssistOnRetry(t *testing.T) {
	db, now := basketballTestDB(t)
	session, err := db.CreatePlaymakerSession(42, 7, 0, "device", now)
	if err != nil {
		t.Fatal(err)
	}
	result, err := db.ResolvePlaymakerPass(42, 7, 0, "device", session.SessionID, 0, 0, 0, -218, now.Add(1400*time.Millisecond))
	if err != nil || result.Assisted || result.Clean || result.Reason != "off_target" || result.Wallet != 0 || result.CoinsEarned != 0 || result.NextAttempt != 1 {
		t.Fatalf("miss %+v: %v", result, err)
	}
	retry, err := db.ResolvePlaymakerPass(42, 7, 0, "device", session.SessionID, 0, 0, -98, -218, now.Add(2*time.Second))
	if err != nil || retry.Assisted || retry.Reason != "off_target" || retry.CoinsEarned != 0 {
		t.Fatalf("miss retry %+v: %v", retry, err)
	}
	rewards, err := db.ListFanJourneyRewards(42, 7, 0, "device")
	if err != nil || rewards["playmaker"] != 0 {
		t.Fatalf("miss progress %+v: %v", rewards, err)
	}
}

func TestPlaymakerScopeOrderAndTiming(t *testing.T) {
	db, now := basketballTestDB(t)
	session, err := db.CreatePlaymakerSession(42, 7, 0, "device", now)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		event, org int
		device     string
	}{{42, 8, "device"}, {43, 7, "device"}, {42, 7, "other"}} {
		if _, err := db.ResolvePlaymakerPass(tc.event, tc.org, 0, tc.device, session.SessionID, 0, 0, -98, -218, now.Add(2*time.Second)); !errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("scope %+v: %v", tc, err)
		}
	}
	if _, err := db.ResolvePlaymakerPass(42, 7, 0, "device", session.SessionID, 1, 0, -98, -218, now.Add(2*time.Second)); !errors.Is(err, ErrPlaymakerAttempt) {
		t.Fatalf("out of order %v", err)
	}
	if _, err := db.ResolvePlaymakerPass(42, 7, 0, "device", session.SessionID, 0, 0, -98, -218, now.Add(time.Second)); !errors.Is(err, ErrPlaymakerTooEarly) {
		t.Fatalf("early %v", err)
	}
	for _, elapsed := range []int{-1, 30001, 5000} {
		if _, err := db.ResolvePlaymakerPass(42, 7, 0, "device", session.SessionID, 0, elapsed, -98, -218, now.Add(1400*time.Millisecond)); !errors.Is(err, ErrPlaymakerGesture) {
			t.Fatalf("invalid elapsed %d: %v", elapsed, err)
		}
	}
	if _, err := db.ResolvePlaymakerPass(42, 7, 0, "device", session.SessionID, 0, 0, 0, -10, now.Add(1400*time.Millisecond)); !errors.Is(err, ErrPlaymakerGesture) {
		t.Fatalf("cancelled gesture %v", err)
	}
	if _, err := db.ResolvePlaymakerPass(42, 7, 0, "device", session.SessionID, 0, 0, -98, -218, now.Add(1400*time.Millisecond)); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ResolvePlaymakerPass(42, 7, 0, "device", session.SessionID, 1, 2000, -98, -218, now.Add(2*time.Second)); !errors.Is(err, ErrPlaymakerTooEarly) {
		t.Fatalf("second pass too early %v", err)
	}
	if _, err := db.ResolvePlaymakerPass(42, 7, 0, "device", session.SessionID, 1, 30000, -98, -218, now.Add(46*time.Second)); !errors.Is(err, ErrPlaymakerExpired) {
		t.Fatalf("expired %v", err)
	}
	// A valid last pass may finish its animation and network trip after 30s.
	late, err := db.ResolvePlaymakerPass(42, 7, 0, "device", session.SessionID, 1, 30000, 0, -218, now.Add(45*time.Second))
	if err != nil || late.RemainingMS != 0 || late.NextAttempt != 2 {
		t.Fatalf("grace window %+v: %v", late, err)
	}
}

func TestPlaymakerRegisteredProgressAndGuestTransfer(t *testing.T) {
	db, now := basketballTestDB(t)
	session, err := db.CreatePlaymakerSession(42, 7, 0, "device", now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.ResolvePlaymakerPass(42, 7, 0, "device", session.SessionID, 0, 0, -98, -218, now.Add(1400*time.Millisecond)); err != nil {
		t.Fatal(err)
	}
	profile, err := db.RegisterFan(FanRegisterInput{OrganizationID: 7, EventID: 42, DeviceID: "device", SessionToken: "session", Nickname: "Fan", Phone: "123", AcceptedTerms: true, GuestCoins: 1})
	if err != nil || profile.Wallet != 1 {
		t.Fatalf("registration %+v: %v", profile, err)
	}
	fanID := profile.Profile.ID
	rewards, err := db.ListFanJourneyRewards(42, 7, fanID, "new-device")
	if err != nil || rewards["playmaker"] != 1 {
		t.Fatalf("portable progress %+v: %v", rewards, err)
	}
	if _, err := db.ResolvePlaymakerPass(42, 7, 0, "device", session.SessionID, 0, 0, -98, -218, now.Add(2*time.Second)); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("owned session guest retry %v", err)
	}
	if err := db.AddFanWalletCoins(fanID, 20); err != nil {
		t.Fatal(err)
	}
	second, err := db.CreatePlaymakerSession(42, 7, fanID, "new-device", now.Add(3*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	result, err := db.ResolvePlaymakerPass(42, 7, fanID, "new-device", second.SessionID, 0, 0, -98, -218, now.Add(4400*time.Millisecond))
	if err != nil || result.Wallet != 22 || result.PlaymakerEarned != 2 || result.CoinsEarned != 1 {
		t.Fatalf("registered increment %+v: %v", result, err)
	}
	if _, err := db.CreatePlaymakerSession(44, 8, fanID, "device", now); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("cross-org fan %v", err)
	}
	other, err := db.ListFanJourneyRewards(43, 7, fanID, "new-device")
	if err != nil || other["playmaker"] != 0 {
		t.Fatalf("different event %+v: %v", other, err)
	}
	// A later round is another earning opportunity, with no once-per-game cap.
	third, err := db.CreatePlaymakerSession(42, 7, fanID, "new-device", now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	result, err = db.ResolvePlaymakerPass(42, 7, fanID, "new-device", third.SessionID, 0, 0, -98, -218, now.Add(time.Minute+1400*time.Millisecond))
	if err != nil || result.Wallet != 23 || result.PlaymakerEarned != 3 {
		t.Fatalf("next round %+v: %v", result, err)
	}
}

func TestPlaymakerScoringGolden(t *testing.T) {
	// Golden opening scene: mates (82,248)/(278,248), defender
	// (208.76553231625218,350). The right lane is intercepted; left is open.
	for _, tc := range []struct {
		dx, dy          float64
		assisted, clean bool
		reason          string
	}{{-98, -218, true, true, "assist"}, {98, -218, false, false, "intercepted"}, {0, -218, false, false, "off_target"}, {-63, -218, true, false, "assist"}, {-62, -218, false, false, "off_target"}} {
		assisted, clean, reason, err := evaluatePlaymakerPass(0, 0, tc.dx, tc.dy)
		if err != nil || assisted != tc.assisted || clean != tc.clean || reason != tc.reason {
			t.Fatalf("vector %+v: %t/%t %s %v", tc, assisted, clean, reason, err)
		}
	}
	if _, _, _, err := evaluatePlaymakerPass(0, 0, math.NaN(), -218); !errors.Is(err, ErrPlaymakerGesture) {
		t.Fatalf("nonfinite %v", err)
	}
	// Additional fixtures exported from the actual JS physics module verify
	// movement, rising difficulty and the second defender without duplicating
	// their calculations in the tests.
	for _, tc := range []struct {
		attempt, elapsed int
		dx               float64
		assisted         bool
		reason           string
	}{
		{4, 7200, -97.39283190629584, true, "assist"},
		{8, 12000, -95.58755240122574, false, "intercepted"},
		{8, 12000, 95.58755240122576, true, "assist"},
		{12, 19000, -102.1796408692725, true, "assist"},
		{20, 29000, 109.12542852717053, true, "assist"},
	} {
		assisted, clean, reason, err := evaluatePlaymakerPass(tc.attempt, tc.elapsed, tc.dx, -218)
		if err != nil || assisted != tc.assisted || clean != tc.assisted || reason != tc.reason {
			t.Fatalf("moving golden %+v: %t/%t %s %v", tc, assisted, clean, reason, err)
		}
	}
}

func TestPlaymakerGraceAcceptsOnlyOneNewReceipt(t *testing.T) {
	db, now := basketballTestDB(t)
	session, err := db.CreatePlaymakerSession(42, 7, 0, "device", now)
	if err != nil {
		t.Fatal(err)
	}
	first, err := db.ResolvePlaymakerPass(42, 7, 0, "device", session.SessionID, 0, 29000, 0, -218, now.Add(31*time.Second))
	if err != nil || first.NextAttempt != 1 {
		t.Fatalf("last in-flight pass %+v: %v", first, err)
	}
	if _, err := db.ResolvePlaymakerPass(42, 7, 0, "device", session.SessionID, 1, 30000, 0, -218, now.Add(33*time.Second)); !errors.Is(err, ErrPlaymakerExpired) {
		t.Fatalf("grace must not permit another pass: %v", err)
	}
	retry, err := db.ResolvePlaymakerPass(42, 7, 0, "device", session.SessionID, 0, 29000, 0, -218, now.Add(time.Minute))
	if err != nil || retry.Reason != first.Reason || retry.CoinsEarned != 0 || retry.NextAttempt != 1 {
		t.Fatalf("saved grace result stays retryable %+v: %v", retry, err)
	}
}

func TestPlaymakerResumeRestoresSessionAssistsAndStreak(t *testing.T) {
	db, now := basketballTestDB(t)
	session, err := db.CreatePlaymakerSession(42, 7, 0, "device", now)
	if err != nil || session.Assists != 0 || session.Streak != 0 || session.BestStreak != 0 {
		t.Fatalf("fresh session %+v: %v", session, err)
	}
	for attempt := 0; attempt < 2; attempt++ {
		result, err := db.ResolvePlaymakerPass(42, 7, 0, "device", session.SessionID, attempt, 0, -98, -218, now.Add(time.Duration(attempt+1)*1400*time.Millisecond))
		if err != nil || !result.Assisted {
			t.Fatalf("assist %d %+v: %v", attempt, result, err)
		}
	}
	resumed, err := db.CreatePlaymakerSession(42, 7, 0, "device", now.Add(3*time.Second))
	if err != nil || resumed.SessionID != session.SessionID || resumed.Attempt != 2 || resumed.Assists != 2 || resumed.Streak != 2 || resumed.BestStreak != 2 {
		t.Fatalf("active streak %+v: %v", resumed, err)
	}
	if result, err := db.ResolvePlaymakerPass(42, 7, 0, "device", session.SessionID, 2, 0, -98, -218, now.Add(4200*time.Millisecond)); err != nil || !result.Assisted {
		t.Fatalf("third assist %+v: %v", result, err)
	}
	if result, err := db.ResolvePlaymakerPass(42, 7, 0, "device", session.SessionID, 3, 0, 0, -218, now.Add(5600*time.Millisecond)); err != nil || result.Assisted {
		t.Fatalf("miss %+v: %v", result, err)
	}
	resumed, err = db.CreatePlaymakerSession(42, 7, 0, "device", now.Add(6*time.Second))
	if err != nil || resumed.Attempt != 4 || resumed.Assists != 3 || resumed.Streak != 0 || resumed.BestStreak != 3 {
		t.Fatalf("restored best streak %+v: %v", resumed, err)
	}
	newRound, err := db.CreatePlaymakerSession(42, 7, 0, "device", now.Add(31*time.Second))
	if err != nil || newRound.Assists != 0 || newRound.Streak != 0 || newRound.BestStreak != 0 {
		t.Fatalf("new round reset %+v: %v", newRound, err)
	}
}
