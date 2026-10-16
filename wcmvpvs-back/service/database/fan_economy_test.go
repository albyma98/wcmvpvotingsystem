package database

import (
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func fanEconomyFixture(t *testing.T) (*appdbimpl, Organization, Organization, int, int) {
	t.Helper()
	conn, err := sql.Open("sqlite3", "file:"+filepath.Join(t.TempDir(), "economy.db")+"?_foreign_keys=on")
	if err != nil {
		t.Fatal(err)
	}
	conn.SetMaxOpenConns(1)
	t.Cleanup(func() { conn.Close() })
	app, err := New(conn)
	if err != nil {
		t.Fatal(err)
	}
	db := app.(*appdbimpl)
	club, err := db.CreateOrganization(Organization{Name: "Club Basket", Slug: "economy-club", IsActive: true})
	if err != nil {
		t.Fatal(err)
	}
	other, err := db.CreateOrganization(Organization{Name: "Avversari", Slug: "economy-other", IsActive: true})
	if err != nil {
		t.Fatal(err)
	}
	insert := func(date string, active int) int {
		res, err := conn.Exec(`INSERT INTO events(organization_id,team1_id,team2_id,start_datetime,is_active) VALUES(?,?,?,?,?)`, club.ID, club.TeamID, other.TeamID, date, active)
		if err != nil {
			t.Fatal(err)
		}
		id, err := res.LastInsertId()
		if err != nil {
			t.Fatal(err)
		}
		return int(id)
	}
	return db, club, other, insert("2026-10-16T18:00:00Z", 1), insert("2026-10-23T18:00:00Z", 0)
}

func fanEconomyRegister(t *testing.T, db *appdbimpl, org, event int, device, phone, session string) FanProfileSummary {
	t.Helper()
	profile, err := db.RegisterFan(FanRegisterInput{OrganizationID: org, EventID: event, DeviceID: device, Phone: phone, Nickname: "Fan " + phone, SessionToken: session, AcceptedTerms: true, GuestCoins: 999999})
	if err != nil {
		t.Fatal(err)
	}
	return profile
}

func fanEconomyVote(t *testing.T, db *appdbimpl, club Organization, event int, device string) {
	t.Helper()
	result, err := db.c.Exec(`INSERT INTO players(first_name,last_name,role,team_id,organization_id) VALUES('Giocatore','Uno','player',?,?)`, club.TeamID, club.ID)
	if err != nil {
		t.Fatal(err)
	}
	player, err := result.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.c.Exec(`INSERT INTO votes(event_id,player_id,device_id,ticket_code,ticket_signature) VALUES(?,?,?,?,'signature')`, event, player, device, "ticket-"+device); err != nil {
		t.Fatal(err)
	}
}

func TestFanEconomyCatalogPriceIdempotencyAndScope(t *testing.T) {
	db, club, other, event, nextEvent := fanEconomyFixture(t)
	if err := db.UpsertGuestCoins(event, club.ID, "guest", 100); err != nil {
		t.Fatal(err)
	}
	if err := db.PurchaseFanShopItem(event, club.ID, 0, "guest", "club_frame"); err != nil {
		t.Fatal(err)
	}
	if err := db.PurchaseFanShopItem(event, club.ID, 0, "guest", "club_frame"); err != nil {
		t.Fatal(err)
	}
	if err := db.PurchaseFanShopItem(event, club.ID, 0, "guest", "match_card"); err != nil {
		t.Fatal(err)
	}
	state, err := db.GetFanWalletState(event, club.ID, 0, "guest")
	if err != nil || state.Wallet != 65 || state.Points != 100 || len(state.Movements) != 3 || len(state.Collection.Items) != 2 || state.Collection.EquippedFrame != "club_frame" {
		t.Fatalf("purchases %+v: %v", state, err)
	}
	if state.Movements[0].Delta != -15 || state.Movements[1].Delta != -20 || state.Movements[0].Balance != 65 {
		t.Fatalf("real ledger %+v", state.Movements)
	}
	for _, item := range state.Catalog {
		if !item.Owned || (item.Key == "club_frame" && item.Cost != 20) || (item.Key == "match_card" && item.Cost != 15) {
			t.Fatalf("catalog %+v", state.Catalog)
		}
	}
	card := state.Collection.Items[0]
	if card.Key != "match_card" || card.ClubName != club.Name || card.Opponent != other.Name || card.EventID != event || card.EventDate != "2026-10-16T18:00:00Z" {
		t.Fatalf("real match metadata %+v", card)
	}
	next, err := db.GetFanWalletState(nextEvent, club.ID, 0, "guest")
	if err != nil || !next.Catalog[0].Owned || next.Catalog[1].Owned {
		t.Fatalf("club/event ownership %+v: %v", next.Catalog, err)
	}
	if err := db.PurchaseFanShopItem(nextEvent, club.ID, 0, "guest", "match_card"); !errors.Is(err, ErrFanShopFunds) {
		t.Fatalf("insufficient funds %v", err)
	}
	if err := db.PurchaseFanShopItem(event, other.ID, 0, "guest", "match_card"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("cross-org purchase %v", err)
	}
	if err := db.PurchaseFanShopItem(event, club.ID, 0, "guest", "free-shirt"); !errors.Is(err, ErrFanShopItem) {
		t.Fatalf("unknown reward %v", err)
	}
	if err := db.EquipFanShopItem(event, club.ID, 0, "guest", ""); err != nil {
		t.Fatal(err)
	}
	if err := db.EquipFanShopItem(event, club.ID, 0, "guest", "club_frame"); err != nil {
		t.Fatal(err)
	}
	if err := db.EquipFanShopItem(event, club.ID, 0, "not-owner", "club_frame"); !errors.Is(err, ErrFanShopNotOwned) {
		t.Fatalf("unowned equip %v", err)
	}
}

func TestFanEconomySpendingPreservesRankAndPoints(t *testing.T) {
	db, club, _, event, _ := fanEconomyFixture(t)
	first := fanEconomyRegister(t, db, club.ID, event, "first", "+390001", "first-session")
	second := fanEconomyRegister(t, db, club.ID, event, "second", "+390002", "second-session")
	if first.Wallet != 0 || second.Wallet != 0 {
		t.Fatal("client guest_coins must never mint funds")
	}
	if err := db.AddFanWalletCoins(first.Profile.ID, 100); err != nil {
		t.Fatal(err)
	}
	if err := db.AddFanWalletCoins(second.Profile.ID, 90); err != nil {
		t.Fatal(err)
	}
	if err := db.PurchaseFanShopItem(event, club.ID, first.Profile.ID, "first", "club_frame"); err != nil {
		t.Fatal(err)
	}
	if err := db.PurchaseFanShopItem(event, club.ID, first.Profile.ID, "first", "match_card"); err != nil {
		t.Fatal(err)
	}
	state, err := db.GetFanWalletState(event, club.ID, first.Profile.ID, "first")
	if err != nil || state.Wallet != 65 || state.Points != 100 {
		t.Fatalf("spendable vs points %+v: %v", state, err)
	}
	rank, err := db.GetFanRank(event, club.ID, first.Profile.ID)
	if err != nil || rank.Rank != 1 || rank.Coins != 100 || rank.Points != 100 || rank.EquippedFrame != "club_frame" {
		t.Fatalf("rank survives spending %+v: %v", rank, err)
	}
	leaders, err := db.GetFanLeaderboard(event, club.ID, 3)
	if err != nil || len(leaders) != 2 || leaders[0].EquippedFrame != "club_frame" || leaders[1].EquippedFrame != "" {
		t.Fatalf("leaderboard equipped frames %+v: %v", leaders, err)
	}
	nearby, err := db.GetFanNearbyRanks(club.ID, first.Profile.ID)
	if err != nil || len(nearby) != 2 || nearby[0].FanID != first.Profile.ID || nearby[0].Points != 100 || nearby[0].EquippedFrame != "club_frame" || nearby[1].EquippedFrame != "" {
		t.Fatalf("nearby score %+v: %v", nearby, err)
	}
	if _, _, err := db.DonateFanSupport(event, club.ID, first.Profile.ID, 5); err != nil {
		t.Fatal(err)
	}
	state, err = db.GetFanWalletState(event, club.ID, first.Profile.ID, "first")
	if err != nil || state.Wallet != 60 || state.Points != 100 {
		t.Fatalf("donation must not reduce score %+v: %v", state, err)
	}
}

func TestFanEconomyVerifiedLoginAdoptsGuestPurchasesOnce(t *testing.T) {
	db, club, _, event, _ := fanEconomyFixture(t)
	profile := fanEconomyRegister(t, db, club.ID, event, "account-device", "+390003", "verified-session")
	if err := db.AddFanWalletCoins(profile.Profile.ID, 100); err != nil {
		t.Fatal(err)
	}
	if err := db.UpsertGuestCoins(event, club.ID, "anonymous-device", 25); err != nil {
		t.Fatal(err)
	}
	if err := db.PurchaseFanShopItem(event, club.ID, 0, "anonymous-device", "club_frame"); err != nil {
		t.Fatal(err)
	}
	fanEconomyVote(t, db, club, event, "anonymous-device")
	for n := 0; n < 2; n++ {
		loggedIn, err := db.GetFanBySessionToken("verified-session", "anonymous-device")
		if err != nil || loggedIn.Wallet != 105 {
			t.Fatalf("login%d %+v: %v", n, loggedIn, err)
		}
	}
	state, err := db.GetFanWalletState(event, club.ID, profile.Profile.ID, "new-device")
	if err != nil || state.Wallet != 105 || state.Points != 125 || len(state.Collection.Items) != 1 || state.Collection.EquippedFrame != "club_frame" || state.Collection.MatchesFollowed != 1 {
		t.Fatalf("portable collection %+v: %v", state, err)
	}
	if state.Movements[0].Balance != 105 {
		t.Fatalf("consolidated ledger %+v", state.Movements)
	}
	var guestWallets int
	if err := db.c.QueryRow(`SELECT COUNT(*) FROM guest_wallets WHERE organization_id=? AND device_id='anonymous-device'`, club.ID).Scan(&guestWallets); err != nil || guestWallets != 0 {
		t.Fatalf("guest balance consumed once %d %v", guestWallets, err)
	}
	// A fresh guest who only voted still brings that verified participation
	// across devices, even without an earned bonus or a wallet row.
	fanEconomyVote(t, db, club, event, "vote-only-device")
	if _, err := db.GetFanBySessionToken("verified-session", "vote-only-device"); err != nil {
		t.Fatal(err)
	}
	var userID int
	if err := db.c.QueryRow(`SELECT user_id FROM votes WHERE device_id='vote-only-device'`).Scan(&userID); err != nil || userID != profile.Profile.ID {
		t.Fatalf("vote-only adoption %d: %v", userID, err)
	}
}

func TestFanEconomyRegistrationRequiresLoginForExistingPhone(t *testing.T) {
	db, club, _, event, _ := fanEconomyFixture(t)
	profile := fanEconomyRegister(t, db, club.ID, event, "owner", "+390004", "owner-session")
	if err := db.AddFanWalletCoins(profile.Profile.ID, 50); err != nil {
		t.Fatal(err)
	}
	_, err := db.RegisterFan(FanRegisterInput{OrganizationID: club.ID, EventID: event, DeviceID: "attacker", Phone: "+390004", Nickname: "Other", SessionToken: "new-token", GuestCoins: 999999})
	if !errors.Is(err, ErrFanLoginRequired) {
		t.Fatalf("existing phone without verified session %v", err)
	}
	updated, err := db.RegisterFan(FanRegisterInput{OrganizationID: club.ID, EventID: event, DeviceID: "owner", Phone: "+390004", Nickname: "Updated", SessionToken: "owner-session", GuestCoins: 999999})
	if err != nil || updated.Profile.ID != profile.Profile.ID || updated.Wallet != 50 {
		t.Fatalf("authorized profile update %+v: %v", updated, err)
	}
}

func TestFanEconomyLegacyClaimsAreFixedAndBadgeUsesProof(t *testing.T) {
	db, club, _, event, _ := fanEconomyFixture(t)
	total := 0
	for _, key := range []string{"reaction", "tap", "sponsor-rush", "memory-flash"} {
		amount := FanJourneyRewardAmount(key)
		total += amount
		wallet, credited, err := db.ClaimFanJourneyReward(event, club.ID, 0, "player", key)
		if err != nil || wallet != total || credited != amount {
			t.Fatalf("claim %s %d/%d: %v", key, wallet, credited, err)
		}
		wallet, credited, err = db.ClaimFanJourneyReward(event, club.ID, 0, "player", key)
		if err != nil || wallet != total || credited != 0 {
			t.Fatalf("claim retry %s %d/%d: %v", key, wallet, credited, err)
		}
	}
	if err := db.PurchaseFanShopItem(event, club.ID, 0, "player", "club_frame"); err != nil {
		t.Fatal(err)
	}
	state, err := db.GetFanWalletState(event, club.ID, 0, "player")
	if err != nil || state.Wallet != 18 || state.Points != 38 || len(state.Collection.Badges) != 1 || state.Collection.Badges[0].EventID != event || state.Collection.MatchesFollowed != 1 {
		t.Fatalf("proof badge survives spending %+v: %v", state, err)
	}
	if err := db.UpsertGuestCoins(event, club.ID, "unproven", 100); err != nil {
		t.Fatal(err)
	}
	unproven, err := db.GetFanWalletState(event, club.ID, 0, "unproven")
	if err != nil || len(unproven.Collection.Badges) != 0 || unproven.Collection.MatchesFollowed != 0 {
		t.Fatalf("no invented participation %+v: %v", unproven.Collection, err)
	}
}

func TestFanEconomyMigrationRecordsOpeningWithoutFakeHistory(t *testing.T) {
	db, club, _, event, _ := fanEconomyFixture(t)
	profile := fanEconomyRegister(t, db, club.ID, event, "legacy", "+390005", "legacy-session")
	for _, name := range []string{"trg_economy_fan_wallets_insert", "trg_economy_fan_wallets_update", "trg_economy_guest_wallets_insert", "trg_economy_guest_wallets_update"} {
		if _, err := db.c.Exec(`DROP TRIGGER ` + name); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.c.Exec(`DELETE FROM fan_economy_accounts; DELETE FROM fan_wallet_movements`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.c.Exec(`UPDATE fan_wallets SET coins=80 WHERE fan_id=?`, profile.Profile.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.c.Exec(`INSERT INTO fan_reward_redemptions(event_id,fan_id,reward_key,cost_coins) VALUES(?,?,'old-confirmed',20)`, event, profile.Profile.ID); err != nil {
		t.Fatal(err)
	}
	for n := 0; n < 2; n++ {
		if err := ensureFanEconomyTables(db.c); err != nil {
			t.Fatal(err)
		}
	}
	state, err := db.GetFanWalletState(event, club.ID, profile.Profile.ID, "legacy")
	if err != nil || state.Wallet != 80 || state.Points != 100 || len(state.Movements) != 1 || state.Movements[0].Label != "Saldo iniziale" || state.Movements[0].Delta != 80 {
		t.Fatalf("honest opening %+v: %v", state, err)
	}
	if err := db.AddFanWalletCoins(profile.Profile.ID, 5); err != nil {
		t.Fatal(err)
	}
	state, err = db.GetFanWalletState(event, club.ID, profile.Profile.ID, "legacy")
	if err != nil || state.Wallet != 85 || state.Points != 105 || len(state.Movements) != 2 {
		t.Fatalf("trigger after migration %+v: %v", state, err)
	}
}

func TestFanEconomySwitchAccountDoesNotRestoreAnotherFansRewards(t *testing.T) {
	db, club, _, event, _ := fanEconomyFixture(t)
	first := fanEconomyRegister(t, db, club.ID, event, "first", "+390006", "first-session")
	second := fanEconomyRegister(t, db, club.ID, event, "second", "+390007", "second-session")
	if _, _, err := db.ClaimFanJourneyReward(event, club.ID, first.Profile.ID, "shared", "memory-flash"); err != nil {
		t.Fatal(err)
	}
	other, err := db.ListFanJourneyRewards(event, club.ID, second.Profile.ID, "shared")
	if err != nil || len(other) != 0 {
		t.Fatalf("account must not inherit device owner's rewards %+v: %v", other, err)
	}
	if _, _, err := db.ClaimFanJourneyReward(event, club.ID, second.Profile.ID, "shared", "memory-flash"); !errors.Is(err, ErrFanRewardDeviceUsed) {
		t.Fatalf("already used device must not claim another owner's reward: %v", err)
	}
	now := time.Now()
	shot, err := db.CreateBasketballShot(event, club.ID, first.Profile.ID, "shared", now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.ResolveBasketballShot(event, club.ID, first.Profile.ID, "shared", shot.ShotID, shot.Hoop.X-180, shot.Hoop.Y-466, now.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	shot, err = db.CreateBasketballShot(event, club.ID, second.Profile.ID, "shared", now.Add(2*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	result, err := db.ResolveBasketballShot(event, club.ID, second.Profile.ID, "shared", shot.ShotID, shot.Hoop.X-180, shot.Hoop.Y-466, now.Add(3*time.Second))
	if err != nil || result.BasketballEarned != 1 || result.Wallet != 1 {
		t.Fatalf("game total must follow owner %+v: %v", result, err)
	}
	other, err = db.ListFanJourneyRewards(event, club.ID, second.Profile.ID, "shared")
	if err != nil || other["basketball"] != 1 || other["memory-flash"] != 0 {
		t.Fatalf("owned journey only %+v: %v", other, err)
	}
	guest, err := db.ListFanJourneyRewards(event, club.ID, 0, "shared")
	if err != nil || len(guest) != 0 {
		t.Fatalf("signed-out device must not expose registered progress %+v: %v", guest, err)
	}
}

func TestFanEconomyTapLiveRestoresCumulativeJourneyAndBadge(t *testing.T) {
	db, club, _, event, _ := fanEconomyFixture(t)
	first := fanEconomyRegister(t, db, club.ID, event, "winner", "+390008", "winner-session")
	second := fanEconomyRegister(t, db, club.ID, event, "loser", "+390009", "loser-session")
	if _, err := db.c.Exec(`INSERT INTO tap_live_matches(match_id,event_id,organization_id,fan1_id,fan2_id,status,fan1_score,fan2_score,fan1_coins,fan2_coins) VALUES('verified-match',?,?,?,?,'finished',3,1,30,8)`, event, club.ID, first.Profile.ID, second.Profile.ID); err != nil {
		t.Fatal(err)
	}
	if err := db.AddFanWalletCoins(first.Profile.ID, 30); err != nil {
		t.Fatal(err)
	}
	restored, err := db.ListFanJourneyRewards(event, club.ID, first.Profile.ID, "another-device")
	if err != nil || restored["tap"] != 30 {
		t.Fatalf("live tap restoration %+v: %v", restored, err)
	}
	state, err := db.GetFanWalletState(event, club.ID, first.Profile.ID, "another-device")
	if err != nil || state.Wallet != 30 || len(state.Collection.Badges) != 1 || state.Collection.Badges[0].EventID != event {
		t.Fatalf("tap badge restoration %+v: %v", state, err)
	}
	if _, _, err := db.ClaimFanJourneyReward(event, club.ID, first.Profile.ID, "winner", "tap"); err != nil {
		t.Fatal(err)
	}
	restored, err = db.ListFanJourneyRewards(event, club.ID, first.Profile.ID, "another-device")
	if err != nil || restored["tap"] != 38 {
		t.Fatalf("local fixed tap plus live cumulative %+v: %v", restored, err)
	}
	guest, err := db.ListFanJourneyRewards(event, club.ID, 0, "winner")
	if err != nil || guest["tap"] != 0 {
		t.Fatalf("live rewards remain fan-owned %+v: %v", guest, err)
	}
}
