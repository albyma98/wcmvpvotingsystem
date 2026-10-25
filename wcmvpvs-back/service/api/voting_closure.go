package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
	"github.com/go-chi/chi/v5"
)

func (rt *_router) getVotingSettings(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	if !strings.EqualFold(ctx.AdminRole, "superadmin") {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	settings, err := rt.db.GetVotingSettings(ctx.OrganizationID)
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot load voting settings")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	_ = writeJSON(w, http.StatusOK, settings)
}

func (rt *_router) putVotingSettings(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	if !strings.EqualFold(ctx.AdminRole, "superadmin") {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	reader := http.MaxBytesReader(w, r.Body, 16<<10)
	defer reader.Close()
	var settings database.VotingSettings
	decoder := json.NewDecoder(reader)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&settings); err != nil {
		_ = writeJSONMessage(w, http.StatusBadRequest, "Configurazione non valida.")
		return
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		_ = writeJSONMessage(w, http.StatusBadRequest, "Configurazione non valida.")
		return
	}
	settings, err := rt.db.SaveVotingSettings(ctx.OrganizationID, settings)
	if errors.Is(err, database.ErrInvalidVotingSettings) {
		_ = writeJSONMessage(w, http.StatusBadRequest, "Controlla modalità, Template SID (HX seguito da 32 caratteri esadecimali) e numeri con prefisso internazionale, senza duplicati. Per i destinatari abilitati serve un Template SID.")
		return
	}
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot save voting settings")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	_ = writeJSON(w, http.StatusOK, settings)
}

func (rt *_router) getVotingClosure(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil || id <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if !rt.ensureEventInOrganization(w, ctx, id) {
		return
	}
	summary, notifications, err := rt.db.GetVotingClosure(id, ctx.OrganizationID)
	if errors.Is(err, sql.ErrNoRows) {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot load voting closure")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	for i := range notifications {
		notifications[i].Phone = maskPhone(notifications[i].Phone)
	}
	w.Header().Set("Cache-Control", "no-store")
	_ = writeJSON(w, http.StatusOK, struct {
		Summary       database.VotingClosure        `json:"summary"`
		Notifications []database.VotingNotification `json:"notifications"`
	}{summary, notifications})
}

func (rt *_router) retryVotingNotifications(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil || id <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if !rt.ensureEventInOrganization(w, ctx, id) {
		return
	}
	if err := rt.db.RetryVotingNotifications(id, ctx.OrganizationID); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	rt.wakeVotingNotifications()
	w.WriteHeader(http.StatusAccepted)
}

type votingNotificationWorker struct {
	router    *_router
	wake      chan struct{}
	cancel    context.CancelFunc
	done      chan struct{}
	startOnce sync.Once
}

func newVotingNotificationWorker(rt *_router) *votingNotificationWorker {
	return &votingNotificationWorker{router: rt, wake: make(chan struct{}, 1), done: make(chan struct{})}
}

func (m *votingNotificationWorker) start() {
	m.startOnce.Do(func() {
		ctx, cancel := context.WithCancel(context.Background())
		m.cancel = cancel
		go func() {
			defer close(m.done)
			ticker := time.NewTicker(3 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				default:
				}
				for ctx.Err() == nil {
					processed, err := m.router.processVotingNotification()
					if err != nil {
						m.router.baseLogger.WithError(err).Warn("cannot process voting notification queue")
						break
					}
					if !processed {
						break
					}
				}
				select {
				case <-ctx.Done():
					return
				case <-m.wake:
				case <-ticker.C:
				}
			}
		}()
	})
}

func (rt *_router) wakeVotingNotifications() {
	if rt.votingNotifications != nil {
		select {
		case rt.votingNotifications.wake <- struct{}{}:
		default:
		}
	}
}

func (rt *_router) processVotingNotification() (bool, error) {
	n, err := rt.db.ClaimVotingNotification()
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	status, message := "submitted", ""
	var result twilioMessageResult
	if n.Kind == "winner" && !rt.twilioMessaging.enabledWinnerWhatsAppTemplate() {
		status, message = "skipped", "Template WhatsApp vincitori non configurato."
	} else if !rt.twilioMessaging.enabledWhatsApp() {
		status, message = "skipped", "Servizio WhatsApp Twilio non configurato sul server."
	} else {
		if n.Kind == "winner" {
			result, err = rt.twilioMessaging.SendWhatsAppWinnerTemplate(n.Phone)
		} else {
			result, err = rt.twilioMessaging.SendWhatsAppResultsTemplate(n.Phone, n.TemplateSID, n.Variables)
		}
		if err != nil {
			// A network timeout can occur after Twilio accepted the message. Do
			// not automatically retry an ambiguous send and risk duplicate texts.
			status, message = "unknown", "Esito incerto: verifica il registro Twilio prima di reinviare."
			var providerError twilioError
			if errors.As(err, &providerError) {
				status, message = "failed", "Twilio ha rifiutato l'invio (HTTP "+strconv.Itoa(providerError.StatusCode)+")."
			}
			rt.baseLogger.WithFields(map[string]interface{}{"notification_id": n.ID, "event_id": n.EventID, "kind": n.Kind, "status": status}).Warn("whatsapp voting notification not accepted")
		} else if result.Status == "failed" || result.Status == "undelivered" {
			status, message = "failed", "Twilio ha segnalato un problema nell'invio."
		}
	}
	if err := rt.db.FinishVotingNotification(n.ID, status, result.SID, result.Status, message); err != nil {
		return true, err
	}
	if n.Kind == "winner" {
		if status == "submitted" {
			err = rt.db.MarkPrizeWinnerNotified(n.EventID, n.PrizeID, result.SID)
		} else if status == "failed" || status == "unknown" {
			err = rt.db.MarkPrizeWinnerNotifyFailed(n.EventID, n.PrizeID)
		}
		if err != nil {
			return true, err
		}
	}
	return true, nil
}
