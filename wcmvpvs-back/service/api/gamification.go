package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
	"github.com/go-chi/chi/v5"
)

func (rt *_router) listMasterMinigames(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	if !rt.ensureSuperAdmin(w, ctx) {
		return
	}
	games, err := rt.db.ListMinigames()
	if err != nil {
		ctx.Logger.WithError(err).Error("list master minigames")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	_ = writeJSON(w, http.StatusOK, games)
}

func (rt *_router) updateMasterMinigame(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	if !rt.ensureSuperAdmin(w, ctx) {
		return
	}
	var payload struct {
		Sports *[]string `json:"sports"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&payload); err != nil || payload.Sports == nil {
		_ = writeJSONMessage(w, http.StatusBadRequest, "Seleziona gli sport oppure Tutti.")
		return
	}
	game, err := rt.db.UpdateMinigameSports(chi.URLParam(r, "gameId"), *payload.Sports)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			w.WriteHeader(http.StatusNotFound)
		case errors.Is(err, database.ErrInvalidMinigameSports):
			_ = writeJSONMessage(w, http.StatusBadRequest, "Scegli Calcio, Pallavolo, Pallacanestro oppure solo Tutti.")
		default:
			ctx.Logger.WithError(err).Error("update master minigame")
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}
	_ = writeJSON(w, http.StatusOK, game)
}

func (rt *_router) getEventMinigames(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	eventID, err := parseNumericID(chi.URLParam(r, "eventId"))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if !rt.ensureEventInOrganization(w, ctx, eventID) {
		return
	}
	games, err := rt.db.GetOrganizationMinigames(ctx.OrganizationID)
	if err != nil {
		ctx.Logger.WithError(err).Error("load available minigames")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	_ = writeJSON(w, http.StatusOK, games)
}

func (rt *_router) ensureMinigameAvailable(w http.ResponseWriter, ctx reqcontext.RequestContext, gameID string) bool {
	games, err := rt.db.GetOrganizationMinigames(ctx.OrganizationID)
	if err != nil {
		ctx.Logger.WithError(err).Error("check minigame sport availability")
		w.WriteHeader(http.StatusInternalServerError)
		return false
	}
	for _, id := range games {
		if id == database.MinigameID(gameID) {
			return true
		}
	}
	_ = writeJSONMessage(w, http.StatusForbidden, "Minigioco non disponibile per lo sport della società.")
	return false
}

func (rt *_router) withMinigame(gameID string, next httpRouterHandler) httpRouterHandler {
	return func(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
		eventID, err := parseNumericID(chi.URLParam(r, "eventId"))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if !rt.ensureEventInOrganization(w, ctx, eventID) {
			return
		}
		id := gameID
		if id == "volleyball" {
			id = chi.URLParam(r, "gameId")
			if !database.ValidVolleyballGame(id) {
				w.WriteHeader(http.StatusNotFound)
				return
			}
		}
		if id == "branded" {
			event, err := rt.db.GetEventByID(eventID)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			cfg, err := parseBrandedGameConfig(event.BrandedGameConfig)
			if !event.ShowBrandedGame || err != nil {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			id = cfg.GameType
		}
		if rt.ensureMinigameAvailable(w, ctx, id) {
			next(w, r, ctx)
		}
	}
}
