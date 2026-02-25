package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
	"github.com/go-chi/chi/v5"
	"github.com/gofrs/uuid"
)

const (
	storyVideoMaxUploadSize = 40 << 20 // 40 MiB
)

var allowedStoryVideoTypes = map[string]string{
	"video/mp4":       ".mp4",
	"video/webm":      ".webm",
	"video/quicktime": ".mov",
}

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
	if payload.ThumbnailURL == "" {
		return "Thumbnail obbligatoria."
	}
	if payload.VideoURL == "" {
		return "URL video obbligatorio."
	}
	return ""
}

func detectStoryVideoContentType(data []byte, fallback string) string {
	if len(data) == 0 {
		return fallback
	}
	detected := http.DetectContentType(data)
	if detected != "application/octet-stream" {
		return strings.ToLower(strings.TrimSpace(detected))
	}
	return strings.ToLower(strings.TrimSpace(fallback))
}

func validateStoryVideoType(contentType string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(contentType))
	if ext, ok := allowedStoryVideoTypes[normalized]; ok {
		return ext, nil
	}
	return "", errors.New("unsupported file type")
}

func (rt *_router) ensureStoryVideoDir(eventID int) (string, error) {
	baseDir := filepath.Join("tmp", "story-videos", fmt.Sprintf("event_%d", eventID))
	absDir, err := filepath.Abs(baseDir)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(absDir, 0o755); err != nil {
		return "", err
	}
	return absDir, nil
}

func readStoryVideoFile(r *http.Request) ([]byte, *multipart.FileHeader, error) {
	if err := r.ParseMultipartForm(storyVideoMaxUploadSize); err != nil {
		return nil, nil, err
	}
	file, header, err := r.FormFile("video")
	if err != nil {
		return nil, nil, err
	}
	defer file.Close()

	limited := io.LimitReader(file, storyVideoMaxUploadSize+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, nil, err
	}
	if len(data) == 0 {
		return nil, nil, errors.New("empty file")
	}
	if len(data) > storyVideoMaxUploadSize {
		return nil, nil, errors.New("file too large")
	}
	return data, header, nil
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

func (rt *_router) uploadAdminStoryVideo(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	eventID, _ := strconv.Atoi(chi.URLParam(r, "eventId"))
	if !rt.ensureEventInOrganization(w, ctx, eventID) {
		return
	}

	data, header, err := readStoryVideoFile(r)
	if err != nil {
		_ = writeJSONMessage(w, http.StatusBadRequest, "Video non valido o troppo pesante.")
		return
	}

	contentType := detectStoryVideoContentType(data, header.Header.Get("Content-Type"))
	ext, err := validateStoryVideoType(contentType)
	if err != nil {
		_ = writeJSONMessage(w, http.StatusBadRequest, "Formato video non supportato (usa MP4, WEBM o MOV).")
		return
	}

	dir, err := rt.ensureStoryVideoDir(eventID)
	if err != nil {
		_ = writeJSONMessage(w, http.StatusInternalServerError, "Impossibile preparare il caricamento video.")
		return
	}

	fileID, err := uuid.NewV4()
	if err != nil {
		_ = writeJSONMessage(w, http.StatusInternalServerError, "Impossibile completare il caricamento video.")
		return
	}
	filename := fileID.String() + ext
	absPath := filepath.Join(dir, filename)
	if err := os.WriteFile(absPath, data, 0o644); err != nil {
		_ = writeJSONMessage(w, http.StatusInternalServerError, "Impossibile salvare il video.")
		return
	}

	resp := map[string]string{
		"video_url": fmt.Sprintf("/events/%d/stories/videos/%s", eventID, filename),
	}
	_ = writeJSON(w, http.StatusCreated, resp)
}

func (rt *_router) getEventStoryVideo(w http.ResponseWriter, r *http.Request, _ reqcontext.RequestContext) {
	eventID, err := strconv.Atoi(chi.URLParam(r, "eventId"))
	if err != nil || eventID <= 0 {
		_ = writeJSONMessage(w, http.StatusBadRequest, "Evento non valido.")
		return
	}
	filename := filepath.Base(strings.TrimSpace(chi.URLParam(r, "filename")))
	if filename == "" || filename == "." {
		_ = writeJSONMessage(w, http.StatusBadRequest, "Video non valido.")
		return
	}
	absPath, err := filepath.Abs(filepath.Join("tmp", "story-videos", fmt.Sprintf("event_%d", eventID), filename))
	if err != nil {
		_ = writeJSONMessage(w, http.StatusNotFound, "Video non trovato.")
		return
	}
	if _, err := os.Stat(absPath); err != nil {
		_ = writeJSONMessage(w, http.StatusNotFound, "Video non trovato.")
		return
	}

	w.Header().Set("Cache-Control", "public, max-age=86400")
	http.ServeFile(w, r, absPath)
}
