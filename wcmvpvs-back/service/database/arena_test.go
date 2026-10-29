package database

import (
	"testing"
)

func TestArenaOnlyCompletedProfilesOfOrganization(t *testing.T) {
	db, club, other, event, _ := fanEconomyFixture(t)
	own := fanEconomyRegister(t, db, club.ID, event, "Own", "+390011", "arena-own")
	fanEconomyRegister(t, db, other.ID, 0, "Other", "+390012", "arena-other")
	rows, err := db.GetArenaFans(club.ID)
	if err != nil || len(rows) != 1 || rows[0].ID != own.Profile.ID {
		t.Fatalf("tenant filter: %+v %v", rows, err)
	}
	if _, err := db.c.Exec(`UPDATE fan_profiles SET registration_pending=1 WHERE id=?`, own.Profile.ID); err != nil {
		t.Fatal(err)
	}
	pending, err := db.GetArenaFans(club.ID)
	if err != nil || len(pending) != 0 {
		t.Fatalf("pending profile exposed: %+v %v", pending, err)
	}
	empty, err := db.GetArenaFans(0)
	if err != nil || len(empty) != 0 {
		t.Fatalf("empty scope: %+v %v", empty, err)
	}
}
