package database

import (
	"database/sql"
	"errors"
	"testing"
	"time"
)

func TestFanPhoneProfilesAreDistinctByOrganization(t *testing.T) {
	db, club, other, event, _ := fanEconomyFixture(t)
	const phone = "+393331234567"
	first := fanEconomyRegister(t, db, club.ID, event, "shared-device", phone, "first-session")
	if err := db.AddFanWalletCoins(first.Profile.ID, 50); err != nil {
		t.Fatal(err)
	}
	// Reproduce an existing installation with the old platform-wide constraint.
	if _, err := db.c.Exec(`CREATE UNIQUE INDEX idx_fan_profiles_phone_e164_unique ON fan_profiles(phone_e164)`); err != nil {
		t.Fatal(err)
	}
	if err := ensureFanProfileTables(db.c); err != nil {
		t.Fatal(err)
	}
	pending, err := db.CreateFanWithPhoneE164(other.ID, phone)
	if err != nil || pending.ID == first.Profile.ID || pending.OrganizationID != other.ID || !pending.RegistrationPending {
		t.Fatalf("other club's pending profile: %+v %v", pending, err)
	}
	if _, err := db.CreateFanWithPhoneE164(other.ID, phone); err == nil {
		t.Fatal("duplicate phone accepted in the same organization")
	}
	if err := db.MarkFanPhoneVerified(other.ID, phone, time.Now()); err != nil {
		t.Fatal(err)
	}
	var verified string
	if err := db.c.QueryRow(`SELECT IFNULL(phone_verified_at,'') FROM fan_profiles WHERE id=?`, first.Profile.ID).Scan(&verified); err != nil || verified != "" {
		t.Fatalf("verification changed another organization's profile: %q %v", verified, err)
	}
	if err := db.UpsertFanSession("second-session", pending.ID, "shared-device"); err != nil {
		t.Fatal(err)
	}
	second, err := db.RegisterFan(FanRegisterInput{OrganizationID: other.ID, Phone: phone, Nickname: "OtherClubFan", AcceptedTerms: true, SessionToken: "second-session", DeviceID: "shared-device"})
	if err != nil || second.Profile.ID != pending.ID || second.Wallet != 0 || second.Profile.RegistrationPending {
		t.Fatalf("independent second registration: %+v %v", second, err)
	}
	for _, tc := range []struct {
		org, id, balance int
		token            string
	}{{club.ID, first.Profile.ID, 50, "first-session"}, {other.ID, second.Profile.ID, 0, "second-session"}} {
		profile, err := db.GetFanByPhoneE164(tc.org, phone)
		if err != nil || profile.ID != tc.id || profile.RegistrationPending {
			t.Fatalf("organization lookup: %+v %v", profile, err)
		}
		session, err := db.GetFanBySessionToken(tc.token, "shared-device")
		if err != nil || session.Profile.ID != tc.id || session.Wallet != tc.balance {
			t.Fatalf("separate session/wallet: %+v %v", session, err)
		}
		device, err := db.GetFanByDevice(0, tc.org, "shared-device")
		if err != nil || device.Profile.ID != tc.id {
			t.Fatalf("device lookup escaped organization: %+v %v", device, err)
		}
	}
	if _, err := db.GetFanByPhoneE164(0, phone); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("global phone lookup allowed: %v", err)
	}
	if err := db.MarkFanPhoneVerified(other.ID+100, phone, time.Now()); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("verification found another organization: %v", err)
	}
	// Restarting with the same phone in multiple clubs must not restore the old index.
	if err := ensureFanProfileTables(db.c); err != nil {
		t.Fatalf("schema is not restart-safe: %v", err)
	}
}

func TestFanPhoneRegistrationLookupIsOrganizationScoped(t *testing.T) {
	db, club, other, event, _ := fanEconomyFixture(t)
	const phone = "+393331234567"
	first := fanEconomyRegister(t, db, club.ID, event, "first", phone, "first-token")
	second := fanEconomyRegister(t, db, other.ID, 0, "second", phone, "second-token")
	if first.Profile.ID == second.Profile.ID {
		t.Fatal("registrations merged")
	}
	if _, err := db.RegisterFan(FanRegisterInput{OrganizationID: other.ID, Phone: phone, Nickname: "AnotherName", SessionToken: "fresh-token"}); !errors.Is(err, ErrFanLoginRequired) {
		t.Fatalf("existing phone in own organization must require login: %v", err)
	}
	if _, err := db.RegisterFan(FanRegisterInput{OrganizationID: other.ID, Phone: phone, Nickname: "WrongTenant", SessionToken: "first-token"}); !errors.Is(err, ErrInvalidSponsorData) {
		t.Fatalf("cross-organization session accepted: %v", err)
	}
	profile, err := db.GetFanByPhoneE164(club.ID, phone)
	if err != nil || profile.Nickname != first.Profile.Nickname {
		t.Fatalf("original profile changed: %+v %v", profile, err)
	}
}
