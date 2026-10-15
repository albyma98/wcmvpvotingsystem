package database

import (
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
)

func TestFanJourneyRewards(t *testing.T) {
	conn, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "journey.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if _, err := conn.Exec(`CREATE TABLE events(id INTEGER PRIMARY KEY,organization_id INTEGER);
		CREATE TABLE votes(id INTEGER PRIMARY KEY,event_id INTEGER,device_id TEXT,ticket_code TEXT);
		CREATE TABLE event_quiz(event_id INTEGER,questions_per_session INTEGER);
		CREATE TABLE event_quiz_questions(id INTEGER,quiz_id INTEGER,order_index INTEGER);
		INSERT INTO events VALUES(42,7);
		INSERT INTO votes(event_id,device_id,ticket_code) VALUES(42,'device','ticket');
		INSERT INTO event_quiz VALUES(42,2);
		INSERT INTO event_quiz_questions VALUES(1,42,0),(2,42,1);`); err != nil {
		t.Fatal(err)
	}
	if err := ensureFanProfileTables(conn); err != nil {
		t.Fatal(err)
	}
	db := &appdbimpl{c: conn}
	wallet, credited, err := db.ClaimFanJourneyReward(42, 7, 0, "device", "vote")
	if err != nil || wallet != 10 || credited != 10 {
		t.Fatalf("vote %d/%d: %v", wallet, credited, err)
	}
	// Simulate a response that was lost: a fresh database wrapper must see the ledger.
	reloaded := &appdbimpl{c: conn}
	wallet, credited, err = reloaded.ClaimFanJourneyReward(42, 7, 0, "device", "vote")
	if err != nil || wallet != 10 || credited != 0 {
		t.Fatalf("retry %d/%d: %v", wallet, credited, err)
	}
	if _, _, err := db.ClaimFanJourneyReward(42, 7, 0, "device", "quiz"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("incomplete quiz %v", err)
	}
	for _, id := range []int{1, 2} {
		if err := db.RecordFanJourneyQuizAnswer(42, 7, "device", id); err != nil {
			t.Fatal(err)
		}
	}
	wallet, credited, err = db.ClaimFanJourneyReward(42, 7, 0, "device", "quiz")
	if err != nil || wallet != 25 || credited != 15 {
		t.Fatalf("quiz %d/%d: %v", wallet, credited, err)
	}
	if _, _, err := db.ClaimFanJourneyReward(42, 8, 0, "device", "vote"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("cross-club claim %v", err)
	}
	profile, err := db.RegisterFan(FanRegisterInput{
		OrganizationID: 7, EventID: 42, DeviceID: "device", SessionToken: "session",
		Nickname: "Fan", Phone: "123", AcceptedTerms: true, GuestCoins: 25,
	})
	if err != nil || profile.Wallet != 25 {
		t.Fatalf("guest registration %+v: %v", profile, err)
	}
	wallet, credited, err = db.ClaimFanJourneyReward(42, 7, profile.Profile.ID, "other-device", "vote")
	if err != nil || wallet != 25 || credited != 0 {
		t.Fatalf("registered retry %d/%d: %v", wallet, credited, err)
	}
	rewards, err := db.ListFanJourneyRewards(42, 7, profile.Profile.ID, "other-device")
	if err != nil || rewards["vote"] != 10 || rewards["quiz"] != 15 {
		t.Fatalf("portable badge %+v: %v", rewards, err)
	}
	other, err := db.ListFanJourneyRewards(43, 7, profile.Profile.ID, "other-device")
	if err != nil || len(other) != 0 {
		t.Fatalf("different match %+v: %v", other, err)
	}
	if _, err := conn.Exec(`INSERT INTO events VALUES(43,7); INSERT INTO votes(event_id,device_id) VALUES(43,'other-device');`); err != nil {
		t.Fatal(err)
	}
	wallet, credited, err = db.ClaimFanJourneyReward(43, 7, profile.Profile.ID, "other-device", "vote")
	if err != nil || wallet != 35 || credited != 10 {
		t.Fatalf("registered fan next match %d/%d: %v", wallet, credited, err)
	}
}
