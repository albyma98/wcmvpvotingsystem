package database

import (
	"database/sql"
	"testing"
)

func TestFanNearbyRanks(t *testing.T) {
	conn, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if _, err := conn.Exec(`CREATE TABLE fan_profiles(id INTEGER PRIMARY KEY, organization_id INTEGER, nickname TEXT);
		CREATE TABLE fan_wallets(fan_id INTEGER PRIMARY KEY, coins INTEGER);
		CREATE TABLE fan_economy_accounts(organization_id INTEGER,owner_key TEXT,points INTEGER);
		CREATE TABLE fan_shop_profiles(organization_id INTEGER,holder_key TEXT,equipped_frame TEXT,PRIMARY KEY(organization_id,holder_key));
		INSERT INTO fan_profiles VALUES (1,1,'A'),(2,1,'B'),(3,1,'C'),(4,1,'D'),(5,2,'Other');
		INSERT INTO fan_shop_profiles VALUES (1,'fan:3','club_frame'),(2,'fan:2','club_frame');
		INSERT INTO fan_wallets VALUES (1,50),(2,40),(3,40),(4,10),(5,1000)`); err != nil {
		t.Fatal(err)
	}
	db := &appdbimpl{c: conn}
	rows, err := db.GetFanNearbyRanks(1, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 || rows[0].FanID != 2 || rows[0].Rank != 2 || rows[1].FanID != 3 || rows[1].Rank != 3 || rows[2].FanID != 4 {
		t.Fatalf("neighborhood with tied coins: %+v", rows)
	}
	if rows[0].EquippedFrame != "" || rows[1].EquippedFrame != "club_frame" || rows[2].EquippedFrame != "" {
		t.Fatalf("frames must match the fan and organization: %+v", rows)
	}
	rows, err = db.GetFanNearbyRanks(1, 1)
	if err != nil || len(rows) != 2 || rows[0].Rank != 1 {
		t.Fatalf("leader: %+v, %v", rows, err)
	}
	rows, err = db.GetFanNearbyRanks(1, 5)
	if err != nil || len(rows) != 0 {
		t.Fatalf("cross-club fan: %+v, %v", rows, err)
	}
}
