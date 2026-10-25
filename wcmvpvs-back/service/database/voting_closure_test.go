package database

import (
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
)

func votingSettingsFixture() VotingSettings {
	return VotingSettings{TemplateSID: "HX" + strings.Repeat("a", 32), DrawMode: VotingDrawLeaderboard,
		Recipients: []VotingRecipient{{Name: "Organizzatore", Phone: "+393331234567", Enabled: true}, {Name: "Disabilitato", Phone: "+393331234568", Enabled: false}}}
}

func votingClosureFixture(t *testing.T) (*appdbimpl, Organization, Organization, int, []int) {
	t.Helper()
	db, club, other, event, _ := fanEconomyFixture(t)
	result, err := db.c.Exec(`INSERT INTO players(first_name,last_name,role,team_id,organization_id) VALUES('Mario','Rossi','player',?,?)`, club.TeamID, club.ID)
	if err != nil {
		t.Fatal(err)
	}
	player, _ := result.LastInsertId()
	var fans []int
	for i, points := range []int{50, 300, 200, 200} {
		device, phone := fmt.Sprintf("voter-%d", i), fmt.Sprintf("+39333000100%d", i)
		fan := fanEconomyRegister(t, db, club.ID, event, device, phone, "session-"+device)
		fans = append(fans, fan.Profile.ID)
		if _, err := db.c.Exec(`UPDATE fan_profiles SET phone_verified_at=CURRENT_TIMESTAMP WHERE id=?`, fan.Profile.ID); err != nil {
			t.Fatal(err)
		}
		if err := db.AddFanWalletCoins(fan.Profile.ID, points); err != nil {
			t.Fatal(err)
		}
		if err := db.AddVote(event, int(player), fmt.Sprintf("%04d", 100+i), "signature", device, &fan.Profile.ID); err != nil {
			t.Fatal(err)
		}
	}
	// The top club fan has no vote, so cannot win either mode.
	nonvoter := fanEconomyRegister(t, db, club.ID, event, "nonvoter", "+393330009999", "session-nonvoter")
	if err := db.AddFanWalletCoins(nonvoter.Profile.ID, 10000); err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= 3; i++ {
		if _, err := db.c.Exec(`INSERT INTO event_prizes(event_id,name,position) VALUES(?,?,?)`, event, fmt.Sprintf("Premio %d", i), i); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.SaveVotingSettings(club.ID, votingSettingsFixture()); err != nil {
		t.Fatal(err)
	}
	return db, club, other, event, fans
}

func TestVotingSettingsValidationAndOrganizationIsolation(t *testing.T) {
	db, club, other, _, _ := fanEconomyFixture(t)
	settings := votingSettingsFixture()
	settings.Recipients[0].Phone = "+39 (333) 123-4567"
	saved, err := db.SaveVotingSettings(club.ID, settings)
	if err != nil || saved.Recipients[0].Phone != "+393331234567" {
		t.Fatalf("save: %+v %v", saved, err)
	}
	got, err := db.GetVotingSettings(club.ID)
	if err != nil || !reflect.DeepEqual(got, saved) {
		t.Fatalf("roundtrip: %+v %v", got, err)
	}
	defaults, err := db.GetVotingSettings(other.ID)
	if err != nil || defaults.TemplateSID != "" || defaults.DrawMode != VotingDrawRandom || len(defaults.Recipients) != 0 {
		t.Fatalf("other club: %+v %v", defaults, err)
	}
	for _, change := range []func(*VotingSettings){
		func(s *VotingSettings) { s.TemplateSID = "AC" + strings.Repeat("b", 32) },
		func(s *VotingSettings) { s.TemplateSID = "" },
		func(s *VotingSettings) { s.DrawMode = "invalid" },
		func(s *VotingSettings) { s.Recipients[0].Phone = "3331234567" },
		func(s *VotingSettings) { s.Recipients[1].Phone = s.Recipients[0].Phone },
		func(s *VotingSettings) { s.Recipients[0].Phone = "+00000000" },
	} {
		invalid := votingSettingsFixture()
		change(&invalid)
		if _, err := db.SaveVotingSettings(club.ID, invalid); !errors.Is(err, ErrInvalidVotingSettings) {
			t.Fatalf("accepted invalid settings: %+v %v", invalid, err)
		}
	}
	unchanged, _ := db.GetVotingSettings(club.ID)
	if !reflect.DeepEqual(unchanged, saved) {
		t.Fatal("invalid save changed persistent settings")
	}
}

func TestVotingClosureLeaderboardUsesEligibleVotersAndRankingPoints(t *testing.T) {
	db, club, other, event, fans := votingClosureFixture(t)
	// Spending does not lower the existing leaderboard's cumulative points.
	if _, err := db.c.Exec(`UPDATE fan_wallets SET coins=0 WHERE fan_id=?`, fans[1]); err != nil {
		t.Fatal(err)
	}
	// A second ticket from the same top fan must not win another rank prize.
	if _, err := db.c.Exec(`INSERT INTO fan_sessions(token,fan_id,device_id) VALUES('extra-session',?,'extra-device')`, fans[1]); err != nil {
		t.Fatal(err)
	}
	var player int
	if err := db.c.QueryRow(`SELECT player_id FROM votes WHERE event_id=? LIMIT 1`, event).Scan(&player); err != nil {
		t.Fatal(err)
	}
	if err := db.AddVote(event, player, "0900", "signature", "extra-device", &fans[1]); err != nil {
		t.Fatal(err)
	}
	if _, err := db.CloseEventVotingAndDraw(event, other.ID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("cross organization close: %v", err)
	}
	summary, err := db.CloseEventVotingAndDraw(event, club.ID)
	if err != nil {
		t.Fatal(err)
	}
	if summary.MVPName != "Mario Rossi" || summary.Percentage != "100,00" || summary.TotalVotes != 5 || summary.MVPVotes != 5 {
		t.Fatalf("MVP: %+v", summary)
	}
	for i, fanID := range []int{fans[1], fans[2], fans[3]} {
		if summary.Prizes[i].FanID != fanID || summary.Prizes[i].TicketCode != fmt.Sprintf("%04d", 101+i) {
			t.Fatalf("ranking winners: %+v", summary.Prizes)
		}
	}
	_, notifications, err := db.GetVotingClosure(event, club.ID)
	if err != nil || len(notifications) != 1 || notifications[0].Phone != "+393331234567" {
		t.Fatalf("recipients: %+v %v", notifications, err)
	}
	variables := summary.TemplateVariables()
	if len(variables) != 8 || variables["1"] != "Mario Rossi" || variables["2"] != "100,00" || variables["3"] != "Premio 1" || variables["4"] != "0101" || variables["8"] != "0103" {
		t.Fatalf("variables: %+v", variables)
	}
	prizes, err := db.ListEventPrizes(event)
	if err != nil || prizes[0].Winner == nil || prizes[0].Winner.UserID != fans[1] || prizes[0].Winner.TicketCode != "0101" {
		t.Fatalf("lottery did not see winners: %+v %v", prizes, err)
	}
	if err := db.AddVote(event, player, "0500", "signature", "late", nil); !errors.Is(err, ErrEventVotingClosed) {
		t.Fatalf("late vote admitted: %v", err)
	}
	if err := db.AddVote(event, player, "0100", "signature", "voter-0", nil); !errors.Is(err, ErrEventVotingClosed) {
		t.Fatalf("late vote edit admitted: %v", err)
	}
	if err := db.ClearPrizeWinner(event, prizes[0].ID); !errors.Is(err, ErrVotingClosureLocked) {
		t.Fatalf("final winner changed: %v", err)
	}
	if _, err := db.AssignPrizeWinner(event, prizes[0].ID, prizes[1].Winner.VoteID); !errors.Is(err, ErrVotingClosureLocked) {
		t.Fatalf("final winner replaced: %v", err)
	}
	if _, _, err := db.GetVotingClosure(event, other.ID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("summary exposed to other club: %v", err)
	}
}

func TestVotingClosureRandomMatchesLotteryEligibilityAndIsIdempotent(t *testing.T) {
	db, club, other, event, _ := votingClosureFixture(t)
	settings := votingSettingsFixture()
	settings.DrawMode = VotingDrawRandom
	if _, err := db.SaveVotingSettings(club.ID, settings); err != nil {
		t.Fatal(err)
	}
	var player int
	_ = db.c.QueryRow(`SELECT player_id FROM votes WHERE event_id=? LIMIT 1`, event).Scan(&player)
	// Guests and unverified/other-society profiles must never enter the draw.
	_ = db.AddVote(event, player, "0800", "signature", "guest", nil)
	fan := fanEconomyRegister(t, db, club.ID, event, "unverified", "+393330007777", "unverified-session")
	_ = db.AddVote(event, player, "0801", "signature", "unverified", &fan.Profile.ID)
	outsider := fanEconomyRegister(t, db, other.ID, 0, "outsider", "+393330006666", "outsider-session")
	_, _ = db.c.Exec(`UPDATE fan_profiles SET phone_verified_at=CURRENT_TIMESTAMP WHERE id=?`, outsider.Profile.ID)
	_ = db.AddVote(event, player, "0802", "signature", "outsider", &outsider.Profile.ID)
	// An already assigned prize stays assigned. Remaining tickets are sampled.
	tickets, err := db.ListEventTickets(event)
	if err != nil {
		t.Fatal(err)
	}
	prizes, _ := db.ListEventPrizes(event)
	if _, err := db.AssignPrizeWinner(event, prizes[0].ID, tickets[0].VoteID); err != nil {
		t.Fatal(err)
	}
	summary, err := db.CloseEventVotingAndDraw(event, club.ID)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, p := range summary.Prizes {
		if p.TicketCode < "0100" || p.TicketCode > "0103" || seen[p.TicketCode] {
			t.Fatalf("ineligible/duplicate winner: %+v", summary.Prizes)
		}
		seen[p.TicketCode] = true
	}
	if summary.Prizes[0].VoteID != tickets[0].VoteID {
		t.Fatal("existing winner overwritten")
	}
	var wg sync.WaitGroup
	errorsCh := make(chan error, 5)
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			got, err := db.CloseEventVotingAndDraw(event, club.ID)
			if err == nil && !reflect.DeepEqual(got, summary) {
				err = fmt.Errorf("closing changed winners")
			}
			errorsCh <- err
		}()
	}
	wg.Wait()
	close(errorsCh)
	for err := range errorsCh {
		if err != nil {
			t.Fatal(err)
		}
	}
	if err := db.SetActiveEvent(event, club.ID); err != nil {
		t.Fatal(err)
	}
	again, err := db.CloseEventVotingAndDraw(event, club.ID)
	if err != nil || !reflect.DeepEqual(again, summary) {
		t.Fatalf("reopening redrew: %+v %v", again, err)
	}
	var jobs int
	if err := db.c.QueryRow(`SELECT COUNT(*) FROM voting_notifications WHERE event_id=?`, event).Scan(&jobs); err != nil || jobs != 3 {
		t.Fatalf("duplicate jobs: %d %v", jobs, err)
	} // 2 new winners + organizer
	if summary.Percentage != "100,00" || summary.TotalVotes != 7 {
		t.Fatalf("MVP denominator excluded guest votes: %+v", summary)
	}
}

func TestVotingClosureEmptyAndMissingPrizesFillTemplate(t *testing.T) {
	db, club, _, event, _ := fanEconomyFixture(t)
	_, _ = db.c.Exec(`INSERT INTO event_prizes(event_id,name,position) VALUES(?,'Maglia',1)`, event)
	summary, err := db.CloseEventVotingAndDraw(event, club.ID)
	if err != nil {
		t.Fatal(err)
	}
	v := summary.TemplateVariables()
	if v["1"] != "Nessun voto" || v["2"] != "0,00" || v["3"] != "Maglia" || v["4"] != "Non assegnato" || v["5"] != "Premio non previsto" || v["6"] != "-" || len(summary.Warnings) != 2 {
		t.Fatalf("empty result: %+v %+v", summary, v)
	}
}

func TestVotingNotificationQueueClaimsRetryAndRestart(t *testing.T) {
	db, club, other, event, _ := votingClosureFixture(t)
	if _, err := db.CloseEventVotingAndDraw(event, club.ID); err != nil {
		t.Fatal(err)
	}
	var organizer VotingNotification
	ids := map[int]bool{}
	for i := 0; i < 4; i++ {
		n, err := db.ClaimVotingNotification()
		if err != nil || ids[n.ID] {
			t.Fatalf("claim: %+v %v", n, err)
		}
		ids[n.ID] = true
		if n.Kind == "organizer" {
			organizer = n
		}
		if err := db.FinishVotingNotification(n.ID, "failed", "", "", "Rejected"); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.ClaimVotingNotification(); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("queue not empty: %v", err)
	}
	if err := db.RetryVotingNotifications(event, other.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ClaimVotingNotification(); !errors.Is(err, sql.ErrNoRows) {
		t.Fatal("another club retried queue")
	}
	settings := votingSettingsFixture()
	settings.TemplateSID = "HX" + strings.Repeat("b", 32)
	_, _ = db.SaveVotingSettings(club.ID, settings)
	if err := db.RetryVotingNotifications(event, club.ID); err != nil {
		t.Fatal(err)
	}
	// A new repository instance reads the persisted pending notification.
	restarted := &appdbimpl{c: db.c}
	retried, err := restarted.ClaimVotingNotification()
	if err != nil || retried.ID != organizer.ID || retried.TemplateSID != settings.TemplateSID || retried.Variables["4"] != "0101" {
		t.Fatalf("retry lost snapshot/config: %+v %v", retried, err)
	}
	_, _ = db.c.Exec(`UPDATE voting_notifications SET updated_at=datetime('now','-6 minutes') WHERE id=?`, retried.ID)
	if _, err := restarted.ClaimVotingNotification(); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("ambiguous resend: %v", err)
	}
	var status string
	_ = db.c.QueryRow(`SELECT status FROM voting_notifications WHERE id=?`, retried.ID).Scan(&status)
	if status != "unknown" {
		t.Fatalf("interrupted claim never recovered: %s", status)
	}
	_ = db.RetryVotingNotifications(event, club.ID)
	if _, err := db.ClaimVotingNotification(); !errors.Is(err, sql.ErrNoRows) {
		t.Fatal("blind retry of ambiguous send")
	}
}

func TestVotingClosureMVPPercentageAndEventDeletion(t *testing.T) {
	db, club, _, event, _ := votingClosureFixture(t)
	result, err := db.c.Exec(`INSERT INTO players(first_name,last_name,role,team_id,organization_id) VALUES('Luca','Bianchi','player',?,?)`, club.TeamID, club.ID)
	if err != nil {
		t.Fatal(err)
	}
	player, _ := result.LastInsertId()
	if err := db.AddVote(event, int(player), "0800", "signature", "guest", nil); err != nil {
		t.Fatal(err)
	}
	summary, err := db.CloseEventVotingAndDraw(event, club.ID)
	if err != nil || summary.Percentage != "80,00" {
		t.Fatalf("percentage: %+v %v", summary, err)
	}
	if err := db.PurgeEventData(event); err != nil {
		t.Fatal(err)
	}
	if _, _, err := db.GetVotingClosure(event, club.ID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("snapshot retained after delete: %v", err)
	}
	if _, err := db.ClaimVotingNotification(); !errors.Is(err, sql.ErrNoRows) {
		t.Fatal("deleted event still sends notifications")
	}
}
