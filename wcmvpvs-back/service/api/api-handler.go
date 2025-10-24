package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Handler returns an instance of chi.Router that handles APIs registered here
func (rt *_router) Handler() chi.Router {
	// Register routes
	rt.router.Get("/", rt.getHelloWorld)
	rt.router.Get("/context", rt.wrap(rt.getContextReply))

	// Special routes
	rt.router.Get("/liveness", rt.liveness)

	voteHandler := rt.wrap(rt.postVote)
	rt.router.Post("/vote", func(w http.ResponseWriter, r *http.Request) {
		rt.antiDoubleVoteMiddleware(voteHandler).ServeHTTP(w, r)
	})
	rt.router.Get("/sponsors", rt.wrap(rt.listSponsors))
	// Admin CRUD routes
	rt.router.Post("/admin/login", rt.wrap(rt.adminLogin))
	rt.router.Get("/admin/sponsors", rt.wrapAdmin(rt.listSponsors))
	rt.router.Put("/sponsors/{slot}", rt.wrapAdmin(rt.upsertSponsor))
	rt.router.Delete("/sponsors/{slot}", rt.wrapAdmin(rt.deleteSponsor))

	rt.router.Get("/teams", rt.wrapAdmin(rt.listTeams))
	rt.router.Post("/teams", rt.wrapAdmin(rt.createTeam))
	rt.router.Put("/teams/{id}", rt.wrapAdmin(rt.updateTeam))
	rt.router.Delete("/teams/{id}", rt.wrapAdmin(rt.deleteTeam))

	rt.router.Get("/players", rt.wrapAdmin(rt.listPlayers))
	rt.router.Post("/players", rt.wrapAdmin(rt.createPlayer))
	rt.router.Put("/players/{id}", rt.wrapAdmin(rt.updatePlayer))
	rt.router.Delete("/players/{id}", rt.wrapAdmin(rt.deletePlayer))

	rt.router.Get("/active-event", rt.wrap(rt.getActiveEvent))

	rt.router.Get("/events", rt.wrapAdmin(rt.listEvents))
	rt.router.Post("/events", rt.wrapAdmin(rt.createEvent))
	rt.router.Post("/events/deactivate", rt.wrapAdmin(rt.deactivateEvents))
	rt.router.Post("/events/{id}/activate", rt.wrapAdmin(rt.activateEvent))
	rt.router.Put("/events/{id}", rt.wrapAdmin(rt.updateEvent))
	rt.router.Delete("/events/{id}", rt.wrapAdmin(rt.deleteEvent))
	rt.router.Get("/events/{id}/tickets", rt.wrapAdmin(rt.listEventTickets))

	rt.router.Get("/votes", rt.wrapAdmin(rt.listVotes))
	rt.router.Delete("/votes/{id}", rt.wrapAdmin(rt.deleteVote))

	rt.router.Get("/admins", rt.wrapAdmin(rt.listAdmins))
	rt.router.Post("/admins", rt.wrapAdmin(rt.createAdmin))
	rt.router.Put("/admins/{id}", rt.wrapAdmin(rt.updateAdmin))
	rt.router.Delete("/admins/{id}", rt.wrapAdmin(rt.deleteAdmin))

	return rt.router
}
