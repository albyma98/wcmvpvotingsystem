package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
	"github.com/go-chi/chi/v5"
)

var (
	emailRegex = regexp.MustCompile(`^[A-Za-z0-9.!#$%&'*+/=?^_` + "`" + `{|}~-]+@[A-Za-z0-9](?:[A-Za-z0-9-]{0,61}[A-Za-z0-9])?(?:\.[A-Za-z0-9](?:[A-Za-z0-9-]{0,61}[A-Za-z0-9])?)+$`)
	phoneRegex = regexp.MustCompile(`^\+?[0-9]{6,15}$`)
)

type contactSubmissionRequest struct {
	ContactValue     string `json:"contact_value"`
	ContactType      string `json:"contact_type"`
	DeviceID         string `json:"device_id"`
	MarketingConsent bool   `json:"marketing_consent"`
	Timestamp        string `json:"timestamp"`
}

type contactSubmissionResponse struct {
	Message          string `json:"message"`
	BadgeLabel       string `json:"badge_label,omitempty"`
	BonusCode        string `json:"bonus_code,omitempty"`
	BonusSignature   string `json:"bonus_signature,omitempty"`
	AlreadySubmitted bool   `json:"already_submitted,omitempty"`
	ContactType      string `json:"contact_type,omitempty"`
}

func normalizeContactValue(value string, contactType string) string {
	trimmed := strings.TrimSpace(value)
	if contactType == "phone" {
		cleaned := strings.Map(func(r rune) rune {
			if r == '+' || (r >= '0' && r <= '9') {
				return r
			}
			return -1
		}, trimmed)
		if strings.Count(cleaned, "+") > 1 {
			cleaned = strings.TrimLeft(cleaned, "+")
		}
		if strings.HasPrefix(cleaned, "++") {
			cleaned = strings.TrimLeft(cleaned, "+")
		}
		return cleaned
	}

	if contactType == "email" {
		return strings.ToLower(trimmed)
	}

	return trimmed
}

func detectContactType(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}

	if emailRegex.MatchString(trimmed) {
		return "email"
	}

	if phoneRegex.MatchString(normalizeContactValue(trimmed, "phone")) {
		return "phone"
	}

	return ""
}

func (rt *_router) submitContact(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	eventID, err := strconv.Atoi(chi.URLParam(r, "eventId"))
	if err != nil || eventID <= 0 {
		_ = writeJSONMessage(w, http.StatusBadRequest, "Evento non valido.")
		return
	}

	if !rt.ensureEventInOrganization(w, ctx, eventID) {
		return
	}

	var payload contactSubmissionRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		_ = writeJSONMessage(w, http.StatusBadRequest, "Formato della richiesta non valido.")
		return
	}

	deviceID := strings.TrimSpace(payload.DeviceID)
	if deviceID == "" {
		deviceID = rt.deviceIDFromRequest(r)
	}
	if deviceID == "" {
		_ = writeJSONMessage(w, http.StatusBadRequest, "Identificativo dispositivo mancante.")
		return
	}

	if !payload.MarketingConsent {
		_ = writeJSONMessage(w, http.StatusBadRequest, "Serve il consenso per ricevere il bonus chance.")
		return
	}

	contactType := strings.ToLower(strings.TrimSpace(payload.ContactType))
	detectedType := detectContactType(payload.ContactValue)
	if detectedType == "" {
		_ = writeJSONMessage(w, http.StatusBadRequest, "Inserisci un contatto valido.")
		return
	}

	if contactType == "" {
		contactType = detectedType
	}

	if contactType != detectedType {
		_ = writeJSONMessage(w, http.StatusBadRequest, "Il formato del contatto non corrisponde al tipo indicato.")
		return
	}

	normalizedValue := normalizeContactValue(payload.ContactValue, contactType)
	if normalizedValue == "" {
		_ = writeJSONMessage(w, http.StatusBadRequest, "Inserisci un contatto valido.")
		return
	}

	existing, err := rt.db.GetContactSubmission(eventID, deviceID)
	switch {
	case err == nil:
		_ = writeJSON(w, http.StatusConflict, contactSubmissionResponse{
			Message:          "Hai già massimizzato le tue chance per oggi!",
			BadgeLabel:       "+1 CHANCE",
			BonusCode:        existing.BonusCode,
			BonusSignature:   existing.BonusSignature,
			AlreadySubmitted: true,
			ContactType:      existing.ContactType,
		})
		return
	case errors.Is(err, sql.ErrNoRows):
	default:
		ctx.Logger.WithError(err).Error("cannot check existing contact submission")
		_ = writeJSONMessage(w, http.StatusInternalServerError, "Servizio non disponibile. Riprova tra poco.")
		return
	}

	var bonusCode string
	var bonusSignature string

	for attempt := 0; attempt < maxCodeGenerationAttempts; attempt++ {
		bonusCode, err = generateNumericCode()
		if err != nil {
			ctx.Logger.WithError(err).Error("cannot generate bonus code")
			_ = writeJSONMessage(w, http.StatusInternalServerError, "Impossibile assegnare il bonus ora.")
			return
		}
		bonusSignature = signCode(rt.VoteSecret, bonusCode)

		_, err = rt.db.RecordContactSubmission(database.ContactSubmission{
			EventID:          eventID,
			DeviceID:         deviceID,
			ContactValue:     normalizedValue,
			ContactType:      contactType,
			MarketingConsent: true,
			BonusCode:        bonusCode,
			BonusSignature:   bonusSignature,
		})
		if err != nil {
			if isUniqueConstraintError(err) {
				_ = writeJSON(w, http.StatusConflict, contactSubmissionResponse{
					Message:          "Hai già massimizzato le tue chance per oggi!",
					BadgeLabel:       "+1 CHANCE",
					AlreadySubmitted: true,
				})
				return
			}
			ctx.Logger.WithError(err).Error("cannot store contact submission")
			_ = writeJSONMessage(w, http.StatusInternalServerError, "Servizio non disponibile. Riprova più tardi.")
			return
		}
		break
	}

	if bonusCode == "" {
		_ = writeJSONMessage(w, http.StatusInternalServerError, "Servizio non disponibile. Riprova più tardi.")
		return
	}

	if err := rt.db.RecordContactEvent(eventID, deviceID, "contactSubmitted"); err != nil {
		ctx.Logger.WithError(err).Warn("contact analytics event failed")
	}

	if err := writeJSON(w, http.StatusCreated, contactSubmissionResponse{
		Message:        "🎉 Extra chance aggiunta! Buona fortuna!",
		BadgeLabel:     "+1 CHANCE",
		BonusCode:      bonusCode,
		BonusSignature: bonusSignature,
		ContactType:    contactType,
	}); err != nil {
		ctx.Logger.WithError(err).Error("cannot encode contact submission response")
	}
}
