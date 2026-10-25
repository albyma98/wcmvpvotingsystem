package database

import (
	"database/sql"
	"errors"
	"math"
	"testing"
	"time"
)

func volleyballTestDB(t *testing.T) (*appdbimpl, time.Time) {
	t.Helper()
	db, now := basketballTestDB(t)
	if err := ensureVolleyballTables(db.c); err != nil {
		t.Fatal(err)
	}
	return db, now
}

func TestVolleyballAtomicRewardsAndReplay(t *testing.T) {
	db, now := volleyballTestDB(t)
	wallet := 0
	for _, game := range []string{"volley-rush", "setter-pro", "monster-block"} {
		for attempt := 0; attempt < 8; attempt++ {
			start := now.Add(time.Duration(attempt) * 2 * time.Second)
			round, err := db.CreateVolleyballRound(42, 7, 0, "device", game, start)
			if err != nil {
				t.Fatal(err)
			}
			target, blocker := volleyballTargetForAttempt(game, attempt)
			if round.Attempt != attempt || round.Target != target || round.Blocker != blocker || round.GameID != game {
				t.Fatalf("target %+v", round)
			}
			repeat, err := db.CreateVolleyballRound(42, 7, 0, "device", game, start.Add(100*time.Millisecond))
			if err != nil || repeat.RoundID != round.RoundID {
				t.Fatalf("pending %+v: %v", repeat, err)
			}
			result, err := db.ResolveVolleyballRound(42, 7, 0, "device", game, round.RoundID, target.X, target.Y, round.TimingTarget, start.Add(time.Second))
			wallet++
			if err != nil || !result.Success || !result.Perfect || result.CoinsEarned != 1 || result.Wallet != wallet || result.GameEarned != attempt+1 {
				t.Fatalf("game %s result %+v: %v", game, result, err)
			}
			reloaded := &appdbimpl{c: db.c}
			retry, err := reloaded.ResolveVolleyballRound(42, 7, 0, "device", game, round.RoundID, 0, 560, 0, start.Add(1100*time.Millisecond))
			if err != nil || !retry.Success || !retry.Perfect || retry.CoinsEarned != 0 || retry.Wallet != wallet || retry.GameEarned != attempt+1 {
				t.Fatalf("retry %+v: %v", retry, err)
			}
		}
	}
	rewards, err := db.ListFanJourneyRewards(42, 7, 0, "device")
	if err != nil || rewards["volley-rush"] != 8 || rewards["setter-pro"] != 8 || rewards["monster-block"] != 8 {
		t.Fatalf("journey %+v: %v", rewards, err)
	}
	var ledgerCount, points int
	if err := db.c.QueryRow(`SELECT COUNT(*) FROM fan_wallet_movements WHERE source IN ('volley-rush','setter-pro','monster-block')`).Scan(&ledgerCount); err != nil {
		t.Fatal(err)
	}
	if err := db.c.QueryRow(`SELECT points FROM fan_economy_accounts WHERE organization_id=7 AND owner_key='guest:42:device'`).Scan(&points); err != nil || ledgerCount != 24 || points != 24 {
		t.Fatalf("ledger=%d points=%d: %v", ledgerCount, points, err)
	}
}

func TestVolleyballScopeTimingAndExpiry(t *testing.T) {
	db, now := volleyballTestDB(t)
	round, err := db.CreateVolleyballRound(42, 7, 0, "device", "volley-rush", now)
	if err != nil {
		t.Fatal(err)
	}
	for _, scope := range []struct {
		event, org   int
		device, game string
	}{{43, 7, "device", "volley-rush"}, {42, 8, "device", "volley-rush"}, {42, 7, "other", "volley-rush"}, {42, 7, "device", "setter-pro"}} {
		if _, err := db.ResolveVolleyballRound(scope.event, scope.org, 0, scope.device, scope.game, round.RoundID, round.Target.X, round.Target.Y, .7, now.Add(time.Second)); !errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("scope %+v: %v", scope, err)
		}
	}
	if _, err := db.ResolveVolleyballRound(42, 7, 0, "device", "volley-rush", round.RoundID, round.Target.X, round.Target.Y, .7, now.Add(599*time.Millisecond)); !errors.Is(err, ErrVolleyballTooEarly) {
		t.Fatalf("early %v", err)
	}
	for _, values := range [][3]float64{{math.NaN(), 150, .7}, {100, math.Inf(1), .7}, {100, 150, math.Inf(-1)}, {-1, 150, .7}, {361, 150, .7}, {100, 561, .7}, {100, 150, -0.1}, {100, 150, 1.1}} {
		if _, err := db.ResolveVolleyballRound(42, 7, 0, "device", "volley-rush", round.RoundID, values[0], values[1], values[2], now.Add(time.Second)); !errors.Is(err, ErrVolleyballGesture) {
			t.Fatalf("gesture %v: %v", values, err)
		}
	}
	miss, err := db.ResolveVolleyballRound(42, 7, 0, "device", "volley-rush", round.RoundID, round.Target.X, round.Target.Y, .1, now.Add(600*time.Millisecond))
	if err != nil || miss.Success || miss.Perfect || miss.Wallet != 0 || miss.CoinsEarned != 0 {
		t.Fatalf("bad timing %+v: %v", miss, err)
	}
	retry, err := db.ResolveVolleyballRound(42, 7, 0, "device", "volley-rush", round.RoundID, round.Target.X, round.Target.Y, .7, now.Add(time.Second))
	if err != nil || retry.Success || retry.CoinsEarned != 0 {
		t.Fatalf("miss replay %+v: %v", retry, err)
	}
	if _, err := db.CreateVolleyballRound(42, 7, 0, "device", "unknown", now); !errors.Is(err, ErrVolleyballGame) {
		t.Fatalf("unknown game %v", err)
	}
	next, err := db.CreateVolleyballRound(42, 7, 0, "device", "volley-rush", now.Add(2*time.Second))
	if err != nil || next.Attempt != 1 {
		t.Fatalf("next %+v: %v", next, err)
	}
	if _, err := db.ResolveVolleyballRound(42, 7, 0, "device", "volley-rush", next.RoundID, next.Target.X, next.Target.Y, .7, now.Add(122*time.Second)); !errors.Is(err, ErrVolleyballExpired) {
		t.Fatalf("expired %v", err)
	}
	renewed, err := db.CreateVolleyballRound(42, 7, 0, "device", "volley-rush", now.Add(112*time.Second))
	if err != nil || renewed.RoundID == next.RoundID || renewed.Attempt != 2 {
		t.Fatalf("renewed %+v: %v", renewed, err)
	}
	if _, err := db.ResolveVolleyballRound(42, 7, 0, "device", "volley-rush", next.RoundID, next.Target.X, next.Target.Y, .7, now.Add(113*time.Second)); !errors.Is(err, ErrVolleyballExpired) {
		t.Fatalf("retired round %v", err)
	}
	if _, err := db.c.Exec(`UPDATE events SET is_concluded=1 WHERE id=42`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ResolveVolleyballRound(42, 7, 0, "device", "volley-rush", renewed.RoundID, renewed.Target.X, renewed.Target.Y, .7, now.Add(113*time.Second)); !errors.Is(err, ErrVolleyballClosed) {
		t.Fatalf("closed resolution %v", err)
	}
	if _, err := db.CreateVolleyballRound(42, 7, 0, "device", "setter-pro", now); !errors.Is(err, ErrVolleyballClosed) {
		t.Fatalf("closed create %v", err)
	}
}

func TestVolleyballRegisteredAndGuestTransfer(t *testing.T) {
	db, now := volleyballTestDB(t)
	first, err := db.CreateVolleyballRound(42, 7, 0, "device", "setter-pro", now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.ResolveVolleyballRound(42, 7, 0, "device", "setter-pro", first.RoundID, first.Target.X, first.Target.Y, .7, now.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	pending, err := db.CreateVolleyballRound(42, 7, 0, "device", "setter-pro", now.Add(2*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	profile, err := db.RegisterFan(FanRegisterInput{OrganizationID: 7, EventID: 42, DeviceID: "device", SessionToken: "session", Nickname: "VolleyFan", Phone: "123", AcceptedTerms: true, GuestCoins: 1})
	if err != nil || profile.Wallet != 1 {
		t.Fatalf("registration %+v: %v", profile, err)
	}
	fanID := profile.Profile.ID
	result, err := db.ResolveVolleyballRound(42, 7, fanID, "new-device", "setter-pro", pending.RoundID, pending.Target.X, pending.Target.Y, .7, now.Add(3*time.Second))
	if err != nil || !result.Success || result.CoinsEarned != 1 || result.Wallet != 2 || result.GameEarned != 2 {
		t.Fatalf("portable pending %+v: %v", result, err)
	}
	retry, err := db.ResolveVolleyballRound(42, 7, fanID, "third-device", "setter-pro", first.RoundID, 0, 0, 0, now.Add(4*time.Second))
	if err != nil || !retry.Success || retry.CoinsEarned != 0 || retry.Wallet != 2 || retry.GameEarned != 2 {
		t.Fatalf("portable retry %+v: %v", retry, err)
	}
	for _, id := range []string{first.RoundID, pending.RoundID} {
		if _, err := db.ResolveVolleyballRound(42, 7, 0, "device", "setter-pro", id, 0, 0, .7, now.Add(4*time.Second)); !errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("guest replay %v", err)
		}
	}
	if _, err := db.CreateVolleyballRound(44, 8, fanID, "device", "setter-pro", now); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("cross organization fan %v", err)
	}
	rewards, err := db.ListFanJourneyRewards(42, 7, fanID, "third-device")
	if err != nil || rewards["setter-pro"] != 2 {
		t.Fatalf("portable journey %+v: %v", rewards, err)
	}
}

func TestVolleyballRewardRollback(t *testing.T) {
	db, now := volleyballTestDB(t)
	round, err := db.CreateVolleyballRound(42, 7, 0, "device", "monster-block", now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.c.Exec(`CREATE TRIGGER reject_volleyball_wallet BEFORE INSERT ON guest_wallets BEGIN SELECT RAISE(ABORT,'wallet rejected'); END`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ResolveVolleyballRound(42, 7, 0, "device", "monster-block", round.RoundID, round.Target.X, round.Target.Y, .7, now.Add(time.Second)); err == nil {
		t.Fatal("expected wallet failure")
	}
	var status string
	if err := db.c.QueryRow(`SELECT status FROM volleyball_rounds WHERE round_id=?`, round.RoundID).Scan(&status); err != nil || status != "pending" {
		t.Fatalf("receipt must roll back: %s %v", status, err)
	}
	if _, err := db.c.Exec(`DROP TRIGGER reject_volleyball_wallet`); err != nil {
		t.Fatal(err)
	}
	result, err := db.ResolveVolleyballRound(42, 7, 0, "device", "monster-block", round.RoundID, round.Target.X, round.Target.Y, .7, now.Add(2*time.Second))
	if err != nil || result.CoinsEarned != 1 || result.Wallet != 1 {
		t.Fatalf("retry after rollback %+v: %v", result, err)
	}
}

func TestVolleyballCourtScoring(t *testing.T) {
	for _, tc := range []struct {
		game           string
		rx, ry, window float64
	}{{"volley-rush", 48, 40, .24}, {"setter-pro", 34, 30, .20}, {"monster-block", 44, 34, .16}} {
		target, blocker := volleyballTargetForAttempt(tc.game, 0)
		for _, shot := range []struct {
			dx, dy, timing   float64
			success, perfect bool
		}{{0, 0, .7, true, true}, {tc.rx, 0, .7, true, false}, {tc.rx + .01, 0, .7, false, false}, {0, tc.ry, .7, true, false}, {0, tc.ry + .01, .7, false, false}, {0, 0, .7 - tc.window, true, false}, {0, 0, .7 - tc.window - .001, false, false}} {
			success, perfect, err := volleyballEvaluateRound(tc.game, target, blocker, .7, target.X+shot.dx, target.Y+shot.dy, shot.timing)
			if err != nil || success != shot.success || perfect != shot.perfect {
				t.Fatalf("%s %+v: %t/%t %v", tc.game, shot, success, perfect, err)
			}
		}
	}
	target := VolleyballPoint{82, 136}
	interceptX := 180 + (target.X-180)*(466-285)/(466-target.Y)
	success, _, err := volleyballEvaluateRound("volley-rush", target, VolleyballPoint{interceptX, 285}, .7, target.X, target.Y, .7)
	if err != nil || success {
		t.Fatalf("spike into blocker %t: %v", success, err)
	}
}
