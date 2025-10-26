package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
)

func (rt *_router) ticketValidationPage(w http.ResponseWriter, r *http.Request) {
	logger := rt.baseLogger

	query := r.URL.Query()
	eventIDStr := strings.TrimSpace(query.Get("e"))
	code := strings.TrimSpace(query.Get("c"))
	signature := strings.TrimSpace(query.Get("s"))

	renderInvalid := func(status int) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(status)
		_, _ = fmt.Fprint(w, ticketValidationHTML("❌ QR NON VALIDO", ""))
	}

	if eventIDStr == "" || code == "" || signature == "" {
		logger.Warn("ticket validation request missing parameters")
		renderInvalid(http.StatusBadRequest)
		return
	}

	eventID, err := strconv.Atoi(eventIDStr)
	if err != nil || eventID <= 0 {
		logger.WithField("event_id", eventIDStr).Warn("ticket validation request with invalid event id")
		renderInvalid(http.StatusBadRequest)
		return
	}

	outcome, status := rt.processTicketValidation(logger, eventID, code, signature)
	if outcome.ErrorCode != "" {
		if status >= http.StatusInternalServerError {
			renderInvalid(http.StatusInternalServerError)
		} else {
			renderInvalid(http.StatusBadRequest)
		}
		return
	}

	message := "✅ QR VALIDO"
	detail := ""
	if outcome.AlreadyRedeemed {
		detail = "Questo ticket risulta già riscattato."
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprint(w, ticketValidationHTML(message, detail))

	logger.WithFields(map[string]interface{}{
		"event_id":        eventID,
		"code":            code,
		"alreadyRedeemed": outcome.AlreadyRedeemed,
	}).Info("ticket validation completed")
}

func (rt *_router) buildTicketValidationURL(eventID int, code, signature string) (string, error) {
	baseURL := strings.TrimSpace(rt.ticketValidationBaseURL)
	if baseURL == "" {
		return "", errors.New("ticket validation base URL is not configured")
	}

	parsed, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}

	parsed.Path = strings.TrimSuffix(parsed.Path, "/") + "/lottery/validate"
	q := parsed.Query()
	q.Set("e", strconv.Itoa(eventID))
	q.Set("c", code)
	q.Set("s", signature)
	parsed.RawQuery = q.Encode()

	return parsed.String(), nil
}

type ticketValidationOutcome struct {
	AlreadyRedeemed bool
	ErrorCode       string
}

func (rt *_router) processTicketValidation(logger logrus.FieldLogger, eventID int, code, signature string) (ticketValidationOutcome, int) {
	var outcome ticketValidationOutcome

	if eventID <= 0 {
		logger.WithField("event_id", eventID).Warn("ticket validation with invalid event id")
		outcome.ErrorCode = "invalid_event_id"
		return outcome, http.StatusBadRequest
	}

	code = strings.TrimSpace(code)
	signature = strings.TrimSpace(signature)
	if code == "" || signature == "" {
		logger.WithFields(map[string]interface{}{
			"event_id": eventID,
		}).Warn("ticket validation missing code or signature")
		outcome.ErrorCode = "missing_parameters"
		return outcome, http.StatusBadRequest
	}

	expectedSignature := signCode(rt.VoteSecret, code)
	if !strings.EqualFold(expectedSignature, signature) {
		logger.WithFields(map[string]interface{}{
			"event_id": eventID,
			"code":     code,
		}).Warn("ticket validation signature mismatch")
		outcome.ErrorCode = "invalid_signature"
		return outcome, http.StatusBadRequest
	}

	result, err := rt.db.ValidateTicket(eventID, code)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			logger.WithFields(map[string]interface{}{
				"event_id": eventID,
				"code":     code,
			}).Warn("ticket validation - ticket not found")
			outcome.ErrorCode = "ticket_not_found"
			return outcome, http.StatusNotFound
		}

		logger.WithError(err).Error("ticket validation - database error")
		outcome.ErrorCode = "internal_error"
		return outcome, http.StatusInternalServerError
	}

	if !strings.EqualFold(result.TicketSignature, expectedSignature) {
		logger.WithFields(map[string]interface{}{
			"event_id": eventID,
			"code":     code,
		}).Warn("ticket validation - stored signature mismatch")
		outcome.ErrorCode = "stored_signature_mismatch"
		return outcome, http.StatusBadRequest
	}

	alreadyRedeemed, err := rt.db.RedeemTicket(eventID, code, expectedSignature)
	if err != nil {
		if errors.Is(err, database.ErrTicketSignatureMismatch) {
			logger.WithFields(map[string]interface{}{
				"event_id": eventID,
				"code":     code,
			}).Warn("ticket validation - redemption signature mismatch")
			outcome.ErrorCode = "redemption_signature_mismatch"
			return outcome, http.StatusBadRequest
		}

		logger.WithError(err).Error("ticket validation - cannot update redemption state")
		outcome.ErrorCode = "internal_error"
		return outcome, http.StatusInternalServerError
	}

	outcome.AlreadyRedeemed = alreadyRedeemed
	return outcome, http.StatusOK
}

func ticketValidationHTML(mainText, detail string) string {
	extra := ""
	if strings.TrimSpace(detail) != "" {
		extra = fmt.Sprintf("<p>%s</p>", detail)
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="it">
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>%s</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif;
            background-color: #0b0b0b;
            color: #f5f5f5;
            margin: 0;
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
            text-align: center;
            padding: 2rem;
        }
        main {
            max-width: 480px;
        }
        h1 {
            font-size: 2.5rem;
            margin-bottom: 1rem;
        }
        p {
            font-size: 1.1rem;
            margin: 0;
        }
    </style>
</head>
<body>
    <main>
        <h1>%s</h1>
        %s
    </main>
</body>
</html>`, mainText, mainText, extra)
}

func (rt *_router) ticketValidationStatus(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	query := r.URL.Query()
	eventIDStr := strings.TrimSpace(query.Get("e"))
	code := strings.TrimSpace(query.Get("c"))
	signature := strings.TrimSpace(query.Get("s"))

	eventID, err := strconv.Atoi(eventIDStr)
	if err != nil {
		ctx.Logger.WithField("event_id", eventIDStr).Warn("ticket validation api request with invalid event id")
		writeJSONError(w, http.StatusBadRequest, "invalid_event_id")
		return
	}

	outcome, status := rt.processTicketValidation(ctx.Logger, eventID, code, signature)
	if outcome.ErrorCode != "" {
		writeJSONError(w, status, outcome.ErrorCode)
		return
	}

	resp := struct {
		Valid           bool `json:"valid"`
		AlreadyRedeemed bool `json:"already_redeemed"`
	}{Valid: true, AlreadyRedeemed: outcome.AlreadyRedeemed}

	w.Header().Set("content-type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		ctx.Logger.WithError(err).Error("ticket validation api - cannot encode response")
	}
}
