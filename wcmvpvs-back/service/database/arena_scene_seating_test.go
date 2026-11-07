package database

import "testing"

func TestConfiguredArenaInvitationSeating(t *testing.T) {
	if seats := arenaSceneSeats(balturArenaScene); len(seats) != arenaVirtualSeatCount || seats[0].side != "long_1" || seats[0].z != 9.65 {
		t.Fatal("Baltur scene should map every virtual slot to a configured seat")
	}
	if seats := arenaSceneSeats(`{"renderer":"joy-volley"}`); len(seats) != 0 {
		t.Fatal("legacy scene should use legacy seating")
	}
	seats := []arenaSceneSeat{
		{side: "long_1", x: 4, y: 1, z: 10},
		{side: "long_1", x: 3.5, y: 1, z: 10},
		{side: "short_1", x: 4, y: 1, z: 10.1},
		{side: "long_1", x: 4.5, y: 1, z: 10.4},
		{side: "long_1", x: 4.1, y: 4, z: 10},
	}
	used := map[int]bool{0: true, 1: true}
	if got := nearestConfiguredArenaSlot(seats, 0, used); got != 3 {
		t.Fatalf("nearest free seat on the same side = %d, want 3", got)
	}
	used[3], used[4] = true, true
	if got := nearestConfiguredArenaSlot(seats, 0, used); got != 2 {
		t.Fatalf("nearest free seat when the side is full = %d, want 2", got)
	}
}
