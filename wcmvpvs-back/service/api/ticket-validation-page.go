package api

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

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

	expectedSignature := signCode(rt.VoteSecret, code)
	if !strings.EqualFold(expectedSignature, signature) {
		logger.WithFields(map[string]interface{}{
			"event_id": eventID,
			"code":     code,
		}).Warn("ticket validation signature mismatch")
		renderInvalid(http.StatusBadRequest)
		return
	}

	result, err := rt.db.ValidateTicket(eventID, code)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			logger.WithFields(map[string]interface{}{
				"event_id": eventID,
				"code":     code,
			}).Warn("ticket validation - ticket not found")
			renderInvalid(http.StatusBadRequest)
			return
		}

		logger.WithError(err).Error("ticket validation - database error")
		renderInvalid(http.StatusInternalServerError)
		return
	}

	if !strings.EqualFold(result.TicketSignature, expectedSignature) {
		logger.WithFields(map[string]interface{}{
			"event_id": eventID,
			"code":     code,
		}).Warn("ticket validation - stored signature mismatch")
		renderInvalid(http.StatusBadRequest)
		return
	}

	alreadyRedeemed, err := rt.db.RedeemTicket(eventID, code, expectedSignature)
	if err != nil {
		if errors.Is(err, database.ErrTicketSignatureMismatch) {
			logger.WithFields(map[string]interface{}{
				"event_id": eventID,
				"code":     code,
			}).Warn("ticket validation - redemption signature mismatch")
			renderInvalid(http.StatusBadRequest)
			return
		}

		logger.WithError(err).Error("ticket validation - cannot update redemption state")
		renderInvalid(http.StatusInternalServerError)
		return
	}

	message := "✅ QR VALIDO"
	detail := ""
	if alreadyRedeemed {
		detail = "Questo ticket risulta già riscattato."
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprint(w, ticketValidationHTML(message, detail))

	logger.WithFields(map[string]interface{}{
		"event_id":        eventID,
		"code":            code,
		"alreadyRedeemed": alreadyRedeemed,
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

	parsed.Path = strings.TrimSuffix(parsed.Path, "/") + "/t"
	q := parsed.Query()
	q.Set("e", strconv.Itoa(eventID))
	q.Set("c", code)
	q.Set("s", signature)
	parsed.RawQuery = q.Encode()

	return parsed.String(), nil
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
