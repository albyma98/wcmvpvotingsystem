package database

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"strconv"
	"strings"
)

// Version of the terms, privacy notice and gadget rules presented by the client.
const FanLegalVersion = "2026-11-01.1"
const FanMinimumAge = 14

const fanMatchdayRegistrationSchema = `CREATE TABLE IF NOT EXISTS fan_matchday_registrations (
	fan_id INTEGER PRIMARY KEY REFERENCES fan_profiles(id) ON DELETE CASCADE,
	organization_id INTEGER NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
	event_id INTEGER NOT NULL REFERENCES events(id) ON DELETE CASCADE,
	device_key TEXT NOT NULL DEFAULT '',
	registered_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);`

func recordFanMatchdayRegistration(tx *sql.Tx, input FanRegisterInput, fanID int) error {
	deviceKey := ""
	if device := strings.TrimSpace(input.DeviceID); device != "" {
		sum := sha256.Sum256([]byte(strconv.Itoa(input.OrganizationID) + ":" + device))
		deviceKey = hex.EncodeToString(sum[:])
	}
	_, err := tx.Exec(`INSERT INTO fan_matchday_registrations(fan_id,organization_id,event_id,device_key)
		VALUES(?,?,?,?) ON CONFLICT(fan_id) DO NOTHING`, fanID, input.OrganizationID, input.EventID, deviceKey)
	return err
}

var (
	ErrFanAgeRequired          = errors.New("fan age eligibility declaration required")
	ErrFanTermsRequired        = errors.New("fan terms acceptance required")
	ErrFanLegalVersion         = errors.New("fan legal document version mismatch")
	ErrFanVerificationRequired = errors.New("verified pending fan session required")
)

type FanRegistrationAcceptance struct {
	AgeEligibilityDeclared bool   `json:"age_eligibility_declared"`
	MinimumAgeDeclared     int    `json:"minimum_age_declared"`
	LegalVersion           string `json:"legal_version"`
}

func (a FanRegistrationAcceptance) Validate(acceptedTerms bool) error {
	if !a.AgeEligibilityDeclared || a.MinimumAgeDeclared != FanMinimumAge {
		return ErrFanAgeRequired
	}
	if !acceptedTerms {
		return ErrFanTermsRequired
	}
	if a.LegalVersion != FanLegalVersion {
		return ErrFanLegalVersion
	}
	return nil
}
