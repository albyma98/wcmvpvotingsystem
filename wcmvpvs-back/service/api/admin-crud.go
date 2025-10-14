package api

import (
	"encoding/json"
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
	_ = json.NewEncoder(w).Encode(admins)
}

func (rt *_router) createAdmin(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	var a database.Admin
	if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	id, err := rt.db.CreateAdmin(a)
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
	var a database.Admin
	if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	a.ID = id
	if err := rt.db.UpdateAdmin(a); err != nil {
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
