package api

import (
	"github.com/go-chi/chi/v5"
)

// Handler returns an instance of chi.Router that handles APIs registered here
func (rt *_router) Handler() chi.Router {
	// Register routes
	rt.router.Get("/", rt.getHelloWorld)
	rt.router.Get("/context", rt.wrap(rt.getContextReply))
	rt.router.Get("/t", rt.ticketValidationPage)
	rt.router.Get("/tickets/validate", rt.wrap(rt.ticketValidationStatus))

	// Special routes
	rt.router.Get("/liveness", rt.liveness)

	rt.router.Post("/vote", rt.wrap(rt.postVote))
	// Admin CRUD routes
	rt.router.Post("/admin/login", rt.wrap(rt.adminLogin))

	rt.router.Get("/public/players", rt.wrap(rt.listPublicPlayers))

	rt.router.Get("/teams", rt.wrapAdmin(rt.listTeams))
	rt.router.Post("/teams", rt.wrapAdmin(rt.createTeam))
	rt.router.Put("/teams/{id}", rt.wrapAdmin(rt.updateTeam))
	rt.router.Delete("/teams/{id}", rt.wrapAdmin(rt.deleteTeam))

	rt.router.Get("/players", rt.wrapAdmin(rt.listPlayers))
	rt.router.Post("/players", rt.wrapAdmin(rt.createPlayer))
	rt.router.Put("/players/{id}", rt.wrapAdmin(rt.updatePlayer))
	rt.router.Delete("/players/{id}", rt.wrapAdmin(rt.deletePlayer))

	rt.router.Get("/active-event", rt.wrap(rt.getActiveEvent))
	rt.router.Get("/sponsors", rt.wrap(rt.listPublicSponsors))

	rt.router.Get("/events", rt.wrapAdmin(rt.listEvents))
	rt.router.Post("/events", rt.wrapAdmin(rt.createEvent))
	rt.router.Post("/events/deactivate", rt.wrapAdmin(rt.deactivateEvents))
	rt.router.Post("/events/{id}/close-votes", rt.wrapAdmin(rt.closeEventVoting))
	rt.router.Post("/events/{id}/activate", rt.wrapAdmin(rt.activateEvent))
	rt.router.Put("/events/{id}", rt.wrapAdmin(rt.updateEvent))
	rt.router.Delete("/events/{id}", rt.wrapAdmin(rt.deleteEvent))
	rt.router.Get("/events/{id}/tickets", rt.wrapAdmin(rt.listEventTickets))
	rt.router.Post("/events/{id}/validate-ticket", rt.wrapAdmin(rt.validateTicket))
	rt.router.Post("/events/{eventId}/prizes/{prizeId}/assign", rt.wrapAdmin(rt.assignPrizeWinner))
	rt.router.Delete("/events/{eventId}/prizes/{prizeId}/winner", rt.wrapAdmin(rt.clearPrizeWinner))
	rt.router.Get("/events/{id}/results", rt.wrap(rt.getEventResults))

	rt.router.Get("/votes", rt.wrapAdmin(rt.listVotes))
	rt.router.Delete("/votes/{id}", rt.wrapAdmin(rt.deleteVote))

	rt.router.Get("/admins", rt.wrapAdmin(rt.listAdmins))
	rt.router.Post("/admins", rt.wrapAdmin(rt.createAdmin))
	rt.router.Put("/admins/{id}", rt.wrapAdmin(rt.updateAdmin))
	rt.router.Delete("/admins/{id}", rt.wrapAdmin(rt.deleteAdmin))

	rt.router.Get("/admin/sponsors", rt.wrapAdmin(rt.listAllSponsors))
	rt.router.Post("/admin/sponsors", rt.wrapAdmin(rt.createSponsor))
	rt.router.Put("/admin/sponsors/{id}", rt.wrapAdmin(rt.updateSponsor))
	rt.router.Delete("/admin/sponsors/{id}", rt.wrapAdmin(rt.deleteSponsor))

	return rt.router
}
