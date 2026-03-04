package api

import (
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
	"github.com/go-chi/chi/v5"
)

func resolveRosterSchemaValue(value int) int {
	switch value {
	case 12, 13, 14:
		return value
	default:
		return 13
	}
}

// Teams
func (rt *_router) listTeams(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	teams, err := rt.db.ListTeams()
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot list teams")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	_ = json.NewEncoder(w).Encode(teams)
	ctx.Logger.WithField("teams", len(teams)).Info("listed teams")
}

func (rt *_router) createTeam(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	var t struct {
		Name         string `json:"name"`
		Championship string `json:"championship"`
	}
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		ctx.Logger.WithError(err).Warn("invalid payload while creating team")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	id, err := rt.db.CreateTeam(t.Name, t.Championship)
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot create team")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	_ = json.NewEncoder(w).Encode(struct {
		ID int `json:"id"`
	}{ID: id})
	ctx.Logger.WithFields(map[string]interface{}{"team_id": id, "name": t.Name}).Info("team created")
}

func (rt *_router) updateTeam(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	id, _ := strconv.Atoi(chi.URLParam(r, "id"))
	var t struct {
		Name         string `json:"name"`
		Championship string `json:"championship"`
	}
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		ctx.Logger.WithError(err).Warn("invalid payload while updating team")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if err := rt.db.UpdateTeam(id, t.Name, t.Championship); err != nil {
		ctx.Logger.WithError(err).Error("cannot update team")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
	ctx.Logger.WithFields(map[string]interface{}{"team_id": id}).Info("team updated")
}

func (rt *_router) deleteTeam(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	id, _ := strconv.Atoi(chi.URLParam(r, "id"))
	if err := rt.db.DeleteTeam(id); err != nil {
		ctx.Logger.WithError(err).Error("cannot delete team")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
	ctx.Logger.WithField("team_id", id).Info("team deleted")
}

// Players
func (rt *_router) listPlayers(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	var (
		players []database.Player
		err     error
	)

	if ctx.OrganizationID > 0 {
		players, err = rt.db.ListPlayersByOrganization(ctx.OrganizationID)
	} else {
		players, err = rt.db.ListPlayers()
	}
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot list players")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	rosterSchema := resolveRosterSchemaValue(13)
	if ctx.OrganizationID > 0 {
		if schema, schemaErr := rt.db.GetOrganizationRosterSchema(ctx.OrganizationID); schemaErr == nil {
			rosterSchema = resolveRosterSchemaValue(schema)
		}
	}

	response := struct {
		Players      []database.Player `json:"players"`
		RosterSchema int               `json:"roster_schema"`
	}{Players: players, RosterSchema: rosterSchema}

	_ = json.NewEncoder(w).Encode(response)
	ctx.Logger.WithFields(map[string]interface{}{"players": len(players), "roster_schema": rosterSchema}).Info("listed players")
}

func (rt *_router) listPublicPlayers(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	if ctx.OrganizationID == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	players, err := rt.db.ListPlayersByOrganization(ctx.OrganizationID)
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot list public players")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	filtered := make([]database.Player, 0, len(players))
	for _, player := range players {
		if player.IsCalledUp {
			filtered = append(filtered, player)
		}
	}

	rosterSchema := resolveRosterSchemaValue(13)
	if schema, schemaErr := rt.db.GetOrganizationRosterSchema(ctx.OrganizationID); schemaErr == nil {
		rosterSchema = resolveRosterSchemaValue(schema)
	}

	response := struct {
		Players      []database.Player `json:"players"`
		RosterSchema int               `json:"roster_schema"`
	}{Players: filtered, RosterSchema: rosterSchema}

	_ = json.NewEncoder(w).Encode(response)
	ctx.Logger.WithFields(map[string]interface{}{"players": len(filtered), "roster_schema": rosterSchema}).Info("listed public players")
}

func (rt *_router) updatePlayerSettings(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	if ctx.OrganizationID == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var payload struct {
		RosterSchema int `json:"roster_schema"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		ctx.Logger.WithError(err).Warn("invalid payload while updating player settings")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	schema := resolveRosterSchemaValue(payload.RosterSchema)
	if err := rt.db.UpdateOrganizationRosterSchema(ctx.OrganizationID, schema); err != nil {
		ctx.Logger.WithError(err).Error("cannot update player settings")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
	ctx.Logger.WithFields(map[string]interface{}{"roster_schema": schema}).Info("player settings updated")
}

func (rt *_router) createPlayer(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	var p database.Player
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		ctx.Logger.WithError(err).Warn("invalid payload while creating player")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	p.OrganizationID = ctx.OrganizationID
	if p.OrganizationID == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	id, err := rt.db.CreatePlayer(p)
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot create player")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	_ = json.NewEncoder(w).Encode(struct {
		ID int `json:"id"`
	}{ID: id})
	ctx.Logger.WithFields(map[string]interface{}{"player_id": id, "team_id": p.TeamID}).Info("player created")
}

func (rt *_router) updatePlayer(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	id, _ := strconv.Atoi(chi.URLParam(r, "id"))
	var p database.Player
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		ctx.Logger.WithError(err).Warn("invalid payload while updating player")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	existing, err := rt.db.GetPlayerByID(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		ctx.Logger.WithError(err).Error("cannot load player")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if ctx.OrganizationID == 0 || existing.OrganizationID != ctx.OrganizationID {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	p.ID = id
	p.OrganizationID = ctx.OrganizationID
	if err := rt.db.UpdatePlayer(p); err != nil {
		ctx.Logger.WithError(err).Error("cannot update player")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
	ctx.Logger.WithFields(map[string]interface{}{"player_id": id}).Info("player updated")
}

func (rt *_router) deletePlayer(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	id, _ := strconv.Atoi(chi.URLParam(r, "id"))
	player, err := rt.db.GetPlayerByID(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		ctx.Logger.WithError(err).Error("cannot load player before delete")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if ctx.OrganizationID == 0 || player.OrganizationID != ctx.OrganizationID {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if err := rt.db.DeletePlayer(id); err != nil {
		ctx.Logger.WithError(err).Error("cannot delete player")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
	ctx.Logger.WithField("player_id", id).Info("player deleted")
}

// Events
func (rt *_router) ensureEventInOrganization(w http.ResponseWriter, ctx reqcontext.RequestContext, eventID int) bool {
	if ctx.OrganizationID == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return false
	}

	organizationID, err := rt.db.GetEventOrganizationID(eventID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			w.WriteHeader(http.StatusNotFound)
			return false
		}
		ctx.Logger.WithError(err).Error("cannot verify event organization")
		w.WriteHeader(http.StatusInternalServerError)
		return false
	}

	if organizationID != ctx.OrganizationID {
		w.WriteHeader(http.StatusNotFound)
		return false
	}

	return true
}

func (rt *_router) listEvents(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	if ctx.OrganizationID == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	events, err := rt.db.ListEventsByOrganization(ctx.OrganizationID)
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot list events")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	_ = json.NewEncoder(w).Encode(events)
	ctx.Logger.WithField("events", len(events)).Info("listed events")
}

func (rt *_router) createEvent(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	reader := http.MaxBytesReader(w, r.Body, 1<<16)
	defer reader.Close()

	body, err := io.ReadAll(reader)
	if err != nil {
		ctx.Logger.WithError(err).Warn("invalid payload while creating event")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var e database.Event
	if err := json.Unmarshal(body, &e); err != nil {
		ctx.Logger.WithError(err).Warn("invalid payload while creating event")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err == nil {
		applyEventPostVoteDefaults(&e, raw)
	} else {
		applyEventPostVoteDefaults(&e, nil)
	}

	if ctx.OrganizationID == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	e.OrganizationID = ctx.OrganizationID

	id, err := rt.db.CreateEvent(e)
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot create event")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	_ = json.NewEncoder(w).Encode(struct {
		ID int `json:"id"`
	}{ID: id})
	ctx.Logger.WithFields(map[string]interface{}{"event_id": id}).Info("event created")
}

func (rt *_router) updateEvent(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	id, _ := strconv.Atoi(chi.URLParam(r, "id"))
	reader := http.MaxBytesReader(w, r.Body, 1<<16)
	defer reader.Close()

	body, err := io.ReadAll(reader)
	if err != nil {
		ctx.Logger.WithError(err).Warn("invalid payload while updating event")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var e database.Event
	if err := json.Unmarshal(body, &e); err != nil {
		ctx.Logger.WithError(err).Warn("invalid payload while updating event")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err == nil {
		applyEventPostVoteDefaults(&e, raw)
	} else {
		applyEventPostVoteDefaults(&e, nil)
	}
	e.ID = id
	e.OrganizationID = ctx.OrganizationID
	if ctx.OrganizationID == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if !rt.ensureEventInOrganization(w, ctx, id) {
		return
	}

	if err := rt.db.UpdateEvent(e); err != nil {
		if errors.Is(err, database.ErrPrizeLockedByWinner) {
			w.WriteHeader(http.StatusConflict)
			return
		}
		ctx.Logger.WithError(err).Error("cannot update event")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
	ctx.Logger.WithField("event_id", id).Info("event updated")
}

func (rt *_router) deleteEvent(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	id, _ := strconv.Atoi(chi.URLParam(r, "id"))
	if !rt.ensureEventInOrganization(w, ctx, id) {
		return
	}
	if err := rt.db.DeleteEvent(id); err != nil {
		ctx.Logger.WithError(err).Error("cannot delete event")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
	ctx.Logger.WithField("event_id", id).Info("event deleted")
}

func (rt *_router) activateEvent(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil || id <= 0 {
		ctx.Logger.Warn("invalid event id while activating event")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if !rt.ensureEventInOrganization(w, ctx, id) {
		return
	}

	if err := rt.db.SetActiveEvent(id, ctx.OrganizationID); err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			w.WriteHeader(http.StatusNotFound)
		case errors.Is(err, database.ErrEventAlreadyConcluded):
			w.WriteHeader(http.StatusConflict)
		default:
			ctx.Logger.WithError(err).Error("cannot activate event")
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
	ctx.Logger.WithField("event_id", id).Info("event activated")
}

func (rt *_router) closeEventVoting(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil || id <= 0 {
		ctx.Logger.Warn("invalid event id while closing votes")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if !rt.ensureEventInOrganization(w, ctx, id) {
		return
	}

	if err := rt.db.CloseEventVoting(id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		ctx.Logger.WithError(err).Error("cannot close voting for event")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
	ctx.Logger.WithField("event_id", id).Info("event voting closed")
}

func (rt *_router) concludeEvent(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil || id <= 0 {
		ctx.Logger.Warn("invalid event id while concluding event")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if !rt.ensureEventInOrganization(w, ctx, id) {
		return
	}

	if err := rt.db.ConcludeEvent(id); err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			w.WriteHeader(http.StatusNotFound)
		case errors.Is(err, database.ErrEventAlreadyConcluded):
			w.WriteHeader(http.StatusConflict)
		default:
			ctx.Logger.WithError(err).Error("cannot conclude event")
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
	ctx.Logger.WithField("event_id", id).Info("event concluded")
}

func (rt *_router) deactivateEvents(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	if ctx.OrganizationID == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if err := rt.db.ClearActiveEvent(ctx.OrganizationID); err != nil {
		ctx.Logger.WithError(err).Error("cannot deactivate events")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
	ctx.Logger.Info("all events deactivated")
}

func (rt *_router) listEventTickets(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	eventID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil || eventID <= 0 {
		ctx.Logger.Warn("invalid event id while listing tickets")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if !rt.ensureEventInOrganization(w, ctx, eventID) {
		return
	}

	tickets, err := rt.db.ListEventTickets(eventID)
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot list event tickets")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	totalCodes, err := rt.db.CountEventTickets(eventID)
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot count event tickets")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("X-Lottery-Allowed-Codes", strconv.Itoa(len(tickets)))
	w.Header().Set("X-Lottery-Total-Codes", strconv.Itoa(totalCodes))

	_ = json.NewEncoder(w).Encode(tickets)
	ctx.Logger.WithFields(map[string]interface{}{"event_id": eventID, "allowed_tickets": len(tickets), "total_tickets": totalCodes}).Info("listed event tickets")
}

func (rt *_router) assignPrizeWinner(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	eventID, err := strconv.Atoi(chi.URLParam(r, "eventId"))
	if err != nil || eventID <= 0 {
		ctx.Logger.Warn("invalid event id while assigning prize winner")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if !rt.ensureEventInOrganization(w, ctx, eventID) {
		return
	}
	prizeID, err := strconv.Atoi(chi.URLParam(r, "prizeId"))
	if err != nil || prizeID <= 0 {
		ctx.Logger.Warn("invalid prize id while assigning prize winner")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var payload struct {
		VoteID int `json:"vote_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		ctx.Logger.WithError(err).Warn("invalid payload while assigning prize winner")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if payload.VoteID <= 0 {
		ctx.Logger.Warn("invalid vote id provided while assigning prize winner")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	prize, err := rt.db.AssignPrizeWinner(eventID, prizeID, payload.VoteID)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			w.WriteHeader(http.StatusNotFound)
		case errors.Is(err, database.ErrPrizeAlreadyAssigned), errors.Is(err, database.ErrPrizeWinnerConflict):
			w.WriteHeader(http.StatusConflict)
		case errors.Is(err, database.ErrPrizeVoteMismatch):
			w.WriteHeader(http.StatusBadRequest)
		default:
			ctx.Logger.WithError(err).Error("cannot assign prize winner")
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}

	if prize.Winner != nil && strings.TrimSpace(prize.Winner.Phone) != "" && strings.TrimSpace(prize.Winner.NotifiedAt) == "" {
		message := buildPrizeWinnerSMSMessage(eventID, prize)
		res, sendErr := rt.twilioMessaging.SendSMS(prize.Winner.Phone, message)
		if sendErr != nil {
			_ = rt.db.MarkPrizeWinnerNotifyFailed(eventID, prizeID)
			ctx.Logger.WithError(sendErr).WithFields(map[string]interface{}{"event_id": eventID, "prize_id": prizeID, "winner_phone": maskPhone(prize.Winner.Phone)}).Warn("winner sms send failed")
		} else {
			if err := rt.db.MarkPrizeWinnerNotified(eventID, prizeID, res.SID); err != nil && !errors.Is(err, sql.ErrNoRows) {
				ctx.Logger.WithError(err).WithFields(map[string]interface{}{"event_id": eventID, "prize_id": prizeID}).Warn("cannot mark winner as notified")
			}
		}
	}

	updatedPrize, loadErr := rt.db.ListEventPrizes(eventID)
	if loadErr == nil {
		for _, p := range updatedPrize {
			if p.ID == prizeID {
				prize = p
				break
			}
		}
	}

	_ = json.NewEncoder(w).Encode(prize)
	ctx.Logger.WithFields(map[string]interface{}{"event_id": eventID, "prize_id": prizeID, "vote_id": payload.VoteID}).Info("prize winner assigned")
}

func buildPrizeWinnerSMSMessage(eventID int, prize database.EventPrize) string {
	template := strings.TrimSpace(prize.WinSMSText)
	if template == "" {
		template = winnerExtractedSMSMessage
	}
	replacer := strings.NewReplacer(
		"{NICKNAME}", strings.TrimSpace(prize.Winner.Nickname),
		"{PREMIO}", strings.TrimSpace(prize.Name),
		"{EVENTO}", fmt.Sprintf("Evento #%d", eventID),
		"{CODICE}", strings.TrimSpace(prize.Winner.TicketCode),
		"{ISTRUZIONI_RITIRO}", "Ritirare il premio allo speaker.",
	)
	return strings.TrimSpace(replacer.Replace(template))
}

func (rt *_router) clearPrizeWinner(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	eventID, err := strconv.Atoi(chi.URLParam(r, "eventId"))
	if err != nil || eventID <= 0 {
		ctx.Logger.Warn("invalid event id while clearing prize winner")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if !rt.ensureEventInOrganization(w, ctx, eventID) {
		return
	}
	prizeID, err := strconv.Atoi(chi.URLParam(r, "prizeId"))
	if err != nil || prizeID <= 0 {
		ctx.Logger.Warn("invalid prize id while clearing prize winner")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := rt.db.ClearPrizeWinner(eventID, prizeID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		ctx.Logger.WithError(err).Error("cannot clear prize winner")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
	ctx.Logger.WithFields(map[string]interface{}{"event_id": eventID, "prize_id": prizeID}).Info("prize winner cleared")
}

// Votes
func (rt *_router) listVotes(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	var (
		votes []database.Vote
		err   error
	)
	if ctx.OrganizationID > 0 {
		votes, err = rt.db.ListVotesByOrganization(ctx.OrganizationID)
	} else {
		votes, err = rt.db.ListVotes()
	}
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot list votes")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	_ = json.NewEncoder(w).Encode(votes)
	ctx.Logger.WithField("votes", len(votes)).Info("listed votes")
}

func (rt *_router) deleteVote(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	id, _ := strconv.Atoi(chi.URLParam(r, "id"))
	eventID, err := rt.db.GetVoteEventID(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		ctx.Logger.WithError(err).Error("cannot load vote event")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if !rt.ensureEventInOrganization(w, ctx, eventID) {
		return
	}
	if err := rt.db.DeleteVote(id); err != nil {
		ctx.Logger.WithError(err).Error("cannot delete vote")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
	ctx.Logger.WithField("vote_id", id).Info("vote deleted")
}

// Admins
func (rt *_router) listAdmins(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	admins, err := rt.db.ListAdmins(ctx.OrganizationID)
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot list admins")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	type adminResponse struct {
		ID        int    `json:"id"`
		Username  string `json:"username"`
		Role      string `json:"role"`
		CreatedAt string `json:"created_at"`
	}
	resp := make([]adminResponse, 0, len(admins))
	for _, admin := range admins {
		resp = append(resp, adminResponse{
			ID:        admin.ID,
			Username:  admin.Username,
			Role:      admin.Role,
			CreatedAt: admin.CreatedAt,
		})
	}
	_ = json.NewEncoder(w).Encode(resp)
	ctx.Logger.WithField("admins", len(resp)).Info("listed admins")
}

func (rt *_router) createAdmin(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	var payload struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		ctx.Logger.WithError(err).Warn("invalid payload while creating admin")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if payload.Username == "" || payload.Password == "" {
		ctx.Logger.Warn("missing username or password while creating admin")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	orgID := ctx.OrganizationID
	if orgID == 0 && !strings.EqualFold(payload.Role, "superadmin") {
		ctx.Logger.Warn("organization context required for non-superadmin creation")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if strings.EqualFold(payload.Role, "superadmin") {
		orgID = 0
	}

	admin := database.Admin{
		Username:       payload.Username,
		PasswordHash:   hashAdminPassword(payload.Password),
		Role:           payload.Role,
		OrganizationID: orgID,
	}

	id, err := rt.db.CreateAdmin(admin)
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot create admin")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	_ = json.NewEncoder(w).Encode(struct {
		ID int `json:"id"`
	}{ID: id})
	ctx.Logger.WithFields(map[string]interface{}{"admin_id": id, "username": admin.Username}).Info("admin created")
}

func (rt *_router) updateAdmin(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	id, _ := strconv.Atoi(chi.URLParam(r, "id"))
	var payload struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		ctx.Logger.WithError(err).Warn("invalid payload while updating admin")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	target, err := rt.db.GetAdminByID(id)
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot retrieve admin for update")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if !strings.EqualFold(ctx.AdminRole, "superadmin") {
		if target.OrganizationID == 0 || target.OrganizationID != ctx.OrganizationID {
			w.WriteHeader(http.StatusForbidden)
			return
		}
	}

	admin := database.Admin{ID: id, Username: payload.Username, Role: payload.Role, OrganizationID: target.OrganizationID}
	if payload.Password != "" {
		admin.PasswordHash = hashAdminPassword(payload.Password)
	}

	if err := rt.db.UpdateAdmin(admin); err != nil {
		ctx.Logger.WithError(err).Error("cannot update admin")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
	ctx.Logger.WithFields(map[string]interface{}{"admin_id": id, "username": admin.Username}).Info("admin updated")
}

func (rt *_router) deleteAdmin(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	id, _ := strconv.Atoi(chi.URLParam(r, "id"))
	admin, err := rt.db.GetAdminByID(id)
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot load admin for deletion")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if !strings.EqualFold(ctx.AdminRole, "superadmin") {
		if admin.OrganizationID == 0 || admin.OrganizationID != ctx.OrganizationID {
			w.WriteHeader(http.StatusForbidden)
			return
		}
	}

	if err := rt.db.DeleteAdmin(id); err != nil {
		ctx.Logger.WithError(err).Error("cannot delete admin")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
	ctx.Logger.WithField("admin_id", id).Info("admin deleted")
}

// Partner accounts
func (rt *_router) listPartners(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	partners, err := rt.db.ListPartners(ctx.OrganizationID)
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot list partners")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	type partnerResponse struct {
		ID        int    `json:"id"`
		Username  string `json:"username"`
		CreatedAt string `json:"created_at"`
	}

	resp := make([]partnerResponse, 0, len(partners))
	for _, p := range partners {
		resp = append(resp, partnerResponse{ID: p.ID, Username: p.Username, CreatedAt: p.CreatedAt})
	}

	_ = json.NewEncoder(w).Encode(resp)
}

func (rt *_router) createPartner(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	var payload struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if payload.Username == "" || payload.Password == "" || ctx.OrganizationID == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	admin := database.Admin{Username: payload.Username, PasswordHash: hashAdminPassword(payload.Password), Role: "partner", OrganizationID: ctx.OrganizationID}
	id, err := rt.db.CreateAdmin(admin)
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot create partner")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(struct {
		ID int `json:"id"`
	}{ID: id})
}

func (rt *_router) updatePartner(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	id, _ := strconv.Atoi(chi.URLParam(r, "id"))
	var payload struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	partner, err := rt.db.GetAdminByID(id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if !strings.EqualFold(partner.Role, "partner") || partner.OrganizationID != ctx.OrganizationID {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	if payload.Password == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	updated := database.Admin{ID: id, Username: partner.Username, Role: partner.Role, OrganizationID: partner.OrganizationID, PasswordHash: hashAdminPassword(payload.Password)}
	if err := rt.db.UpdateAdmin(updated); err != nil {
		ctx.Logger.WithError(err).Error("cannot update partner")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (rt *_router) deletePartner(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	id, _ := strconv.Atoi(chi.URLParam(r, "id"))
	partner, err := rt.db.GetAdminByID(id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if !strings.EqualFold(partner.Role, "partner") || partner.OrganizationID != ctx.OrganizationID {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	if err := rt.db.DeleteAdmin(id); err != nil {
		ctx.Logger.WithError(err).Error("cannot delete partner")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Sponsors
func (rt *_router) listAllSponsors(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	sponsors, err := rt.db.ListSponsors(ctx.OrganizationID)
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot list sponsors")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("content-type", "application/json")
	_ = json.NewEncoder(w).Encode(sponsors)
	ctx.Logger.WithField("sponsors", len(sponsors)).Info("listed sponsors")
}

func (rt *_router) createSponsor(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	var payload struct {
		Name       string `json:"name"`
		ReportName string `json:"report_name"`
		LogoData   string `json:"logo_data"`
		LinkURL    string `json:"link_url"`
		Position   int    `json:"position"`
		IsActive   bool   `json:"is_active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		ctx.Logger.WithError(err).Warn("invalid payload while creating sponsor")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if ctx.OrganizationID == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	sponsor := database.Sponsor{
		OrganizationID: ctx.OrganizationID,
		Name:           strings.TrimSpace(payload.Name),
		ReportName:     strings.TrimSpace(payload.ReportName),
		LogoData:       payload.LogoData,
		LinkURL:        strings.TrimSpace(payload.LinkURL),
		Position:       payload.Position,
		IsActive:       payload.IsActive,
	}

	id, err := rt.db.CreateSponsor(sponsor)
	if err != nil {
		switch {
		case errors.Is(err, database.ErrInvalidSponsorData), errors.Is(err, database.ErrInvalidSponsorPos), errors.Is(err, database.ErrMaxSponsors):
			w.WriteHeader(http.StatusBadRequest)
		default:
			ctx.Logger.WithError(err).Error("cannot create sponsor")
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("content-type", "application/json")
	_ = json.NewEncoder(w).Encode(struct {
		ID int `json:"id"`
	}{ID: id})
	ctx.Logger.WithField("sponsor_id", id).Info("sponsor created")
}

func (rt *_router) updateSponsor(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil || id <= 0 {
		ctx.Logger.Warn("invalid sponsor id while updating")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var payload struct {
		Name       string `json:"name"`
		ReportName string `json:"report_name"`
		LogoData   string `json:"logo_data"`
		LinkURL    string `json:"link_url"`
		Position   int    `json:"position"`
		IsActive   bool   `json:"is_active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		ctx.Logger.WithError(err).Warn("invalid payload while updating sponsor")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	sponsor := database.Sponsor{
		ID:             id,
		OrganizationID: ctx.OrganizationID,
		Name:           strings.TrimSpace(payload.Name),
		ReportName:     strings.TrimSpace(payload.ReportName),
		LogoData:       payload.LogoData,
		LinkURL:        strings.TrimSpace(payload.LinkURL),
		Position:       payload.Position,
		IsActive:       payload.IsActive,
	}

	if err := rt.db.UpdateSponsor(sponsor); err != nil {
		switch {
		case errors.Is(err, database.ErrInvalidSponsorData), errors.Is(err, database.ErrInvalidSponsorPos):
			w.WriteHeader(http.StatusBadRequest)
		case errors.Is(err, sql.ErrNoRows):
			w.WriteHeader(http.StatusNotFound)
		default:
			ctx.Logger.WithError(err).Error("cannot update sponsor")
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
	ctx.Logger.WithField("sponsor_id", id).Info("sponsor updated")
}

func (rt *_router) deleteSponsor(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil || id <= 0 {
		ctx.Logger.Warn("invalid sponsor id while deleting")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := rt.db.DeleteSponsor(id, ctx.OrganizationID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		ctx.Logger.WithError(err).Error("cannot delete sponsor")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
	ctx.Logger.WithField("sponsor_id", id).Info("sponsor deleted")
}

func (rt *_router) adminLogin(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	var payload struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		ctx.Logger.WithError(err).Warn("invalid payload while logging admin in")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if payload.Username == "" || payload.Password == "" {
		ctx.Logger.Warn("missing credentials while logging admin in")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	orgID := ctx.OrganizationID
	admin, err := rt.db.GetAdminByUsername(payload.Username, orgID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) && orgID > 0 {
			admin, err = rt.db.GetAdminByUsername(payload.Username, 0)
		}
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				ctx.Logger.WithField("username", payload.Username).Warn("admin login failed: user not found")
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			ctx.Logger.WithError(err).Error("cannot retrieve admin by username")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	if !adminPasswordMatches(admin.PasswordHash, payload.Password) {
		ctx.Logger.WithField("username", payload.Username).Warn("admin login failed: wrong password")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	orgSlug := ctx.OrganizationSlug
	orgTeamID := ctx.OrganizationTeamID
	if !strings.EqualFold(admin.Role, "superadmin") {
		if orgID == 0 || orgSlug == "" || admin.OrganizationID != orgID {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
	} else {
		orgID = 0
		orgSlug = ""
		orgTeamID = 0
	}

	token, err := rt.createAdminSession(admin.ID, admin.Username, admin.Role, orgID, orgTeamID, orgSlug)
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot create admin session")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(struct {
		Token    string `json:"token"`
		Username string `json:"username"`
		Role     string `json:"role"`
	}{Token: token, Username: admin.Username, Role: admin.Role})
	ctx.Logger.WithField("username", admin.Username).Info("admin logged in")
}

func (rt *_router) partnerLogin(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	var payload struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		ctx.Logger.WithError(err).Warn("invalid payload while logging partner in")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if payload.Username == "" || payload.Password == "" || ctx.OrganizationID == 0 || ctx.OrganizationSlug == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	admin, err := rt.db.GetAdminByUsername(payload.Username, ctx.OrganizationID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		ctx.Logger.WithError(err).Error("cannot retrieve partner credentials")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if !strings.EqualFold(admin.Role, "partner") {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	if !adminPasswordMatches(admin.PasswordHash, payload.Password) {
		ctx.Logger.WithField("username", payload.Username).Warn("partner login failed: wrong password")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	token, err := rt.createPartnerSession(admin.ID, admin.Username, ctx.OrganizationID, ctx.OrganizationSlug)
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot create partner session")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(struct {
		Token    string `json:"token"`
		Username string `json:"username"`
	}{Token: token, Username: admin.Username})
	ctx.Logger.WithField("username", admin.Username).Info("partner logged in")
}

func hashAdminPassword(password string) string {
	sum := sha256.Sum256([]byte(password))
	return hex.EncodeToString(sum[:])
}

func adminPasswordMatches(hash, password string) bool {
	if hash == "" {
		return false
	}
	candidate := hashAdminPassword(password)
	if len(candidate) != len(hash) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(hash), []byte(candidate)) == 1
}

func applyEventPostVoteDefaults(e *database.Event, raw map[string]json.RawMessage) {
	if e == nil {
		return
	}

	if !eventFlagProvided(raw, "show_reaction_test", "showReactionTest") {
		if !e.ShowReactionTest {
			e.ShowReactionTest = true
		}
	}

	if !eventFlagProvided(raw, "show_selfie", "showSelfie") {
		if !e.ShowSelfie {
			e.ShowSelfie = true
		}
	}

	if !eventFlagProvided(raw, "show_vote_trend", "showVoteTrend") {
		if !e.ShowVoteTrend {
			e.ShowVoteTrend = true
		}
	}

	if !eventFlagProvided(raw, "show_feedback_survey", "showFeedbackSurvey") {
		if !e.ShowFeedbackSurvey {
			e.ShowFeedbackSurvey = true
		}
	}

	if !eventFlagProvided(raw, "show_pre_vote_sponsors", "showPreVoteSponsors") {
		if !e.ShowPreVoteSponsors {
			e.ShowPreVoteSponsors = true
		}
	}

	if !eventFlagProvided(raw, "show_pre_vote_bottom_sponsors", "showPreVoteBottomSponsors", "show_pre_vote_sponsor_wall") {
		if !e.ShowPreVoteBottomSponsors {
			e.ShowPreVoteBottomSponsors = true
		}
	}

	if !eventFlagProvided(raw, "show_vote_counter", "showVoteCounter", "show_pre_vote_vote_counter") {
		if !e.ShowVoteCounter {
			e.ShowVoteCounter = true
		}
	}
}

func eventFlagProvided(raw map[string]json.RawMessage, keys ...string) bool {
	if raw == nil {
		return false
	}
	for _, key := range keys {
		if _, ok := raw[key]; ok {
			return true
		}
	}
	return false
}
