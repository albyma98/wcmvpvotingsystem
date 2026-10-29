package database

import (
	"errors"
	"sync"
	"testing"
	"time"
)

func TestArenaRomeDayAndDeterministicTreasure(t *testing.T) {
	for _, tc := range []struct{ instant, day string }{{"2026-03-28T23:30:00Z", "2026-03-29"}, {"2026-03-29T22:30:00Z", "2026-03-30"}, {"2026-10-25T23:30:00Z", "2026-10-26"}} {
		now, _ := time.Parse(time.RFC3339, tc.instant)
		a := arenaTreasure(7, now, 20)
		b := arenaTreasure(7, now, 20)
		if a != b || a.Day != tc.day || a.Location < 0 || a.Location >= 10 || a.Kind < 0 || a.Kind >= 4 {
			t.Fatalf("treasure/timezone: %+v", a)
		}
	}
}
func TestArenaActivityEconomyPredictionSeatsAndPrivacy(t *testing.T) {
	db, club, other, _, event := fanEconomyFixture(t)
	fan := fanEconomyRegister(t, db, club.ID, 0, "arena-fan", "+39003001", "arena-session")
	friend := fanEconomyRegister(t, db, club.ID, 0, "arena-friend", "+39003002", "friend-session")
	outsider := fanEconomyRegister(t, db, other.ID, 0, "arena-other", "+39003003", "other-session")
	now, _ := time.Parse(time.RFC3339, "2026-10-20T12:00:00Z")
	read := func(id int) ArenaActivityState {
		t.Helper()
		s, e := db.ArenaActivities(club.ID, id, ArenaCommand{}, now)
		if e != nil {
			t.Fatal(e)
		}
		return s
	}
	initial := read(fan.Profile.ID)
	if initial.Member == nil || initial.Member.Public || len(initial.Owned) != 0 {
		t.Fatalf("defaults: %+v", initial)
	}
	if _, e := db.ArenaActivities(club.ID, outsider.Profile.ID, ArenaCommand{}, now); e == nil {
		t.Fatal("cross-club identity accepted")
	}
	claim := ArenaCommand{Action: "claim", Day: initial.Treasure.Day, Location: initial.Treasure.Location}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, e := db.ArenaActivities(club.ID, fan.Profile.ID, claim, now); e != nil {
				t.Error(e)
			}
		}()
	}
	wg.Wait()
	claimed := read(fan.Profile.ID)
	if claimed.Wallet != initial.Wallet+20 || !claimed.Treasure.Claimed {
		t.Fatalf("duplicate claim: %+v", claimed)
	}
	if _, e := db.ArenaActivities(club.ID, fan.Profile.ID, ArenaCommand{Action: "claim", Day: "2099-01-01", Location: claim.Location}, now); e == nil {
		t.Fatal("forged day accepted")
	}
	buy := ArenaCommand{Action: "buy", Key: "club-shirt"}
	for i := 0; i < 2; i++ {
		if _, e := db.ArenaActivities(club.ID, fan.Profile.ID, buy, now); e != nil {
			t.Fatal(e)
		}
	}
	bought := read(fan.Profile.ID)
	if bought.Wallet != initial.Wallet || len(bought.Owned) != 1 {
		t.Fatal("purchase not idempotent", bought.Wallet, bought.Owned)
	}
	if _, e := db.ArenaActivities(club.ID, fan.Profile.ID, ArenaCommand{Action: "buy", Key: "away-shirt"}, now); !errors.Is(e, ErrFanShopFunds) {
		t.Fatal("funds validation", e)
	}
	if _, e := db.ArenaActivities(club.ID, fan.Profile.ID, ArenaCommand{Action: "equip", Key: "club-shirt"}, now); e != nil {
		t.Fatal(e)
	}
	if read(fan.Profile.ID).Equipment["shirt"] != "club-shirt" {
		t.Fatal("equipment not persisted")
	}
	if _, e := db.ArenaActivities(club.ID, fan.Profile.ID, ArenaCommand{Action: "predict", EventID: event, Score: "3-1"}, now); e != nil {
		t.Fatal(e)
	}
	closed, _ := time.Parse(time.RFC3339, "2026-10-24T12:00:00Z")
	if _, e := db.ArenaActivities(club.ID, fan.Profile.ID, ArenaCommand{Action: "predict", EventID: event, Score: "3-2"}, closed); e == nil {
		t.Fatal("late prediction accepted")
	}
	before := read(fan.Profile.ID).Points
	for i := 0; i < 2; i++ {
		if _, e := db.ArenaActivities(club.ID, 0, ArenaCommand{Action: "result", EventID: event, Score: "3-1", Confirm: true}, closed); e != nil {
			t.Fatal(e)
		}
	}
	if read(fan.Profile.ID).Points != before+50 {
		t.Fatal("prediction paid more than once")
	}
	invitation, e := db.ArenaActivities(club.ID, fan.Profile.ID, ArenaCommand{Action: "invite"}, now)
	if e != nil {
		t.Fatal(e)
	}
	accepted, e := db.ArenaActivities(club.ID, friend.Profile.ID, ArenaCommand{Action: "accept", Code: invitation.Invitation}, now)
	if e != nil {
		t.Fatal(e)
	}
	if accepted.Member.FanID != friend.Profile.ID {
		t.Fatal("invitation changed identity")
	}
	used := map[int]bool{}
	for _, m := range accepted.Members {
		if used[m.Slot] {
			t.Fatal("duplicate seat")
		}
		used[m.Slot] = true
	}
	rows, e := db.GetArenaFans(club.ID)
	if e != nil {
		t.Fatal(e)
	}
	for _, row := range rows {
		if row.Nickname != "" {
			t.Fatal("private nickname leaked")
		}
	}
	if _, e := db.ArenaActivities(club.ID, fan.Profile.ID, ArenaCommand{Action: "privacy", Public: true}, now); e != nil {
		t.Fatal(e)
	}
	rows, _ = db.GetArenaFans(club.ID)
	if rows[0].Nickname == "" {
		t.Fatal("public opt-in not saved")
	}
	for _, tr := range read(fan.Profile.ID).Trophies {
		if tr.Unlocked != "" {
			t.Fatalf("invented trophy %+v", tr)
		}
	}
	tomorrow := now.Add(24 * time.Hour)
	next, e := db.ArenaActivities(club.ID, fan.Profile.ID, ArenaCommand{}, tomorrow)
	if e != nil || next.Treasure.Claimed {
		t.Fatal("daily reset", e)
	}
}
