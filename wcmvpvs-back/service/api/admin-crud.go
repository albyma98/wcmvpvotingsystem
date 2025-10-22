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
	"github.com/julienschmidt/httprouter"
)

// Teams
func (rt *_router) listTeams(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	teams, err := rt.db.ListTeams()
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot list teams")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	_ = json.NewEncoder(w).Encode(teams)
}

func (rt *_router) createTeam(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	var t struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
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
}

func (rt *_router) updateTeam(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	id, _ := strconv.Atoi(ps.ByName("id"))
	var t struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if err := rt.db.UpdateTeam(id, t.Name); err != nil {
		ctx.Logger.WithError(err).Error("cannot update team")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (rt *_router) deleteTeam(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	id, _ := strconv.Atoi(ps.ByName("id"))
	if err := rt.db.DeleteTeam(id); err != nil {
		ctx.Logger.WithError(err).Error("cannot delete team")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Players
func (rt *_router) listPlayers(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	players, err := rt.db.ListPlayers()
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot list players")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	_ = json.NewEncoder(w).Encode(players)
}

func (rt *_router) createPlayer(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	var p database.Player
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
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
}

func (rt *_router) updatePlayer(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	id, _ := strconv.Atoi(ps.ByName("id"))
	var p database.Player
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
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
}

func (rt *_router) deletePlayer(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	id, _ := strconv.Atoi(ps.ByName("id"))
	if err := rt.db.DeletePlayer(id); err != nil {
		ctx.Logger.WithError(err).Error("cannot delete player")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Events
func (rt *_router) listEvents(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	events, err := rt.db.ListEvents()
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot list events")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	_ = json.NewEncoder(w).Encode(events)
}

func (rt *_router) getEvent(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	eventID, err := strconv.Atoi(ps.ByName("id"))
	if err != nil || eventID <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	event, err := rt.db.GetEvent(eventID)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		w.WriteHeader(http.StatusNotFound)
		return
	case err != nil:
		ctx.Logger.WithError(err).Error("cannot load event")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(event)
}

func (rt *_router) getActiveEvent(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	event, err := rt.db.GetActiveEvent()
	switch {
	case errors.Is(err, sql.ErrNoRows):
		w.WriteHeader(http.StatusNotFound)
		return
	case err != nil:
		ctx.Logger.WithError(err).Error("cannot load active event")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(event)
}

func (rt *_router) createEvent(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	var e database.Event
	if err := json.NewDecoder(r.Body).Decode(&e); err != nil {
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
}

func (rt *_router) updateEvent(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	id, _ := strconv.Atoi(ps.ByName("id"))
	var e database.Event
	if err := json.NewDecoder(r.Body).Decode(&e); err != nil {
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
}

func (rt *_router) deleteEvent(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	id, _ := strconv.Atoi(ps.ByName("id"))
	if err := rt.db.DeleteEvent(id); err != nil {
		ctx.Logger.WithError(err).Error("cannot delete event")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (rt *_router) activateEvent(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	id, err := strconv.Atoi(ps.ByName("id"))
	if err != nil || id <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if err := rt.db.SetActiveEvent(id); err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			w.WriteHeader(http.StatusNotFound)
		default:
			ctx.Logger.WithError(err).Error("cannot activate event")
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (rt *_router) deactivateEvents(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	if err := rt.db.ClearActiveEvent(); err != nil {
		ctx.Logger.WithError(err).Error("cannot deactivate events")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (rt *_router) listEventTickets(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	eventID, err := strconv.Atoi(ps.ByName("id"))
	if err != nil || eventID <= 0 {
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
}

// Votes
func (rt *_router) listVotes(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	votes, err := rt.db.ListVotes()
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot list votes")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	_ = json.NewEncoder(w).Encode(votes)
}

func (rt *_router) deleteVote(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	id, _ := strconv.Atoi(ps.ByName("id"))
	if err := rt.db.DeleteVote(id); err != nil {
		ctx.Logger.WithError(err).Error("cannot delete vote")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Admins
func (rt *_router) listAdmins(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
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
}

func (rt *_router) createAdmin(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	var payload struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if payload.Username == "" || payload.Password == "" {
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
}

func (rt *_router) updateAdmin(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	id, _ := strconv.Atoi(ps.ByName("id"))
	var payload struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
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
}

func (rt *_router) deleteAdmin(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	id, _ := strconv.Atoi(ps.ByName("id"))
	if err := rt.db.DeleteAdmin(id); err != nil {
		ctx.Logger.WithError(err).Error("cannot delete admin")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (rt *_router) adminLogin(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	var payload struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if payload.Username == "" || payload.Password == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	admin, err := rt.db.GetAdminByUsername(payload.Username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		ctx.Logger.WithError(err).Error("cannot retrieve admin by username")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if !adminPasswordMatches(admin.PasswordHash, payload.Password) {
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
