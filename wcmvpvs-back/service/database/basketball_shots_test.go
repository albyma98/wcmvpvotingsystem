package database

import (
	"database/sql"
	"errors"
	"math"
	"path/filepath"
	"testing"
	"time"
)

func basketballTestDB(t *testing.T) (*appdbimpl, time.Time) {
	t.Helper()
	conn, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "basketball.db"))
	if err != nil {
		t.Fatal(err)
	}
	conn.SetMaxOpenConns(1)
	t.Cleanup(func() { conn.Close() })
	if _, err := conn.Exec(`CREATE TABLE events(id INTEGER PRIMARY KEY,organization_id INTEGER,is_active INTEGER,is_concluded INTEGER);
	CREATE TABLE votes(id INTEGER PRIMARY KEY,event_id INTEGER,device_id TEXT,ticket_code TEXT);
	INSERT INTO events VALUES(42,7,1,0),(43,7,1,0),(44,8,1,0);`); err != nil {
		t.Fatal(err)
	}
	if err := ensureFanProfileTables(conn); err != nil {
		t.Fatal(err)
	}
	return &appdbimpl{c: conn}, time.Date(2026, 10, 15, 18, 0, 0, 0, time.UTC)
}

func TestBasketballAtomicRewardsAndRetries(t *testing.T) {
	db, now := basketballTestDB(t)
	for attempt := 0; attempt < 12; attempt++ {
		start := now.Add(time.Duration(attempt) * 2 * time.Second)
		shot, err := db.CreateBasketballShot(42, 7, 0, "device", start)
		if err != nil {
			t.Fatal(err)
		}
		if shot.Attempt != attempt || shot.Hoop != basketballHoopForAttempt(attempt) {
			t.Fatalf("target %+v", shot)
		}
		repeat, err := db.CreateBasketballShot(42, 7, 0, "device", start.Add(100*time.Millisecond))
		if err != nil || repeat.ShotID != shot.ShotID {
			t.Fatalf("pending %+v: %v", repeat, err)
		}
		result, err := db.ResolveBasketballShot(42, 7, 0, "device", shot.ShotID, shot.Hoop.X-180, shot.Hoop.Y-466, start.Add(time.Second))
		if err != nil || !result.Made || !result.Clean || result.CoinsEarned != 1 || result.Wallet != attempt+1 || result.BasketballEarned != attempt+1 {
			t.Fatalf("made %+v: %v", result, err)
		}
		// Retrying with different aiming data cannot alter a recorded result.
		reloaded := &appdbimpl{c: db.c}
		retry, err := reloaded.ResolveBasketballShot(42, 7, 0, "device", shot.ShotID, 100, -50, start.Add(1100*time.Millisecond))
		if err != nil || !retry.Made || retry.CoinsEarned != 0 || retry.Wallet != attempt+1 || retry.BasketballEarned != attempt+1 {
			t.Fatalf("retry %+v: %v", retry, err)
		}
	}
	rewards, err := db.ListFanJourneyRewards(42, 7, 0, "device")
	if err != nil || rewards["basketball"] != 12 {
		t.Fatalf("progress %+v: %v", rewards, err)
	}
	other, err := db.ListFanJourneyRewards(43, 7, 0, "device")
	if err != nil || len(other) != 0 {
		t.Fatalf("other event %+v: %v", other, err)
	}
}

func TestBasketballMissScopeAndTiming(t *testing.T) {
	db, now := basketballTestDB(t)
	shot, err := db.CreateBasketballShot(42, 7, 0, "device", now)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		event, org int
		device     string
	}{{42, 8, "device"}, {43, 7, "device"}, {42, 7, "other"}} {
		if _, err := db.ResolveBasketballShot(tc.event, tc.org, 0, tc.device, shot.ShotID, 0, -266, now.Add(time.Second)); !errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("scope %+v: %v", tc, err)
		}
	}
	if _, err := db.ResolveBasketballShot(42, 7, 0, "device", shot.ShotID, 0, -266, now.Add(100*time.Millisecond)); !errors.Is(err, ErrBasketballTooEarly) {
		t.Fatalf("early %v", err)
	}
	if _, err := db.ResolveBasketballShot(42, 7, 0, "device", shot.ShotID, math.NaN(), -266, now.Add(time.Second)); !errors.Is(err, ErrBasketballGesture) {
		t.Fatalf("nonfinite %v", err)
	}
	if _, err := db.ResolveBasketballShot(42, 7, 0, "device", shot.ShotID, 0, -10, now.Add(time.Second)); !errors.Is(err, ErrBasketballGesture) {
		t.Fatalf("cancelled gesture %v", err)
	}
	miss, err := db.ResolveBasketballShot(42, 7, 0, "device", shot.ShotID, 110, -266, now.Add(time.Second))
	if err != nil || miss.Made || miss.CoinsEarned != 0 || miss.Wallet != 0 || miss.BasketballEarned != 0 {
		t.Fatalf("miss %+v: %v", miss, err)
	}
	retry, err := db.ResolveBasketballShot(42, 7, 0, "device", shot.ShotID, 0, -266, now.Add(time.Second))
	if err != nil || retry.Made || retry.CoinsEarned != 0 {
		t.Fatalf("miss retry %+v: %v", retry, err)
	}
	next, err := db.CreateBasketballShot(42, 7, 0, "device", now.Add(2*time.Second))
	if err != nil || next.Attempt != 1 {
		t.Fatalf("next %+v: %v", next, err)
	}
	if _, err := db.ResolveBasketballShot(42, 7, 0, "device", next.ShotID, -60, -261, now.Add(3*time.Minute)); !errors.Is(err, ErrBasketballExpired) {
		t.Fatalf("expired %v", err)
	}
	newShot, err := db.CreateBasketballShot(42, 7, 0, "device", now.Add(3*time.Minute))
	if err != nil || newShot.ShotID == next.ShotID {
		t.Fatalf("renewed %+v: %v", newShot, err)
	}
	if _, err := db.c.Exec(`UPDATE events SET is_concluded=1 WHERE id=42`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ResolveBasketballShot(42, 7, 0, "device", newShot.ShotID, 60, -271, now.Add(3*time.Minute+time.Second)); !errors.Is(err, ErrBasketballClosed) {
		t.Fatalf("closed shot %v", err)
	}
	if _, err := db.CreateBasketballShot(42, 7, 0, "other", now); !errors.Is(err, ErrBasketballClosed) {
		t.Fatalf("closed start %v", err)
	}
}

func TestBasketballRegisteredProgressAndGuestTransfer(t *testing.T) {
	db, now := basketballTestDB(t)
	shot, err := db.CreateBasketballShot(42, 7, 0, "device", now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.ResolveBasketballShot(42, 7, 0, "device", shot.ShotID, 0, -266, now.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	profile, err := db.RegisterFan(FanRegisterInput{OrganizationID: 7, EventID: 42, DeviceID: "device", SessionToken: "session", Nickname: "Fan", Phone: "123", AcceptedTerms: true, GuestCoins: 1})
	if err != nil || profile.Wallet != 1 {
		t.Fatalf("registration %+v: %v", profile, err)
	}
	fanID := profile.Profile.ID
	rewards, err := db.ListFanJourneyRewards(42, 7, fanID, "other-device")
	if err != nil || rewards["basketball"] != 1 {
		t.Fatalf("portable progress %+v: %v", rewards, err)
	}
	shot2, err := db.CreateBasketballShot(42, 7, fanID, "other-device", now.Add(2*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	result, err := db.ResolveBasketballShot(42, 7, fanID, "other-device", shot2.ShotID, 0, -266, now.Add(3*time.Second))
	if err != nil || result.Wallet != 2 || result.BasketballEarned != 2 || result.CoinsEarned != 1 {
		t.Fatalf("registered %+v: %v", result, err)
	}
	if _, err := db.CreateBasketballShot(44, 8, fanID, "device", now); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("cross-org fan %v", err)
	}
	if _, err := db.ResolveBasketballShot(42, 7, 0, "other-device", shot2.ShotID, 0, -266, now.Add(3*time.Second)); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("guest cannot replay owned shot %v", err)
	}
	// Increment the existing wallet, preserving unrelated earnings.
	if err := db.AddFanWalletCoins(fanID, 20); err != nil {
		t.Fatal(err)
	}
	shot3, err := db.CreateBasketballShot(42, 7, fanID, "other-device", now.Add(4*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	result, err = db.ResolveBasketballShot(42, 7, fanID, "other-device", shot3.ShotID, shot3.Hoop.X-180, shot3.Hoop.Y-466, now.Add(5*time.Second))
	if err != nil || result.Wallet != 23 || result.BasketballEarned != 3 {
		t.Fatalf("increment %+v: %v", result, err)
	}
}

func TestBasketballScoringMatchesCourt(t *testing.T) {
	for attempt := 0; attempt < 8; attempt++ {
		hoop := basketballHoopForAttempt(attempt)
		for _, tc := range []struct {
			dx, dy      float64
			made, clean bool
		}{{0, 0, true, true}, {30, 0, true, false}, {31, 0, false, false}, {0, 40, true, false}, {0, 41, false, false}, {6, 8, true, true}} {
			made, clean, err := basketballEvaluateShot(hoop, hoop.X-180+tc.dx, hoop.Y-466+tc.dy)
			if err != nil || made != tc.made || clean != tc.clean {
				t.Fatalf("attempt %d offset %+v: %t/%t %v", attempt, tc, made, clean, err)
			}
		}
	}
}

func TestBasketballRenewsNearlyExpiredBall(t *testing.T) {
	db, now := basketballTestDB(t)
	shot, err := db.CreateBasketballShot(42, 7, 0, "device", now)
	if err != nil {
		t.Fatal(err)
	}
	stillPending, err := db.CreateBasketballShot(42, 7, 0, "device", now.Add(109*time.Second))
	if err != nil || stillPending.ShotID != shot.ShotID {
		t.Fatalf("11 seconds left should keep pending ball: %+v %v", stillPending, err)
	}
	renewed, err := db.CreateBasketballShot(42, 7, 0, "device", now.Add(110*time.Second))
	if err != nil || renewed.ShotID == shot.ShotID || renewed.Attempt != 1 {
		t.Fatalf("10 seconds left should renew ball: %+v %v", renewed, err)
	}
	if _, err := db.ResolveBasketballShot(42, 7, 0, "device", shot.ShotID, 0, -266, now.Add(111*time.Second)); !errors.Is(err, ErrBasketballExpired) {
		t.Fatalf("replaced ball should be expired: %v", err)
	}
}
