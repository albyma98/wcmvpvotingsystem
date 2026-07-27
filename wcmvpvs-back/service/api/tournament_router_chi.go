package api

// ============================================================================
// INTEGRAZIONE ROUTER — modalità torneo.
// Il router del progetto è un chi minimale vendorizzato (vendor/.../chi/v5)
// che espone solo Get/Post/Put/Delete + URLParam: NIENTE Mount/Route/sub-router.
// Quindi le route del torneo vengono registrate direttamente sul mux principale
// con path completi, esattamente come tutte le altre route in api-handler.go.
//
// Caddy/nginx strippano il prefisso /api prima del backend: il client chiama
// /api/v1/tournaments/{slug}/home e qui arriva /v1/tournaments/{slug}/home.
// ============================================================================

// registerTournamentRoutes monta le due route della home torneo sul router reale.
// Chiamato da Handler() in api-handler.go.
func registerTournamentRoutes(rt *_router) {
	// Bootstrap tabelle del mondo torneo (idempotente) + route admin.
	if err := rt.store.EnsureTournamentAdminTables(); err != nil {
		panic("tournament admin tables: " + err.Error())
	}
	registerTournamentAdminRoutes(rt)
	if err := rt.store.EnsureTournamentP1Tables(); err != nil {
		panic("tournament p1 tables: " + err.Error())
	}
	registerTournamentP1Routes(rt)
	rt.resumeBracketAutoGenerations()
	if err := rt.store.EnsureTournamentGalleryTables(); err != nil {
		panic("tournament gallery tables: " + err.Error())
	}
	registerTournamentGalleryRoutes(rt)
	if err := rt.store.EnsureTournamentMVPTables(); err != nil {
		panic("tournament mvp tables: " + err.Error())
	}
	registerTournamentMVPRoutes(rt)

	// Snapshot completo della home: chiamato una volta al load. Cache-Control 30s.
	rt.router.Get("/v1/tournaments/{slug}/home", rt.HandleTournamentHome)
	rt.router.Post("/v1/tournaments/{slug}/shop/reservations", rt.HandleTournamentShopReservation)

	// Polling leggero dello stato live: ogni 10s dai device in tribuna.
	// Cache in-memory TTL 3s protegge SQLite dal thundering herd.
	rt.router.Get("/v1/tournaments/{slug}/live", rt.HandleTournamentLive)

}
