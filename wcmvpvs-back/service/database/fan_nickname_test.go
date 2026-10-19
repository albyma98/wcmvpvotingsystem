package database

import (
	"errors"
	"testing"
)

func TestFanNicknameUniquenessByOrganization(t *testing.T) {
	db, club, other, event, _ := fanEconomyFixture(t)
	register := func(org, event int, phone, nickname, token string) (FanProfileSummary, error) {
		return db.RegisterFan(FanRegisterInput{OrganizationID: org, EventID: event, Phone: phone, Nickname: nickname, SessionToken: token, AcceptedTerms: true})
	}
	first, err := register(club.ID, event, "+391111", "CurvaNord", "first")
	if err != nil {
		t.Fatal(err)
	}
	available, err := db.IsFanNicknameAvailable(club.ID, " curvanord ", 0)
	if err != nil || available {
		t.Fatalf("case and surrounding spaces must match: %v %v", available, err)
	}
	available, err = db.IsFanNicknameAvailable(other.ID, "CurvaNord", 0)
	if err != nil || !available {
		t.Fatalf("other club must remain available: %v %v", available, err)
	}
	if _, err := register(club.ID, event, "+392222", " CURVANORD ", "duplicate"); !errors.Is(err, ErrFanNicknameTaken) {
		t.Fatalf("duplicate registration: %v", err)
	}
	if _, err := register(other.ID, 0, "+392222", "curvanord", "other"); err != nil {
		t.Fatalf("same name in another organization: %v", err)
	}
	second, err := register(club.ID, event, "+393333", "TribunaVIP", "second")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.UpdateFanNickname(second.Profile.ID, "CurvaNord"); !errors.Is(err, ErrFanNicknameTaken) {
		t.Fatalf("nickname update must also enforce uniqueness: %v", err)
	}
	if _, err := register(club.ID, event, "+393333", "curvanord", "second"); !errors.Is(err, ErrFanNicknameTaken) {
		t.Fatalf("registration update must enforce uniqueness: %v", err)
	}
	if _, err := register(club.ID, event, "+391111", "CURVANORD", "first"); err != nil {
		t.Fatalf("own nickname may change case: %v", err)
	}
	available, err = db.IsFanNicknameAvailable(club.ID, "CurvaNord", first.Profile.ID)
	if err != nil || !available {
		t.Fatalf("own nickname excluded from profile edits: %v %v", available, err)
	}
	// Database enforcement applies even when a caller skipped an earlier check.
	if _, err := db.c.Exec(`INSERT INTO fan_profiles(organization_id,nickname,phone) VALUES(?, ' curvanord ', '+394444')`, club.ID); err == nil {
		t.Fatal("direct insert bypassed nickname uniqueness")
	}
}
