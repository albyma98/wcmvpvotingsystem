package database

import "errors"

// Version of the terms, privacy notice and gadget rules presented by the client.
const FanLegalVersion = "2026-10-25.1"
const FanMinimumAge = 14

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
