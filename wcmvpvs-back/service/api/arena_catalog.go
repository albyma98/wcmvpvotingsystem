package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
	"github.com/go-chi/chi/v5"
)

func (rt *_router) listMasterArenas(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	if !rt.ensureSuperAdmin(w, ctx) {
		return
	}
	arenas, err := rt.db.ListArenas()
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot list arenas")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	_ = writeJSON(w, http.StatusOK, struct {
		Arenas []database.ArenaSummary `json:"arenas"`
	}{Arenas: arenas})
}

func (rt *_router) getMasterArena(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	if !rt.ensureSuperAdmin(w, ctx) {
		return
	}
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil || id <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	arena, err := rt.db.GetArena(id)
	if err != nil {
		writeArenaCatalogError(w, ctx, err)
		return
	}
	_ = writeJSON(w, http.StatusOK, arena)
}

func (rt *_router) saveMasterArena(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	if !rt.ensureSuperAdmin(w, ctx) {
		return
	}
	var id int
	if r.Method == http.MethodPut {
		var err error
		id, err = strconv.Atoi(chi.URLParam(r, "id"))
		if err != nil || id <= 0 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}

	var arena database.Arena
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 512*1024))
	if err := decoder.Decode(&arena); err != nil || decoder.Decode(new(json.RawMessage)) != io.EOF {
		writeArenaCatalogError(w, ctx, database.ErrInvalidArenaData)
		return
	}
	arena.ID = id

	var err error
	status := http.StatusCreated
	if id == 0 {
		arena, err = rt.db.CreateArena(arena)
	} else {
		status = http.StatusOK
		arena, err = rt.db.UpdateArena(arena)
	}
	if err != nil {
		writeArenaCatalogError(w, ctx, err)
		return
	}
	_ = writeJSON(w, status, arena)
}

func writeArenaCatalogError(w http.ResponseWriter, ctx reqcontext.RequestContext, err error) {
	switch {
	case errors.Is(err, database.ErrInvalidArenaData):
		_ = writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Dati dell’arena non validi."})
	case errors.Is(err, database.ErrArenaSlugExists):
		_ = writeJSON(w, http.StatusConflict, map[string]string{"error": "Lo slug è già utilizzato da un’altra arena."})
	case errors.Is(err, database.ErrInvalidArenaID):
		_ = writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Seleziona un’arena presente nel catalogo."})
	case errors.Is(err, sql.ErrNoRows):
		w.WriteHeader(http.StatusNotFound)
	default:
		ctx.Logger.WithError(err).Error("arena catalog request failed")
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (rt *_router) getOrganizationArenaScene(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	if ctx.OrganizationID <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	arena, err := rt.db.GetArenaByOrganization(ctx.OrganizationID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		writeArenaCatalogError(w, ctx, err)
		return
	}
	var result *database.Arena
	if err == nil && arena.IsActive {
		result = &arena
	}
	w.Header().Set("Cache-Control", "no-store")
	_ = writeJSON(w, http.StatusOK, struct {
		Arena *database.Arena `json:"arena"`
	}{Arena: result})
}

func parseArenaAssignment(raw json.RawMessage) (*int, error) {
	var arenaID *int
	if err := json.Unmarshal(raw, &arenaID); err != nil || (arenaID != nil && *arenaID <= 0) {
		return nil, database.ErrInvalidArenaID
	}
	return arenaID, nil
}

func (rt *_router) setMasterOrganizationArena(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	if !rt.ensureSuperAdmin(w, ctx) {
		return
	}
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil || id <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	var payload struct {
		ArenaID json.RawMessage `json:"arena_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	arenaID, err := parseArenaAssignment(payload.ArenaID)
	if err != nil {
		writeArenaCatalogError(w, ctx, err)
		return
	}
	org, err := rt.db.SetOrganizationArena(id, arenaID)
	if err != nil {
		writeArenaCatalogError(w, ctx, err)
		return
	}
	_ = writeJSON(w, http.StatusOK, org)
}
