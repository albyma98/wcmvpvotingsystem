package api

import (
	"database/sql"
	"errors"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
	"github.com/go-chi/chi/v5"
	"github.com/gofrs/uuid"
)

type fanRegisterPayload struct {
	EventID       int    `json:"event_id"`
	Nickname      string `json:"nickname"`
	Gender        string `json:"gender"`
	Phone         string `json:"phone"`
	AcceptedTerms bool   `json:"accepted_terms"`
	GuestCoins    int    `json:"guest_coins"`
	EnterLottery  bool   `json:"enter_lottery"`
}

func (rt *_router) fanSessionTokenFromRequest(r *http.Request) string {
	token := strings.TrimSpace(r.Header.Get("X-Fan-Session"))
	if token != "" {
		return token
	}
	return strings.TrimSpace(r.URL.Query().Get("fan_session"))
}

func (rt *_router) postRegisterFan(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	if ctx.OrganizationID == 0 {
		_ = writeJSONMessage(w, http.StatusBadRequest, "Organizzazione non valida")
		return
	}
	var payload fanRegisterPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		_ = writeJSONMessage(w, http.StatusBadRequest, "Richiesta non valida")
		return
	}
	deviceID := rt.deviceIDFromRequest(r)
	sessionToken := rt.fanSessionTokenFromRequest(r)
	if sessionToken == "" {
		sessionToken = uuid.Must(uuid.NewV4()).String()
	}

	summary, err := rt.db.RegisterFan(database.FanRegisterInput{
		OrganizationID: ctx.OrganizationID,
		EventID:        payload.EventID,
		DeviceID:       deviceID,
		SessionToken:   sessionToken,
		Nickname:       payload.Nickname,
		Gender:         payload.Gender,
		Phone:          payload.Phone,
		AcceptedTerms:  payload.AcceptedTerms,
		GuestCoins:     payload.GuestCoins,
		EnterLottery:   payload.EnterLottery,
	})
	if err != nil {
		ctx.Logger.WithError(err).Error("register fan failed")
		_ = writeJSONMessage(w, http.StatusBadRequest, "Impossibile salvare il profilo tifoso")
		return
	}

	if err := writeJSON(w, http.StatusOK, map[string]interface{}{
		"session_token": sessionToken,
		"user":          summary.Profile,
		"wallet":        summary.Wallet,
	}); err != nil {
		ctx.Logger.WithError(err).Error("write register fan response")
	}
}

func (rt *_router) getFanMe(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	resp := map[string]interface{}{"registered": false}
	sessionToken := rt.fanSessionTokenFromRequest(r)
	deviceID := rt.deviceIDFromRequest(r)
	eventID := parseIDFromQuery(r, "event_id")

	if sessionToken != "" {
		summary, err := rt.db.GetFanBySessionToken(sessionToken)
		if err == nil {
			resp["registered"] = true
			resp["user"] = summary.Profile
			resp["wallet"] = summary.Wallet
			if eventID > 0 {
				if redemptions, redemptionsErr := rt.db.ListFanRewardRedemptions(eventID, summary.Profile.ID); redemptionsErr == nil {
					resp["reward_redemptions"] = redemptions
				}
				if ticket, ticketErr := rt.db.GetFanLotteryTicket(eventID, summary.Profile.ID); ticketErr == nil {
					resp["lottery_ticket"] = ticket
				}
				if rank, rankErr := rt.db.GetFanRank(eventID, ctx.OrganizationID, summary.Profile.ID); rankErr == nil {
					resp["user_rank"] = map[string]interface{}{"rank": rank.Rank, "coins": rank.Coins}
				}
			}
			_ = writeJSON(w, http.StatusOK, resp)
			return
		}
	}

	if eventID > 0 && deviceID != "" {
		coins, _ := rt.db.GetGuestCoins(eventID, ctx.OrganizationID, deviceID)
		resp["guest_coins"] = coins
	}
	_ = writeJSON(w, http.StatusOK, resp)
}

func (rt *_router) postGuestCoins(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	eventID, err := parseNumericID(chi.URLParam(r, "eventId"))
	if err != nil {
		_ = writeJSONMessage(w, http.StatusBadRequest, "Evento non valido")
		return
	}
	deviceID := rt.deviceIDFromRequest(r)
	if deviceID == "" {
		_ = writeJSONMessage(w, http.StatusBadRequest, "Dispositivo non valido")
		return
	}
	var payload struct{ Coins int `json:"coins"` }
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		_ = writeJSONMessage(w, http.StatusBadRequest, "Richiesta non valida")
		return
	}
	if err := rt.db.UpsertGuestCoins(eventID, ctx.OrganizationID, deviceID, payload.Coins); err != nil {
		ctx.Logger.WithError(err).Error("upsert guest coins")
		_ = writeJSONMessage(w, http.StatusInternalServerError, "Errore salvataggio monete")
		return
	}
	_ = writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true})
}

func (rt *_router) getCoinsLeaderboard(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	eventID, err := parseNumericID(chi.URLParam(r, "eventId"))
	if err != nil {
		_ = writeJSONMessage(w, http.StatusBadRequest, "Evento non valido")
		return
	}
	rows, err := rt.db.GetFanLeaderboard(eventID, ctx.OrganizationID, 3)
	if err != nil {
		ctx.Logger.WithError(err).Error("fan leaderboard")
		_ = writeJSON(w, http.StatusOK, map[string]interface{}{"top3": []interface{}{}})
		return
	}
	top := make([]map[string]interface{}, 0, len(rows))
	for _, e := range rows {
		top = append(top, map[string]interface{}{"name": e.Nickname, "coins": e.Coins})
	}
	resp := map[string]interface{}{"top3": top}
	if token := rt.fanSessionTokenFromRequest(r); token != "" {
		if me, e := rt.db.GetFanBySessionToken(token); e == nil {
			if rank, rankErr := rt.db.GetFanRank(eventID, ctx.OrganizationID, me.Profile.ID); rankErr == nil {
				resp["userRank"] = map[string]interface{}{"rank": rank.Rank, "coins": rank.Coins}
			}
		}
	}
	_ = writeJSON(w, http.StatusOK, resp)
}

func (rt *_router) postFanRewardRedeem(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	eventID, err := parseNumericID(chi.URLParam(r, "eventId"))
	if err != nil {
		_ = writeJSONMessage(w, http.StatusBadRequest, "Evento non valido")
		return
	}
	token := rt.fanSessionTokenFromRequest(r)
	if token == "" {
		_ = writeJSONMessage(w, http.StatusUnauthorized, "Profilo tifoso richiesto")
		return
	}
	me, err := rt.db.GetFanBySessionToken(token)
	if err != nil {
		_ = writeJSONMessage(w, http.StatusUnauthorized, "Profilo tifoso richiesto")
		return
	}
	var payload struct {
		RewardKey string `json:"reward_key"`
		CostCoins int    `json:"cost_coins"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		_ = writeJSONMessage(w, http.StatusBadRequest, "Richiesta non valida")
		return
	}
	if err := rt.db.RecordFanRewardRedemption(eventID, me.Profile.ID, payload.RewardKey, payload.CostCoins); err != nil {
		if err == sql.ErrNoRows {
			_ = writeJSONMessage(w, http.StatusConflict, "Monete insufficienti")
			return
		}
		_ = writeJSONMessage(w, http.StatusInternalServerError, "Errore riscatto")
		return
	}
	updated, _ := rt.db.GetFanBySessionToken(token)
	_ = writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true, "wallet": updated.Wallet})
}

func parseIDFromQuery(r *http.Request, key string) int {
	v := strings.TrimSpace(r.URL.Query().Get(key))
	id, _ := parseNumericID(v)
	return id
}

func parseNumericID(value string) (int, error) {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || parsed <= 0 {
		return 0, errors.New("invalid_id")
	}
	if err != nil {
		return 0, err
	}
	return parsed, nil
}
