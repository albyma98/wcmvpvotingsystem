package database

import (
	"database/sql"
	"errors"
	"testing"
	"time"
)

func TestTapVolleyMigrationAndSeparateLiveRewards(t *testing.T) {
	db, club, _, event, _ := fanEconomyFixture(t)
	first := fanEconomyRegister(t, db, club.ID, event, "winner", "+3911111001", "winner-session")
	second := fanEconomyRegister(t, db, club.ID, event, "rival", "+3911111002", "rival-session")
	now := time.Now().UTC().Add(-time.Minute)
	old, err := db.CreateTapLiveMatch(event, club.ID, "old-basket-match", first.Profile.ID, second.Profile.ID, now, now.Add(3*time.Second), now.Add(13*time.Second))
	if err != nil || old.GameID != "tap" {
		t.Fatalf("default match: %+v %v", old, err)
	}
	// Only this temporary test database is changed to the pre-variant schema.
	if _, err := db.c.Exec(`ALTER TABLE tap_live_matches DROP COLUMN game_id`); err != nil {
		t.Fatal(err)
	}
	if _, err := New(db.c); err != nil {
		t.Fatal(err)
	}
	restored, err := db.GetTapLiveMatchByID(old.MatchID)
	if err != nil || restored.GameID != "tap" {
		t.Fatalf("legacy migration: %+v %v", restored, err)
	}
	finish := func(id string) {
		t.Helper()
		if err := db.SubmitTapLiveScore(id, first.Profile.ID, 10); err != nil {
			t.Fatal(err)
		}
		if err := db.SubmitTapLiveScore(id, second.Profile.ID, 5); err != nil {
			t.Fatal(err)
		}
		if err := db.TryFinalizeTapLiveMatch(id); err != nil {
			t.Fatal(err)
		}
		if err := db.TryFinalizeTapLiveMatch(id); err != nil {
			t.Fatal(err)
		}
	}
	finish(old.MatchID)
	volley, err := db.CreateTapLiveMatch(event, club.ID, "volley-match", first.Profile.ID, second.Profile.ID, now, now.Add(3*time.Second), now.Add(13*time.Second), "tap-volley")
	if err != nil || volley.GameID != "tap-volley" {
		t.Fatalf("volley match: %+v %v", volley, err)
	}
	if _, err := db.GetOpenTapLiveMatchByFan(event, first.Profile.ID, "tap"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("basket saw volley match: %v", err)
	}
	if active, err := db.GetOpenTapLiveMatchByFan(event, first.Profile.ID, "tap-volley"); err != nil || active.MatchID != volley.MatchID {
		t.Fatalf("active volley: %+v %v", active, err)
	}
	finish(volley.MatchID)
	latest, err := db.GetLatestTapLiveMatchByFan(event, first.Profile.ID, "tap")
	if err != nil || latest.MatchID != old.MatchID {
		t.Fatalf("basket history: %+v %v", latest, err)
	}
	receipt, err := db.ClaimFanMinigameReward(event, club.ID, first.Profile.ID, "winner", "tap-volley", "volley-round-123")
	if err != nil || receipt.CoinsEarned != 8 || receipt.RewardTotal != 38 || receipt.Wallet != 68 {
		t.Fatalf("volley reward: %+v %v", receipt, err)
	}
	receipt, err = db.ClaimFanMinigameReward(event, club.ID, first.Profile.ID, "winner", "tap-volley", "volley-round-123")
	if err != nil || receipt.CoinsEarned != 0 || receipt.Wallet != 68 {
		t.Fatalf("retry: %+v %v", receipt, err)
	}
	rewards, err := db.ListFanJourneyRewards(event, club.ID, first.Profile.ID, "other-device")
	if err != nil || rewards["tap"] != 30 || rewards["tap-volley"] != 38 {
		t.Fatalf("separate progress: %+v %v", rewards, err)
	}
}
