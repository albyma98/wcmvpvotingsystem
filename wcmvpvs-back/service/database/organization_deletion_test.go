package database

import (
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestDeleteOrganizationDataIsAtomicAndKeepsOtherOrganizations(t *testing.T) {
	conn, err := sql.Open("sqlite3", "file:"+filepath.Join(t.TempDir(), "organizations.db")+"?_foreign_keys=on")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	conn.SetMaxOpenConns(1)
	appdb, err := New(conn)
	if err != nil {
		t.Fatal(err)
	}
	first, err := appdb.CreateOrganization(Organization{Name: "Prima società", Slug: "prima", IsActive: true})
	if err != nil {
		t.Fatal(err)
	}
	second, err := appdb.CreateOrganization(Organization{Name: "Seconda società", Slug: "seconda", IsActive: true})
	if err != nil {
		t.Fatal(err)
	}
	exec := func(query string, args ...interface{}) int64 {
		t.Helper()
		result, err := conn.Exec(query, args...)
		if err != nil {
			t.Fatalf("%s: %v", query, err)
		}
		id, _ := result.LastInsertId()
		return id
	}
	count := func(query string, args ...interface{}) int {
		t.Helper()
		var n int
		if err := conn.QueryRow(query, args...).Scan(&n); err != nil {
			t.Fatalf("%s: %v", query, err)
		}
		return n
	}
	firstEvent := exec(`INSERT INTO events (organization_id, team1_id, team2_id, start_datetime) VALUES (?, ?, ?, '2026-10-21')`, first.ID, first.TeamID, second.TeamID)
	secondEvent := exec(`INSERT INTO events (organization_id, team1_id, team2_id, start_datetime) VALUES (?, ?, ?, '2026-10-24')`, second.ID, second.TeamID, first.TeamID)
	firstPlayer := exec(`INSERT INTO players (first_name, last_name, role, team_id, organization_id) VALUES ('A', 'Uno', 'player', ?, ?)`, first.TeamID, first.ID)
	secondPlayer := exec(`INSERT INTO players (first_name, last_name, role, team_id, organization_id) VALUES ('B', 'Due', 'player', ?, ?)`, second.TeamID, second.ID)
	exec(`INSERT INTO votes (event_id, player_id, ticket_code, ticket_signature, device_id) VALUES (?, ?, '111111', 'sig', 'device-1')`, firstEvent, firstPlayer)
	exec(`INSERT INTO votes (event_id, player_id, ticket_code, ticket_signature, device_id) VALUES (?, ?, '222222', 'sig', 'device-2')`, secondEvent, secondPlayer)
	exec(`INSERT INTO event_prizes (event_id, name) VALUES (?, 'Premio')`, firstEvent)
	firstFan := exec(`INSERT INTO fan_profiles (organization_id, nickname, phone) VALUES (?, 'fan-a', '+391111')`, first.ID)
	exec(`INSERT INTO fan_profiles (organization_id, nickname, phone) VALUES (?, 'fan-b', '+392222')`, second.ID)
	exec(`INSERT INTO fan_sessions (token, fan_id) VALUES ('session-a', ?)`, firstFan)
	exec(`INSERT INTO fan_wallets (fan_id, coins) VALUES (?, 10)`, firstFan)
	sponsor := exec(`INSERT INTO sponsors (organization_id, position, name, logo_data) VALUES (?, 1, 'Sponsor A', '')`, first.ID)
	coupon := exec(`INSERT INTO coupons (title, sponsor_id, organization_id) VALUES ('Coupon A', ?, ?)`, sponsor, first.ID)
	exec(`INSERT INTO user_coupons (coupon_id, user_id, match_id, code) VALUES (?, ?, ?, 'coupon-a')`, coupon, firstFan, firstEvent)
	exec(`CREATE TABLE club_sites (id INTEGER PRIMARY KEY, organization_id INTEGER NOT NULL)`)
	exec(`CREATE TABLE club_teams (id INTEGER PRIMARY KEY, site_id INTEGER NOT NULL)`)
	site := exec(`INSERT INTO club_sites (organization_id) VALUES (?)`, first.ID)
	exec(`INSERT INTO club_teams (site_id) VALUES (?)`, site)
	exec(`CREATE TABLE extra_event_data (id INTEGER PRIMARY KEY, event_id INTEGER NOT NULL)`)
	exec(`INSERT INTO extra_event_data (event_id) VALUES (?)`, firstEvent)
	exec(`CREATE TABLE extra_organization_data (id INTEGER PRIMARY KEY, organization_id INTEGER NOT NULL)`)
	exec(`INSERT INTO extra_organization_data (organization_id) VALUES (?)`, first.ID)
	exec(`INSERT INTO qr_redirects (source_path, target_path) VALUES ('/qr/prima', '/prima')`)
	exec(`INSERT INTO qr_redirects (source_path, target_path) VALUES ('/qr/prima/admin', '/prima/admin')`)
	exec(`INSERT INTO qr_redirects (source_path, target_path) VALUES ('/qr/primavera', '/primavera')`)
	exec(`INSERT INTO qr_redirects (source_path, target_path) VALUES ('/qr/seconda', '/seconda')`)

	// A vote in the second society referring to the first one's player must
	// block the whole operation rather than delete another society's vote.
	crossVote := exec(`INSERT INTO votes (event_id, player_id, ticket_code, ticket_signature, device_id) VALUES (?, ?, '333333', 'sig', 'device-3')`, secondEvent, firstPlayer)
	if _, err := appdb.DeleteOrganizationData(first.ID); !errors.Is(err, ErrOrganizationSharedData) {
		t.Fatalf("shared player: got %v, want ErrOrganizationSharedData", err)
	}
	if count(`SELECT COUNT(*) FROM organizations WHERE id = ?`, first.ID) != 1 || count(`SELECT COUNT(*) FROM events WHERE id = ?`, firstEvent) != 1 {
		t.Fatal("failed deletion changed organization data")
	}
	exec(`DELETE FROM votes WHERE id = ?`, crossVote)

	eventIDs, err := appdb.DeleteOrganizationData(first.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(eventIDs) != 1 || eventIDs[0] != int(firstEvent) {
		t.Fatalf("deleted event IDs: %v", eventIDs)
	}
	for _, item := range []struct {
		table, where string
		id           int64
	}{
		{"organizations", "id", int64(first.ID)},
		{"events", "id", firstEvent},
		{"players", "id", firstPlayer},
		{"fan_profiles", "id", firstFan},
		{"fan_sessions", "fan_id", firstFan},
		{"fan_wallets", "fan_id", firstFan},
		{"sponsors", "id", sponsor},
		{"coupons", "id", coupon},
		{"user_coupons", "coupon_id", coupon},
		{"club_sites", "id", site},
		{"club_teams", "site_id", site},
		{"extra_event_data", "event_id", firstEvent},
		{"extra_organization_data", "organization_id", int64(first.ID)},
	} {
		if count(`SELECT COUNT(*) FROM `+item.table+` WHERE `+item.where+` = ?`, item.id) != 0 {
			t.Fatalf("%s still contains deleted society data", item.table)
		}
	}
	if count(`SELECT COUNT(*) FROM organizations WHERE id = ?`, second.ID) != 1 ||
		count(`SELECT COUNT(*) FROM events WHERE id = ?`, secondEvent) != 1 ||
		count(`SELECT COUNT(*) FROM votes WHERE event_id = ?`, secondEvent) != 1 {
		t.Fatal("second society's data was changed")
	}
	if count(`SELECT COUNT(*) FROM teams WHERE id = ?`, first.TeamID) != 1 {
		t.Fatal("team referenced by another society's event was deleted")
	}
	if count(`SELECT COUNT(*) FROM qr_redirects WHERE target_path LIKE '/prima%'`) != 1 ||
		count(`SELECT COUNT(*) FROM qr_redirects WHERE target_path = '/seconda'`) != 1 {
		t.Fatal("QR redirects were not isolated by society route")
	}
	if count(`SELECT COUNT(*) FROM pragma_foreign_key_check`) != 0 {
		t.Fatal("foreign key violation after deleting society")
	}
}
