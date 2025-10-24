package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
)

func (rt *_router) validateTicket(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	eventID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil || eventID <= 0 {
		ctx.Logger.Warn("invalid event id while validating ticket")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var req struct {
		Code      string `json:"code"`
		Signature string `json:"signature"`
		QRData    string `json:"qr_data"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ctx.Logger.WithError(err).Warn("cannot decode ticket validation payload")
		writeJSONError(w, http.StatusBadRequest, "invalid_payload")
		return
	}

	code := strings.TrimSpace(req.Code)
	signature := strings.TrimSpace(req.Signature)

	if code == "" && strings.TrimSpace(req.QRData) != "" {
		var qrPayload struct {
			Code      string `json:"code"`
			Signature string `json:"signature"`
		}
		if err := json.Unmarshal([]byte(req.QRData), &qrPayload); err == nil {
			code = strings.TrimSpace(qrPayload.Code)
			signature = strings.TrimSpace(qrPayload.Signature)
		}
	}

	if code == "" || signature == "" {
		ctx.Logger.Warn("missing ticket code or signature in validation request")
		writeJSONError(w, http.StatusBadRequest, "invalid_ticket_data")
		return
	}

	expectedSignature := signCode(rt.VoteSecret, code)
	if !strings.EqualFold(expectedSignature, signature) {
		ctx.Logger.WithFields(map[string]interface{}{
			"event_id":    eventID,
			"ticket_code": code,
		}).Warn("ticket signature mismatch in validation request")
		writeJSONError(w, http.StatusBadRequest, "invalid_signature")
		return
	}

	result, err := rt.db.ValidateTicket(eventID, code)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(w, http.StatusNotFound, "ticket_not_found")
			return
		}
		ctx.Logger.WithError(err).Error("cannot validate ticket")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if !strings.EqualFold(result.TicketSignature, expectedSignature) {
		ctx.Logger.WithFields(map[string]interface{}{
			"event_id":    result.EventID,
			"ticket_code": result.TicketCode,
		}).Error("stored ticket signature mismatch during validation")
		writeJSONError(w, http.StatusConflict, "ticket_signature_mismatch")
		return
	}

	result.TicketSignature = expectedSignature

	resp := struct {
		Valid  bool                            `json:"valid"`
		Ticket database.TicketValidationResult `json:"ticket"`
	}{Valid: true, Ticket: result}

	w.Header().Set("content-type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		ctx.Logger.WithError(err).Error("cannot encode ticket validation response")
	}
	ctx.Logger.WithFields(map[string]interface{}{
		"event_id":    eventID,
		"ticket_code": code,
	}).Info("ticket validated successfully")
}

func writeJSONError(w http.ResponseWriter, status int, code string) {
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": code})
}
