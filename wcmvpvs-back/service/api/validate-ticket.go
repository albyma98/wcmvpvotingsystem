package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
)

func (rt *_router) validateTicket(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	var req struct {
		Code      string `json:"code"`
		Signature string `json:"signature"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ctx.Logger.WithError(err).Warn("invalid ticket validation payload")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	code := strings.TrimSpace(req.Code)
	signature := strings.TrimSpace(req.Signature)
	if code == "" || signature == "" {
		ctx.Logger.Warn("ticket validation missing code or signature")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	vote, err := rt.db.GetVoteByCodeAndSignature(code, signature)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			w.Header().Set("content-type", "application/json")
			_ = json.NewEncoder(w).Encode(struct {
				Status  string `json:"status"`
				Message string `json:"message"`
			}{Status: "invalid", Message: "Ticket non valido"})
			return
		}

		ctx.Logger.WithError(err).Error("cannot retrieve vote for ticket validation")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	alreadyScanned, err := rt.db.RecordTicketScan(vote.ID, code, signature)
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot record ticket scan")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	status := "valid"
	message := "Ticket valido"
	if alreadyScanned {
		status = "already_scanned"
		message = "Già scannerizzato"
	}

	w.Header().Set("content-type", "application/json")
	_ = json.NewEncoder(w).Encode(struct {
		Status   string `json:"status"`
		Message  string `json:"message"`
		VoteID   int    `json:"vote_id"`
		EventID  int    `json:"event_id"`
		PlayerID int    `json:"player_id"`
	}{Status: status, Message: message, VoteID: vote.ID, EventID: vote.EventID, PlayerID: vote.PlayerID})
}
