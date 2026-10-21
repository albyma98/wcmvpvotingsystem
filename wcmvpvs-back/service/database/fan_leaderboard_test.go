package database

import (
	"fmt"
	"testing"
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
