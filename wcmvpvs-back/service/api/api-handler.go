package api

import (
	"net/http"
)

// Handler returns an instance of httprouter.Router that handle APIs registered here
func (rt *_router) Handler() http.Handler {
	// Register routes
	rt.router.GET("/", rt.getHelloWorld)
	rt.router.GET("/context", rt.wrap(rt.getContextReply))

	// Special routes
	rt.router.GET("/liveness", rt.liveness)

	rt.router.POST("/vote", rt.wrap(rt.postVote))
	rt.router.POST("/ticket", rt.wrap(rt.postTicket))

	// Admin CRUD routes
	rt.router.POST("/admin/login", rt.wrap(rt.adminLogin))

	rt.router.GET("/teams", rt.wrapAdmin(rt.listTeams))
	rt.router.POST("/teams", rt.wrapAdmin(rt.createTeam))
	rt.router.PUT("/teams/:id", rt.wrapAdmin(rt.updateTeam))
	rt.router.DELETE("/teams/:id", rt.wrapAdmin(rt.deleteTeam))

	rt.router.GET("/players", rt.wrapAdmin(rt.listPlayers))
	rt.router.POST("/players", rt.wrapAdmin(rt.createPlayer))
	rt.router.PUT("/players/:id", rt.wrapAdmin(rt.updatePlayer))
	rt.router.DELETE("/players/:id", rt.wrapAdmin(rt.deletePlayer))

	rt.router.GET("/events", rt.wrapAdmin(rt.listEvents))
	rt.router.POST("/events", rt.wrapAdmin(rt.createEvent))
	rt.router.PUT("/events/:id", rt.wrapAdmin(rt.updateEvent))
	rt.router.DELETE("/events/:id", rt.wrapAdmin(rt.deleteEvent))

	rt.router.GET("/votes", rt.wrapAdmin(rt.listVotes))
	rt.router.DELETE("/votes/:id", rt.wrapAdmin(rt.deleteVote))

	rt.router.GET("/admins", rt.wrapAdmin(rt.listAdmins))
	rt.router.POST("/admins", rt.wrapAdmin(rt.createAdmin))
	rt.router.PUT("/admins/:id", rt.wrapAdmin(rt.updateAdmin))
	rt.router.DELETE("/admins/:id", rt.wrapAdmin(rt.deleteAdmin))

	return rt.router
}
