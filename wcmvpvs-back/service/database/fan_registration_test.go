package database

import (
	"errors"
	"testing"
	"time"
)

func TestFanRegistrationRecordsVerifiedAcceptanceAndAutomaticGadgetEntry(t *testing.T) {
	db, club, other, event, _ := fanEconomyFixture(t)
	const phone = "+393331234567"
	pending, err := db.CreateFanWithPhoneE164(club.ID, phone)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.UpsertFanSession("verified-session", pending.ID, "device"); err != nil {
		t.Fatal(err)
	}
	input := FanRegisterInput{OrganizationID: club.ID, EventID: event, DeviceID: "device", SessionToken: "verified-session", Nickname: "CurvaFan", Phone: phone, AcceptedTerms: true,
		Acceptance: &FanRegistrationAcceptance{AgeEligibilityDeclared: true, MinimumAgeDeclared: FanMinimumAge, LegalVersion: FanLegalVersion}}
	if _, err := db.RegisterFan(input); !errors.Is(err, ErrFanVerificationRequired) {
		t.Fatalf("unverified registration: %v", err)
	}
	var receipts int
	if err := db.c.QueryRow(`SELECT COUNT(*) FROM fan_registration_acceptances`).Scan(&receipts); err != nil || receipts != 0 {
		t.Fatalf("premature receipt: %d %v", receipts, err)
	}
	if err := db.MarkFanPhoneVerified(club.ID, phone, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	wrongClub := input
	wrongClub.OrganizationID, wrongClub.EventID = other.ID, 0
	if _, err := db.RegisterFan(wrongClub); !errors.Is(err, ErrFanVerificationRequired) {
		t.Fatalf("cross-club proof: %v", err)
	}
	wrongPhone := input
	wrongPhone.Phone = "+393339999999"
	if _, err := db.RegisterFan(wrongPhone); err == nil {
		t.Fatal("session verified a different number")
	}
	input.SessionToken = "unknown"
	if _, err := db.RegisterFan(input); !errors.Is(err, ErrFanVerificationRequired) {
		t.Fatalf("unknown session: %v", err)
	}
	input.SessionToken = "verified-session"
	profile, err := db.RegisterFan(input)
	if err != nil {
		t.Fatal(err)
	}
	var version, acceptedAt, proof string
	var minimumAge int
	if err := db.c.QueryRow(`SELECT legal_version,minimum_age_declared,accepted_at,phone_verified_at FROM fan_registration_acceptances WHERE fan_id=? AND organization_id=?`, profile.Profile.ID, club.ID).Scan(&version, &minimumAge, &acceptedAt, &proof); err != nil {
		t.Fatal(err)
	}
	if version != FanLegalVersion || minimumAge != 14 || acceptedAt == "" || proof == "" {
		t.Fatalf("incomplete receipt: %s %d %s %s", version, minimumAge, acceptedAt, proof)
	}
	var entries int
	if err := db.c.QueryRow(`SELECT COUNT(*) FROM fan_lottery_entries WHERE event_id=? AND fan_id=?`, event, profile.Profile.ID).Scan(&entries); err != nil || entries != 1 {
		t.Fatalf("automatic entry without prior vote: %d %v", entries, err)
	}
	if _, err := db.RegisterFan(input); err != nil {
		t.Fatal(err)
	}
	if err := db.c.QueryRow(`SELECT COUNT(*) FROM fan_registration_acceptances WHERE fan_id=?`, profile.Profile.ID).Scan(&receipts); err != nil || receipts != 1 {
		t.Fatalf("duplicate receipts: %d %v", receipts, err)
	}
	if err := db.c.QueryRow(`SELECT COUNT(*) FROM fan_lottery_entries WHERE event_id=? AND fan_id=?`, event, profile.Profile.ID).Scan(&entries); err != nil || entries != 1 {
		t.Fatalf("duplicate entry: %d %v", entries, err)
	}
	audience, err := db.ListMarketingAudience(club.ID, "", true)
	if err != nil || len(audience) != 0 {
		t.Fatalf("terms granted marketing: %+v %v", audience, err)
	}
}

func TestFanRegistrationMigrationsPreserveExistingProfilesWithoutInventingAcceptance(t *testing.T) {
	db, club, _, event, _ := fanEconomyFixture(t)
	profile := fanEconomyRegister(t, db, club.ID, event, "old-device", "+393330000000", "old-session")
	if err := ensureFanProfileTables(db.c); err != nil {
		t.Fatal(err)
	}
	var receipts int
	if err := db.c.QueryRow(`SELECT COUNT(*) FROM fan_registration_acceptances WHERE fan_id=?`, profile.Profile.ID).Scan(&receipts); err != nil || receipts != 0 {
		t.Fatalf("invented historical acceptance: %d %v", receipts, err)
	}
	if _, err := db.GetFanBySessionToken("old-session", ""); err != nil {
		t.Fatalf("existing login broken: %v", err)
	}
}
