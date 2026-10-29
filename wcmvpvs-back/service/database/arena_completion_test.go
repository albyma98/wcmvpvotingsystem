package database

import (
	"testing"
	"time"
)

func TestArenaCollectiveRequiresCompletedAnimations(t *testing.T) {
	db, club, _, _, _ := fanEconomyFixture(t)
	now := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	ids := []int{}
	for i, phone := range []string{"+39004101", "+39004102", "+39004103"} {
		fan := fanEconomyRegister(t, db, club.ID, 0, phone, phone, phone)
		ids = append(ids, fan.Profile.ID)
		if _, err := db.ArenaActivities(club.ID, fan.Profile.ID, ArenaCommand{Action: "wave"}, now.Add(time.Duration(i)*time.Second)); err != nil {
			t.Fatal(err)
		}
	}
	early, err := db.ArenaActivities(club.ID, ids[0], ArenaCommand{Action: "wave-complete"}, now.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	for _, tr := range early.Trophies {
		if tr.Key == "ola" && tr.Unlocked != "" {
			t.Fatal("rewarded an incomplete choreography")
		}
	}
	var completed ArenaActivityState
	for _, id := range ids {
		completed, err = db.ArenaActivities(club.ID, id, ArenaCommand{Action: "wave-complete"}, now.Add(11*time.Second))
		if err != nil {
			t.Fatal(err)
		}
	}
	found := false
	for _, tr := range completed.Trophies {
		if tr.Key == "ola" && tr.Unlocked != "" {
			found = true
		}
	}
	if !found {
		t.Fatal("completed collective choreography did not unlock trophy")
	}
}
