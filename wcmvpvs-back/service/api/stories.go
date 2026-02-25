package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
	"github.com/go-chi/chi/v5"
)

func parseStoryPayload(r *http.Request) (database.EventStory, error) {
	var payload database.EventStory
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		return database.EventStory{}, err
	}
	payload.PlayerName = strings.TrimSpace(payload.PlayerName)
	payload.ThumbnailURL = strings.TrimSpace(payload.ThumbnailURL)
	payload.VideoURL = strings.TrimSpace(payload.VideoURL)
	payload.Title = strings.TrimSpace(payload.Title)
	return payload, nil
}

func validateStoryPayload(payload database.EventStory) string {
	if payload.PlayerName == "" {
		return "Nome giocatore obbligatorio."
	}
	if payload.ThumbnailURL == "" {
		return "Thumbnail obbligatoria."
	}
	if payload.VideoURL == "" {
		return "URL video obbligatorio."
	}
	return ""
}

func (rt *_router) listPublicEventStories(w http.ResponseWriter, r *http.Request, _ reqcontext.RequestContext) {
	eventID, err := strconv.Atoi(chi.URLParam(r, "eventId"))
	if err != nil || eventID <= 0 {
		_ = writeJSONMessage(w, http.StatusBadRequest, "Evento non valido.")
		return
	}
	stories, err := rt.db.ListEventStories(eventID, false)
	if err != nil {
		_ = writeJSONMessage(w, http.StatusInternalServerError, "Impossibile caricare le stories.")
		return
	}
	_ = writeJSON(w, http.StatusOK, stories)
}

func (rt *_router) listAdminEventStories(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	eventID, _ := strconv.Atoi(chi.URLParam(r, "eventId"))
	if !rt.ensureEventInOrganization(w, ctx, eventID) {
		return
	}
	stories, err := rt.db.ListEventStories(eventID, true)
	if err != nil {
		_ = writeJSONMessage(w, http.StatusInternalServerError, "Impossibile caricare le stories.")
		return
	}
	_ = writeJSON(w, http.StatusOK, stories)
}

func (rt *_router) createAdminEventStory(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	eventID, _ := strconv.Atoi(chi.URLParam(r, "eventId"))
	if !rt.ensureEventInOrganization(w, ctx, eventID) {
		return
	}
	payload, err := parseStoryPayload(r)
	if err != nil {
		_ = writeJSONMessage(w, http.StatusBadRequest, "Payload non valido.")
		return
	}
	if msg := validateStoryPayload(payload); msg != "" {
		_ = writeJSONMessage(w, http.StatusBadRequest, msg)
		return
	}
	created, err := rt.db.CreateEventStory(eventID, payload)
	if err != nil {
		_ = writeJSONMessage(w, http.StatusInternalServerError, "Impossibile creare la story.")
		return
	}
	_ = writeJSON(w, http.StatusCreated, created)
}

func (rt *_router) updateAdminEventStory(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	eventID, _ := strconv.Atoi(chi.URLParam(r, "eventId"))
	if !rt.ensureEventInOrganization(w, ctx, eventID) {
		return
	}
	storyID, err := strconv.Atoi(chi.URLParam(r, "storyId"))
	if err != nil || storyID <= 0 {
		_ = writeJSONMessage(w, http.StatusBadRequest, "Story non valida.")
		return
	}
	payload, err := parseStoryPayload(r)
	if err != nil {
		_ = writeJSONMessage(w, http.StatusBadRequest, "Payload non valido.")
		return
	}
	if msg := validateStoryPayload(payload); msg != "" {
		_ = writeJSONMessage(w, http.StatusBadRequest, msg)
		return
	}
	updated, err := rt.db.UpdateEventStory(eventID, storyID, payload)
	if err != nil {
		if err == sql.ErrNoRows {
			_ = writeJSONMessage(w, http.StatusNotFound, "Story non trovata.")
			return
		}
		_ = writeJSONMessage(w, http.StatusInternalServerError, "Impossibile aggiornare la story.")
		return
	}
	_ = writeJSON(w, http.StatusOK, updated)
}

func (rt *_router) deleteAdminEventStory(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	eventID, _ := strconv.Atoi(chi.URLParam(r, "eventId"))
	if !rt.ensureEventInOrganization(w, ctx, eventID) {
		return
	}
	storyID, err := strconv.Atoi(chi.URLParam(r, "storyId"))
	if err != nil || storyID <= 0 {
		_ = writeJSONMessage(w, http.StatusBadRequest, "Story non valida.")
		return
	}
	if err := rt.db.DeleteEventStory(eventID, storyID); err != nil {
		if err == sql.ErrNoRows {
			_ = writeJSONMessage(w, http.StatusNotFound, "Story non trovata.")
			return
		}
		_ = writeJSONMessage(w, http.StatusInternalServerError, "Impossibile eliminare la story.")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
