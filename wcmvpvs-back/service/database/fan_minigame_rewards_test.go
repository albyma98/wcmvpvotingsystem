package database

import (
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"testing"
)

func TestFanMinigameRewardsRepeatAndSurviveRegistration(t *testing.T) {
	db, club, _, event, _ := fanEconomyFixture(t)
	if _, _, err := db.ClaimFanJourneyReward(event, club.ID, 0, "player", "reaction"); err != nil {
		t.Fatal(err)
	}
	wallet := 10
	for _, key := range []string{"reaction", "tap", "sponsor-rush", "memory-flash"} {
		for round := 1; round <= 2; round++ {
			id := fmt.Sprintf("%s-round-%d", key, round)
			result, err := db.ClaimFanMinigameReward(event, club.ID, 0, "player", key, id)
			wallet += FanJourneyRewardAmount(key)
			if err != nil || result.Wallet != wallet || result.CoinsEarned != FanJourneyRewardAmount(key) {
				t.Fatalf("%s %d: %+v %v", key, round, result, err)
			}
			retry, err := (&appdbimpl{c: db.c}).ClaimFanMinigameReward(event, club.ID, 0, "player", key, id)
			if err != nil || retry.Wallet != wallet || retry.CoinsEarned != 0 || retry.RewardTotal != result.RewardTotal {
				t.Fatalf("retry: %+v %v", retry, err)
			}
		}
	}
	profile := fanEconomyRegister(t, db, club.ID, event, "player", "+393331234567", "session")
	if profile.Wallet != wallet {
		t.Fatalf("registration wallet %d want %d", profile.Wallet, wallet)
	}
	retry, err := db.ClaimFanMinigameReward(event, club.ID, profile.Profile.ID, "new-device", "reaction", "reaction-round-1")
	if err != nil || retry.Wallet != wallet || retry.CoinsEarned != 0 || retry.RewardTotal != 30 {
		t.Fatalf("portable receipt %+v %v", retry, err)
	}
	next, err := db.ClaimFanMinigameReward(event, club.ID, profile.Profile.ID, "new-device", "reaction", "reaction-round-3")
	if err != nil || next.Wallet != wallet+10 || next.RewardTotal != 40 {
		t.Fatalf("next %+v %v", next, err)
	}
	rewards, err := db.ListFanJourneyRewards(event, club.ID, profile.Profile.ID, "other-device")
	if err != nil || rewards["reaction"] != 40 || rewards["tap"] != 16 || rewards["memory-flash"] != 16 || rewards["sponsor-rush"] != 24 {
		t.Fatalf("cumulative progress %+v %v", rewards, err)
	}
	state, err := db.GetFanWalletState(event, club.ID, profile.Profile.ID, "other-device")
	if err != nil || state.Wallet != wallet+10 || state.Points != wallet+10 {
		t.Fatalf("wallet/points %+v %v", state, err)
	}
	if _, _, err := db.ClaimFanJourneyReward(event, club.ID, profile.Profile.ID, "new-device", "vote"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("vote still needs a real vote: %v", err)
	}
}

func TestFanMinigameQuizNeedsAnswersForEachRound(t *testing.T) {
	db, club, _, event, _ := fanEconomyFixture(t)
	if _, err := db.c.Exec(`INSERT INTO event_quiz(event_id,enabled,questions_per_session) VALUES(?,1,2)`, event); err != nil {
		t.Fatal(err)
	}
	ids := []int{}
	for index := 0; index < 2; index++ {
		res, err := db.c.Exec(`INSERT INTO event_quiz_questions(quiz_id,question_text,answers_json,correct_index,order_index) VALUES(?,'Question','["A","B"]',0,?)`, event, index)
		if err != nil {
			t.Fatal(err)
		}
		id, _ := res.LastInsertId()
		ids = append(ids, int(id))
	}
	for round := 1; round <= 2; round++ {
		id := fmt.Sprintf("quiz-round-%d", round)
		if _, err := db.ClaimFanMinigameReward(event, club.ID, 0, "player", "quiz", id); !errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("old answers cannot complete a new round: %v", err)
		}
		for _, question := range ids {
			if err := db.RecordFanMinigameQuizAnswer(event, club.ID, "player", id, question); err != nil {
				t.Fatal(err)
			}
		}
		result, err := db.ClaimFanMinigameReward(event, club.ID, 0, "player", "quiz", id)
		if err != nil || result.CoinsEarned != 15 || result.RewardTotal != round*15 || result.Wallet != round*15 {
			t.Fatalf("quiz %+v %v", result, err)
		}
	}
}

func TestFanMinigameConcurrentClaimsCreditOneRoundOnce(t *testing.T) {
	db, club, _, event, _ := fanEconomyFixture(t)
	var wg sync.WaitGroup
	for n := 0; n < 12; n++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := db.ClaimFanMinigameReward(event, club.ID, 0, "player", "tap", "same-round-123"); err != nil {
				t.Errorf("concurrent claim: %v", err)
			}
		}()
	}
	wg.Wait()
	var receipts, balance int
	if err := db.c.QueryRow(`SELECT COUNT(*) FROM fan_minigame_rewards WHERE event_id=?`, event).Scan(&receipts); err != nil {
		t.Fatal(err)
	}
	if err := db.c.QueryRow(`SELECT coins FROM guest_wallets WHERE event_id=? AND device_id='player'`, event).Scan(&balance); err != nil {
		t.Fatal(err)
	}
	if receipts != 1 || balance != 8 {
		t.Fatalf("receipts=%d balance=%d", receipts, balance)
	}
}

func TestFanMinigameOwnershipClosedEventAndRollback(t *testing.T) {
	db, club, other, event, _ := fanEconomyFixture(t)
	if _, err := db.ClaimFanMinigameReward(event, club.ID, 0, "player", "tap", "bad"); !errors.Is(err, ErrFanMinigameRound) {
		t.Fatal(err)
	}
	if _, err := db.ClaimFanMinigameReward(event, other.ID, 0, "player", "tap", "scoped-round"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatal(err)
	}
	if _, err := db.c.Exec(`CREATE TRIGGER fail_minigame_wallet BEFORE INSERT ON guest_wallets WHEN NEW.device_id='player' BEGIN SELECT RAISE(ABORT,'test wallet failure'); END`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ClaimFanMinigameReward(event, club.ID, 0, "player", "tap", "rollback-round"); err == nil {
		t.Fatal("wallet failure must roll back receipt")
	}
	var receipts int
	if err := db.c.QueryRow(`SELECT COUNT(*) FROM fan_minigame_rewards`).Scan(&receipts); err != nil || receipts != 0 {
		t.Fatalf("partial receipt %d %v", receipts, err)
	}
	if _, err := db.c.Exec(`DROP TRIGGER fail_minigame_wallet`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ClaimFanMinigameReward(event, club.ID, 0, "player", "tap", "rollback-round"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ClaimFanMinigameReward(event, club.ID, 0, "other-player", "tap", "rollback-round"); !errors.Is(err, ErrFanRewardDeviceUsed) {
		t.Fatal(err)
	}
	if _, err := db.ClaimFanMinigameReward(event, club.ID, 0, "player", "reaction", "rollback-round"); !errors.Is(err, ErrFanMinigameRound) {
		t.Fatal(err)
	}
	if _, err := db.c.Exec(`UPDATE events SET is_concluded=1 WHERE id=?`, event); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ClaimFanMinigameReward(event, club.ID, 0, "player", "tap", "another-round"); !errors.Is(err, ErrBasketballClosed) {
		t.Fatal(err)
	}
	if r, err := db.ClaimFanMinigameReward(event, club.ID, 0, "player", "tap", "rollback-round"); err != nil || r.CoinsEarned != 0 || r.Wallet != 8 {
		t.Fatalf("closed event receipt %+v %v", r, err)
	}
}
