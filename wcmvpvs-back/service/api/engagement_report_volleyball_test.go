package api

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
)

func TestEngagementReportVolleyballRoundsExcludePreparedGamesAndPreserveWalletEvidence(t *testing.T) {
	f := newEngagementReportFixture(t)
	result, err := f.conn.Exec(`INSERT INTO fan_profiles(organization_id,nickname,phone,registration_pending) VALUES(?,'Volley fan','+39444',0)`, f.org.ID)
	if err != nil {
		t.Fatal(err)
	}
	fanID, _ := result.LastInsertId()
	otherEvent, err := f.router.db.CreateEvent(database.Event{OrganizationID: f.other.ID, Team1ID: f.other.TeamID, Team2ID: f.org.TeamID, StartDateTime: "2026-10-22T19:00:00Z"})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UnixMilli()
	insertRound := func(id, game, status string, attempt, event, org int) {
		t.Helper()
		_, err := f.conn.Exec(`INSERT INTO volleyball_rounds(round_id,event_id,organization_id,device_id,fan_id,game_id,attempt,target_x,target_y,blocker_x,blocker_y,timing_target,issued_at,expires_at,status,perfect) VALUES(?,?,?,'volley-browser',?,?,?,0.5,0.5,0.2,0.2,0.5,?,?,?,0)`, id, event, org, fanID, game, attempt, now, now+120000, status)
		if err != nil {
			t.Fatal(err)
		}
	}
	for _, game := range []string{"volley-rush", "setter-pro", "monster-block"} {
		for attempt, status := range []string{"pending", "success", "miss", "expired"} {
			insertRound(game+":"+status, game, status, attempt, f.event, f.org.ID)
		}
		insertRound(game+":foreign", game, "success", 0, otherEvent, f.other.ID)
		if _, err := f.conn.Exec(`INSERT INTO fan_wallet_movements(organization_id,owner_key,event_id,fan_id,device_id,delta,balance,label,source) VALUES(?,'fan:volley',?,?,'volley-browser',1,1,'Punto pallavolo',?)`, f.org.ID, f.event, fanID, game); err != nil {
			t.Fatal(err)
		}
	}
	data, err := f.router.loadReportOperationalData(f.event, f.org.ID)
	if err != nil {
		t.Fatal(err)
	}
	if data.Summary.CoinsEarned != 3 || data.Summary.IdentifiedParticipants != 1 {
		t.Fatalf("volleyball economy missing from report: %+v", data.Summary)
	}
	completed := map[string]int{}
	for _, signal := range data.Signals {
		if signal.Metadata["_table"] != "volleyball_rounds" {
			continue
		}
		game, _ := signal.Metadata["game_id"].(string)
		round, _ := signal.Metadata["round_id"].(string)
		if signal.Name != "game_completed" || (!strings.HasSuffix(round, ":success") && !strings.HasSuffix(round, ":miss")) || signal.FanID != int(fanID) || signal.DeviceID != reportOperationalDeviceKey(f.org.ID, "volley-browser") || signal.OccurredAt != reportOperationalEpoch(now) {
			t.Fatalf("incorrect resolved volleyball evidence: %+v", signal)
		}
		completed[game]++
	}
	for _, game := range []string{"volley-rush", "setter-pro", "monster-block"} {
		if completed[game] != 2 {
			t.Fatalf("%s completed rounds=%d, want 2", game, completed[game])
		}
	}
	facts, details, _ := calculateReportMetrics(data.Signals, nil)
	if facts["rounds_completed"] == nil || *facts["rounds_completed"] != 6 || len(details.Games) != 3 {
		t.Fatalf("volleyball rounds not counted correctly: facts=%+v games=%+v", facts, details.Games)
	}
}

func TestEngagementReportVolleyballTelemetryMatchesServerRoundOnce(t *testing.T) {
	f := newEngagementReportFixture(t)
	now := time.Now().UnixMilli()
	for _, game := range []string{"volley-rush", "setter-pro", "monster-block"} {
		round := game + ":resolved"
		if _, err := f.conn.Exec(`INSERT INTO volleyball_rounds(round_id,event_id,organization_id,device_id,game_id,attempt,target_x,target_y,blocker_x,blocker_y,timing_target,issued_at,expires_at,status) VALUES(?,?,?,'volley-browser',?,0,0.5,0.5,0.2,0.2,0.5,?,?,'success')`, round, f.event, f.org.ID, game, now, now+120000); err != nil {
			t.Fatal(err)
		}
		events := []interface{}{}
		for _, name := range []string{"game_started", "game_completed"} {
			events = append(events, engagementReportSignalPayload(game+":"+name, name, map[string]interface{}{"game_id": game, "round_id": round}))
		}
		response := engagementReportRequest(t, f.router.recordReportSignals, http.MethodPost, f.event, map[string]interface{}{"events": events}, f.ctx, map[string]string{"X-Device-ID": "volley-browser"})
		engagementReportRequireStatus(t, response, http.StatusNoContent)
	}
	data, err := f.router.loadReportOperationalData(f.event, f.org.ID)
	if err != nil {
		t.Fatal(err)
	}
	telemetry, err := f.router.loadReportSignals(f.event)
	if err != nil || len(telemetry) != 6 {
		t.Fatalf("volleyball telemetry not retained: count=%d err=%v", len(telemetry), err)
	}
	facts, details, _ := calculateReportMetrics(append(data.Signals, telemetry...), nil)
	if facts["rounds_completed"] == nil || *facts["rounds_completed"] != 3 || facts["players"] == nil || *facts["players"] != 1 || len(details.Games) != 3 {
		t.Fatalf("volleyball telemetry double counted: facts=%+v games=%+v", facts, details.Games)
	}
	for _, game := range details.Games {
		if game.RoundsStarted != 1 || game.RoundsCompleted != 1 {
			t.Fatalf("duplicate volleyball round details: %+v", game)
		}
	}
}
