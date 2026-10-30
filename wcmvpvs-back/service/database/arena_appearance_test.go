package database

import (
	"testing"
	"time"
)

func TestArenaAppearancePersistsWithoutChangingEconomyOrSeat(t *testing.T) {
	db, club, _, _, _ := fanEconomyFixture(t)
	fan := fanEconomyRegister(t, db, club.ID, 0, "visual", "+39007111", "visual-session")
	now := time.Now()
	before, err := db.ArenaActivities(club.ID, fan.Profile.ID, ArenaCommand{}, now)
	if err != nil {
		t.Fatal(err)
	}
	changed, err := db.ArenaActivities(club.ID, fan.Profile.ID, ArenaCommand{Action: "appearance", Appearance: map[string]int{"hair": 3, "skin": 1, "body": 2, "hairColor": 4, "shirt": 0, "expression": 1}}, now)
	if err != nil {
		t.Fatal(err)
	}
	after, err := db.ArenaActivities(club.ID, fan.Profile.ID, ArenaCommand{}, now)
	if err != nil {
		t.Fatal(err)
	}
	if changed.Equipment["look-hair"] != "3" || after.Equipment["look-hairColor"] != "4" || after.Wallet != before.Wallet || after.Points != before.Points || after.Member.Slot != before.Member.Slot {
		t.Fatal("appearance changed identity/economy or was not saved")
	}
	for _, invalid := range []map[string]int{{"hair": 99}, {"wallet": 100}, {"body": -1}} {
		if _, err := db.ArenaActivities(club.ID, fan.Profile.ID, ArenaCommand{Action: "appearance", Appearance: invalid}, now); err == nil {
			t.Fatal("invalid appearance accepted")
		}
	}
}
