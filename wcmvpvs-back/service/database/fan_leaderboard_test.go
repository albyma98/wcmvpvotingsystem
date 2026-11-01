package database

import (
	"fmt"
	"testing"
	"time"
)

func TestFanLeaderboardCompleteListKeepsClubScopeAndDeterministicRanks(t *testing.T) {
	db, club, other, event, _ := fanEconomyFixture(t)
	for index := 0; index < 6; index++ {
		fan := fanEconomyRegister(t, db, club.ID, event, fmt.Sprintf("fan-%d", index), fmt.Sprintf("+39001%d", index), fmt.Sprintf("session-%d", index))
		if err := db.AddFanWalletCoins(fan.Profile.ID, 100); err != nil {
			t.Fatal(err)
		}
	}
	outsider := fanEconomyRegister(t, db, other.ID, 0, "outsider", "+3900199", "outsider-session")
	if err := db.AddFanWalletCoins(outsider.Profile.ID, 999); err != nil {
		t.Fatal(err)
	}
	all, err := db.GetFanLeaderboard(event, club.ID, -1)
	if err != nil || len(all) != 6 {
		t.Fatalf("ranking truncated: %+v, %v", all, err)
	}
	for index, row := range all {
		if row.Rank != index+1 || row.Points != 100 || row.FanID == outsider.Profile.ID || (index > 0 && row.FanID <= all[index-1].FanID) {
			t.Fatalf("club scope or tie order: %+v", all)
		}
	}
	top, err := db.GetFanLeaderboard(event, club.ID, 3)
	if err != nil || len(top) != 3 || top[2].FanID != all[2].FanID {
		t.Fatalf("legacy podium changed: %+v, %v", top, err)
	}
}

func TestFanEventLeaderboardSeparatesEventsAndPreservesEarnedPoints(t *testing.T) {
	db, club, other, event, next := fanEconomyFixture(t)
	first := fanEconomyRegister(t, db, club.ID, event, "first", "+390101", "first-session")
	second := fanEconomyRegister(t, db, club.ID, event, "second", "+390102", "second-session")
	third := fanEconomyRegister(t, db, club.ID, event, "third", "+390103", "third-session")
	outsider := fanEconomyRegister(t, db, other.ID, 0, "outsider", "+390104", "other-session")
	// Write through the real wallet context and triggers used by game rewards.
	earn := func(eventID, org, fan, amount int) {
		t.Helper()
		tx, err := db.c.Begin()
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback()
		if _, err := economyExec(tx, org, eventID, fan, "", "Premio", "test:reward", `UPDATE fan_wallets SET coins=coins+? WHERE fan_id=?`, amount, fan); err != nil {
			t.Fatal(err)
		}
		if err := tx.Commit(); err != nil {
			t.Fatal(err)
		}
	}
	earn(event, club.ID, first.Profile.ID, 20)
	earn(next, club.ID, first.Profile.ID, 100)
	earn(event, club.ID, second.Profile.ID, 30)
	earn(event, club.ID, third.Profile.ID, 30)
	earn(event, other.ID, outsider.Profile.ID, 999)
	if err := db.PurchaseFanShopItem(event, club.ID, second.Profile.ID, "second", "club_frame"); err != nil {
		t.Fatal(err)
	}
	rows, err := db.GetFanEventLeaderboard(event, club.ID, -1)
	if err != nil || len(rows) != 3 {
		t.Fatalf("event ranking: %+v, %v", rows, err)
	}
	for index, want := range []struct{ fan, points int }{{second.Profile.ID, 30}, {third.Profile.ID, 30}, {first.Profile.ID, 20}} {
		if rows[index].Rank != index+1 || rows[index].FanID != want.fan || rows[index].Points != want.points {
			t.Fatalf("event totals/tie order: %+v", rows)
		}
	}
	if rows[0].EquippedFrame != "club_frame" {
		t.Fatalf("missing equipped frame: %+v", rows[0])
	}
	nextRows, err := db.GetFanEventLeaderboard(next, club.ID, -1)
	if err != nil || len(nextRows) != 1 || nextRows[0].Points != 100 {
		t.Fatalf("next event: %+v, %v", nextRows, err)
	}
	general, err := db.GetFanLeaderboard(event, club.ID, -1)
	if err != nil || general[0].FanID != first.Profile.ID || general[0].Points != 120 {
		t.Fatalf("general ranking: %+v, %v", general, err)
	}
	wrongClub, err := db.GetFanEventLeaderboard(event, other.ID, -1)
	if err != nil || len(wrongClub) != 0 {
		t.Fatalf("event belongs to another club: %+v, %v", wrongClub, err)
	}
	limited, err := db.GetFanEventLeaderboard(event, club.ID, 1)
	if err != nil || len(limited) != 1 || limited[0].FanID != second.Profile.ID {
		t.Fatalf("limited ranking: %+v, %v", limited, err)
	}
}

func TestFanEventLeaderboardAdoptsGuestPointsWithoutCountingOpeningBalances(t *testing.T) {
	db, club, _, event, next := fanEconomyFixture(t)
	if _, _, err := db.ClaimFanJourneyReward(event, club.ID, 0, "guest", "tap"); err != nil {
		t.Fatal(err)
	}
	if err := db.UpsertGuestCoins(next, club.ID, "guest", 12); err != nil {
		t.Fatal(err)
	}
	// An opening balance is known only at club level, even if an old guest
	// wallet happens to reference this event.
	if _, err := db.c.Exec(`INSERT INTO fan_wallet_movements(organization_id,owner_key,event_id,device_id,delta,balance,label,source) VALUES(?,'guest-opening',?,'guest',500,500,'Saldo iniziale','opening')`, club.ID, event); err != nil {
		t.Fatal(err)
	}
	guestRows, err := db.GetFanEventLeaderboard(event, club.ID, -1)
	if err != nil || len(guestRows) != 0 {
		t.Fatalf("anonymous fans must not appear: %+v, %v", guestRows, err)
	}
	fan := fanEconomyRegister(t, db, club.ID, event, "guest", "+390105", "guest-session")
	rows, err := db.GetFanEventLeaderboard(event, club.ID, -1)
	if err != nil || len(rows) != 1 || rows[0].FanID != fan.Profile.ID || rows[0].Points != 8 {
		t.Fatalf("adopted event points: %+v, %v", rows, err)
	}
	rows, err = db.GetFanEventLeaderboard(next, club.ID, -1)
	if err != nil || len(rows) != 1 || rows[0].Points != 12 {
		t.Fatalf("adopted points keep their original event: %+v, %v", rows, err)
	}
}

func TestFanEventLeaderboardCountsFinalizedTapLivePayoutsOnce(t *testing.T) {
	db, club, _, event, _ := fanEconomyFixture(t)
	first := fanEconomyRegister(t, db, club.ID, event, "first", "+390106", "first-session")
	second := fanEconomyRegister(t, db, club.ID, event, "second", "+390107", "second-session")
	now := time.Now()
	if _, err := db.CreateTapLiveMatch(event, club.ID, "ranking-live", first.Profile.ID, second.Profile.ID, now, now, now); err != nil {
		t.Fatal(err)
	}
	if err := db.SubmitTapLiveScore("ranking-live", first.Profile.ID, 20); err != nil {
		t.Fatal(err)
	}
	if err := db.SubmitTapLiveScore("ranking-live", second.Profile.ID, 10); err != nil {
		t.Fatal(err)
	}
	for attempt := 0; attempt < 2; attempt++ {
		if err := db.TryFinalizeTapLiveMatch("ranking-live"); err != nil {
			t.Fatal(err)
		}
	}
	rows, err := db.GetFanEventLeaderboard(event, club.ID, -1)
	if err != nil || len(rows) != 2 || rows[0].Points != 30 || rows[1].Points != 8 {
		t.Fatalf("live payout attribution: %+v, %v", rows, err)
	}
}
