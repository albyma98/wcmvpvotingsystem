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
	"github.com/gofrs/uuid"
)

const (
	fanNicknameMinLen = 3
	fanNicknameMaxLen = 24
)

var fanNicknamePattern = regexp.MustCompile(`^[A-Za-z0-9._ -]+$`)

type fanRegisterPayload struct {
	EventID       int    `json:"event_id"`
	Nickname      string `json:"nickname"`
	Gender        string `json:"gender"`
	Phone         string `json:"phone"`
	AcceptedTerms bool   `json:"accepted_terms"`
	GuestCoins    int    `json:"guest_coins"`
	EnterLottery  bool   `json:"enter_lottery"`
}

type fanNicknameUpdatePayload struct {
	Nickname string `json:"nickname"`
}

func validateFanNickname(raw string) (string, string) {
	nickname := strings.TrimSpace(raw)
	if nickname == "" {
		return "", "Il nickname non può essere vuoto."
	}
	if len([]rune(nickname)) < fanNicknameMinLen {
		return "", "Il nickname deve avere almeno 3 caratteri."
	}
	if len([]rune(nickname)) > fanNicknameMaxLen {
		return "", "Il nickname può avere massimo 24 caratteri."
	}
	if !fanNicknamePattern.MatchString(nickname) {
		return "", "Usa solo lettere, numeri, spazi, punto, trattino o underscore."
	}
	return nickname, ""
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
		summary, err := rt.db.GetFanBySessionToken(sessionToken, deviceID)
		if err == nil {
			resp["registered"] = true
			resp["user"] = summary.Profile
			resp["wallet"] = summary.Wallet
			if eventID > 0 {
				if redemptions, redemptionsErr := rt.db.ListFanRewardRedemptions(eventID, summary.Profile.ID); redemptionsErr == nil {
					resp["reward_redemptions"] = redemptions
				}
				if ticket, ticketErr := rt.db.GetFanLotteryTicket(eventID, summary.Profile.ID); ticketErr == nil {
					resp["lottery_ticket"] = rt.serializeLotteryTicket(eventID, ticket)
				} else if deviceID != "" {
					// Fallback: if the vote was cast from this device but user_id linkage is not yet persisted,
					// expose the current device ticket so the QR appears immediately in fan profile.
					if vote, voteErr := rt.db.GetDeviceVote(eventID, deviceID); voteErr == nil {
						resp["lottery_ticket"] = rt.serializeLotteryTicket(eventID, database.EventTicket{
							VoteID:          vote.ID,
							TicketCode:      vote.TicketCode,
							TicketSignature: vote.TicketSignature,
							PlayerID:        vote.PlayerID,
							CreatedAt:       vote.CreatedAt,
						})
					}
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

func (rt *_router) putFanNickname(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	token := rt.fanSessionTokenFromRequest(r)
	if token == "" {
		_ = writeJSONMessage(w, http.StatusUnauthorized, "Profilo tifoso richiesto")
		return
	}

	me, err := rt.db.GetFanBySessionToken(token, rt.deviceIDFromRequest(r))
	if err != nil {
		_ = writeJSONMessage(w, http.StatusUnauthorized, "Profilo tifoso richiesto")
		return
	}
	if me.Profile.OrganizationID != 0 && me.Profile.OrganizationID != ctx.OrganizationID {
		_ = writeJSONMessage(w, http.StatusForbidden, "Profilo non valido per questa organizzazione")
		return
	}

	var payload fanNicknameUpdatePayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		_ = writeJSONMessage(w, http.StatusBadRequest, "Richiesta non valida")
		return
	}

	nickname, validationMessage := validateFanNickname(payload.Nickname)
	if validationMessage != "" {
		_ = writeJSONMessage(w, http.StatusBadRequest, validationMessage)
		return
	}

	isAvailable, err := rt.db.IsFanNicknameAvailable(ctx.OrganizationID, nickname, me.Profile.ID)
	if err != nil {
		ctx.Logger.WithError(err).Error("check nickname availability failed")
		_ = writeJSONMessage(w, http.StatusInternalServerError, "Errore durante il controllo del nickname")
		return
	}
	if !isAvailable {
		_ = writeJSONMessage(w, http.StatusConflict, "Nickname già in uso, scegli un altro nome.")
		return
	}

	profile, err := rt.db.UpdateFanNickname(me.Profile.ID, nickname)
	if err != nil {
		ctx.Logger.WithError(err).Error("update fan nickname failed")
		_ = writeJSONMessage(w, http.StatusInternalServerError, "Impossibile aggiornare il nickname")
		return
	}

	_ = writeJSON(w, http.StatusOK, map[string]interface{}{
		"ok":      true,
		"message": "Nickname aggiornato",
		"user":    profile,
	})
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
	var payload struct {
		Coins int `json:"coins"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		_ = writeJSONMessage(w, http.StatusBadRequest, "Richiesta non valida")
		return
	}

	token := rt.fanSessionTokenFromRequest(r)
	if token != "" {
		me, err := rt.db.GetFanBySessionToken(token, rt.deviceIDFromRequest(r))
		if err == nil {
			if me.Profile.OrganizationID != 0 && me.Profile.OrganizationID != ctx.OrganizationID {
				_ = writeJSONMessage(w, http.StatusForbidden, "Profilo non valido per questa organizzazione")
				return
			}
			if err := rt.db.SetFanWalletCoins(me.Profile.ID, payload.Coins); err != nil {
				ctx.Logger.WithError(err).Error("set fan wallet coins")
				_ = writeJSONMessage(w, http.StatusInternalServerError, "Errore salvataggio monete")
				return
			}
			rt.coinsHub.Broadcast(eventID)
			_ = writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true, "wallet": payload.Coins, "registered": true})
			return
		}
	}

	if err := rt.db.UpsertGuestCoins(eventID, ctx.OrganizationID, deviceID, payload.Coins); err != nil {
		ctx.Logger.WithError(err).Error("upsert guest coins")
		_ = writeJSONMessage(w, http.StatusInternalServerError, "Errore salvataggio monete")
		return
	}
	rt.coinsHub.Broadcast(eventID)
	_ = writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true, "wallet": payload.Coins, "registered": false})
}

func (rt *_router) serializeLotteryTicket(eventID int, ticket database.EventTicket) map[string]interface{} {
	lotteryTicket := map[string]interface{}{
		"vote_id":           ticket.VoteID,
		"ticket_code":       ticket.TicketCode,
		"ticket_signature":  ticket.TicketSignature,
		"player_id":         ticket.PlayerID,
		"player_first_name": ticket.PlayerFirstName,
		"player_last_name":  ticket.PlayerLastName,
		"created_at":        ticket.CreatedAt,
	}
	if validationURL, buildErr := rt.buildTicketValidationURL(eventID, ticket.TicketCode, ticket.TicketSignature); buildErr == nil {
		lotteryTicket["qr_data"] = validationURL
	}
	return lotteryTicket
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
		if me, e := rt.db.GetFanBySessionToken(token, rt.deviceIDFromRequest(r)); e == nil {
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
	me, err := rt.db.GetFanBySessionToken(token, rt.deviceIDFromRequest(r))
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
	updated, _ := rt.db.GetFanBySessionToken(token, rt.deviceIDFromRequest(r))
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
