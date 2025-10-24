package api

import (
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
	"github.com/go-chi/chi/v5"
)

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
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		ctx.Logger.WithError(err).Warn("invalid payload while creating team")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	id, err := rt.db.CreateTeam(t.Name)
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
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		ctx.Logger.WithError(err).Warn("invalid payload while updating team")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if err := rt.db.UpdateTeam(id, t.Name); err != nil {
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
	players, err := rt.db.ListPlayers()
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot list players")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	_ = json.NewEncoder(w).Encode(players)
	ctx.Logger.WithField("players", len(players)).Info("listed players")
}

func (rt *_router) createPlayer(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	var p database.Player
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		ctx.Logger.WithError(err).Warn("invalid payload while creating player")
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
	p.ID = id
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
	if err := rt.db.DeletePlayer(id); err != nil {
		ctx.Logger.WithError(err).Error("cannot delete player")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
	ctx.Logger.WithField("player_id", id).Info("player deleted")
}

// Events
func (rt *_router) listEvents(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	events, err := rt.db.ListEvents()
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot list events")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	_ = json.NewEncoder(w).Encode(events)
	ctx.Logger.WithField("events", len(events)).Info("listed events")
}

func (rt *_router) createEvent(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	var e database.Event
	if err := json.NewDecoder(r.Body).Decode(&e); err != nil {
		ctx.Logger.WithError(err).Warn("invalid payload while creating event")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
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
	var e database.Event
	if err := json.NewDecoder(r.Body).Decode(&e); err != nil {
		ctx.Logger.WithError(err).Warn("invalid payload while updating event")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	e.ID = id
	if err := rt.db.UpdateEvent(e); err != nil {
		ctx.Logger.WithError(err).Error("cannot update event")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
	ctx.Logger.WithField("event_id", id).Info("event updated")
}

func (rt *_router) deleteEvent(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	id, _ := strconv.Atoi(chi.URLParam(r, "id"))
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

	if err := rt.db.SetActiveEvent(id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		ctx.Logger.WithError(err).Error("cannot activate event")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
	ctx.Logger.WithField("event_id", id).Info("event activated")
}

func (rt *_router) deactivateEvents(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	if err := rt.db.ClearActiveEvent(); err != nil {
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

	tickets, err := rt.db.ListEventTickets(eventID)
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot list event tickets")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(tickets)
	ctx.Logger.WithFields(map[string]interface{}{"event_id": eventID, "tickets": len(tickets)}).Info("listed event tickets")
}

// Votes
func (rt *_router) listVotes(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	votes, err := rt.db.ListVotes()
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
	admins, err := rt.db.ListAdmins()
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

	admin := database.Admin{
		Username:     payload.Username,
		PasswordHash: hashAdminPassword(payload.Password),
		Role:         payload.Role,
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

	admin := database.Admin{ID: id, Username: payload.Username, Role: payload.Role}
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
	if err := rt.db.DeleteAdmin(id); err != nil {
		ctx.Logger.WithError(err).Error("cannot delete admin")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
	ctx.Logger.WithField("admin_id", id).Info("admin deleted")
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

	admin, err := rt.db.GetAdminByUsername(payload.Username)
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

	if !adminPasswordMatches(admin.PasswordHash, payload.Password) {
		ctx.Logger.WithField("username", payload.Username).Warn("admin login failed: wrong password")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	token, err := rt.createAdminSession(admin.ID)
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
