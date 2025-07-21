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
	rt.router.GET("/teams", rt.wrap(rt.listTeams))
	rt.router.POST("/teams", rt.wrap(rt.createTeam))
	rt.router.PUT("/teams/:id", rt.wrap(rt.updateTeam))
	rt.router.DELETE("/teams/:id", rt.wrap(rt.deleteTeam))

	rt.router.GET("/players", rt.wrap(rt.listPlayers))
	rt.router.POST("/players", rt.wrap(rt.createPlayer))
	rt.router.PUT("/players/:id", rt.wrap(rt.updatePlayer))
	rt.router.DELETE("/players/:id", rt.wrap(rt.deletePlayer))

	rt.router.GET("/events", rt.wrap(rt.listEvents))
	rt.router.POST("/events", rt.wrap(rt.createEvent))
	rt.router.PUT("/events/:id", rt.wrap(rt.updateEvent))
	rt.router.DELETE("/events/:id", rt.wrap(rt.deleteEvent))

	rt.router.GET("/votes", rt.wrap(rt.listVotes))
	rt.router.DELETE("/votes/:id", rt.wrap(rt.deleteVote))

	rt.router.GET("/admins", rt.wrap(rt.listAdmins))
	rt.router.POST("/admins", rt.wrap(rt.createAdmin))
	rt.router.PUT("/admins/:id", rt.wrap(rt.updateAdmin))
	rt.router.DELETE("/admins/:id", rt.wrap(rt.deleteAdmin))
	
	return rt.router
}
