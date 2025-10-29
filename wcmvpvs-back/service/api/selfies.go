package api

import (
	"database/sql"
	"encoding/base64"
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
	selfieMaxCaptionLength = 80
	selfieMaxUploadSize    = 8 << 20 // 8 MiB
)

var allowedSelfieTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

type selfieResponse struct {
	ID           int    `json:"id"`
	EventID      int    `json:"event_id"`
	Caption      string `json:"caption"`
	ImageURL     string `json:"image_url"`
	ContentType  string `json:"content_type"`
	Approved     bool   `json:"approved"`
	ShowOnScreen bool   `json:"show_on_screen"`
	SubmittedAt  string `json:"submitted_at"`
}

type adminSelfieResponse struct {
	selfieResponse
	DeviceToken string `json:"device_token"`
}

func (rt *_router) deviceIDFromRequest(r *http.Request) string {
	header := strings.TrimSpace(r.Header.Get("X-Device-ID"))
	if header != "" {
		return header
	}
	return strings.TrimSpace(r.URL.Query().Get("device_id"))
}

func sanitizeCaption(value string) string {
	trimmed := strings.TrimSpace(value)
	runes := []rune(trimmed)
	if len(runes) > selfieMaxCaptionLength {
		return string(runes[:selfieMaxCaptionLength])
	}
	return trimmed
}

func (rt *_router) ensureSelfieDir(eventID int) (string, error) {
	baseDir := filepath.Join("tmp", "selfies", fmt.Sprintf("event_%d", eventID))
	absDir, err := filepath.Abs(baseDir)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(absDir, 0o755); err != nil {
		return "", err
	}
	return absDir, nil
}

func detectContentType(data []byte, fallback string) string {
	if len(data) == 0 {
		return fallback
	}
	detected := http.DetectContentType(data)
	if detected != "application/octet-stream" {
		return detected
	}
	return fallback
}

func (rt *_router) buildSelfieImagePath(eventID, selfieID int) string {
	return fmt.Sprintf("/events/%d/selfies/%d/image", eventID, selfieID)
}

func (rt *_router) ensureSelfieURL(selfie database.Selfie) (database.Selfie, error) {
	if strings.TrimSpace(selfie.ImageURL) != "" {
		return selfie, nil
	}
	if selfie.ID == 0 || selfie.EventID == 0 {
		return selfie, nil
	}
	imageURL := rt.buildSelfieImagePath(selfie.EventID, selfie.ID)
	if err := rt.db.UpdateSelfieURL(selfie.ID, imageURL); err != nil {
		return selfie, err
	}
	selfie.ImageURL = imageURL
	return selfie, nil
}

func decodeBase64Image(raw string) ([]byte, string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, "", errors.New("empty image data")
	}

	var payload string
	var contentType string
	if strings.HasPrefix(trimmed, "data:") {
		comma := strings.Index(trimmed, ",")
		if comma <= 0 {
			return nil, "", errors.New("invalid data url")
		}
		meta := trimmed[:comma]
		payload = trimmed[comma+1:]
		parts := strings.Split(meta, ";")
		if len(parts) > 0 {
			prefix := strings.TrimPrefix(parts[0], "data:")
			if prefix != "" && prefix != parts[0] {
				contentType = prefix
			}
		}
		if len(parts) == 0 || parts[len(parts)-1] != "base64" {
			return nil, "", errors.New("unsupported data url encoding")
		}
	} else {
		payload = trimmed
	}

	data, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return nil, "", err
	}
	if contentType == "" {
		contentType = detectContentType(data, "")
	}
	return data, contentType, nil
}

func validateSelfieContentType(contentType string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(contentType))
	if normalized == "" {
		return "", errors.New("unsupported file type")
	}
	if ext, ok := allowedSelfieTypes[normalized]; ok {
		return ext, nil
	}
	return "", errors.New("unsupported file type")
}

func (rt *_router) readMultipartSelfie(r *http.Request) (string, []byte, string, error) {
	if err := r.ParseMultipartForm(selfieMaxUploadSize); err != nil {
		return "", nil, "", err
	}
	caption := sanitizeCaption(r.FormValue("caption"))
	fieldNames := []string{"image", "photo", "file"}
	var file io.ReadCloser
	var header *multipart.FileHeader
	var err error
	for _, field := range fieldNames {
		file, header, err = r.FormFile(field)
		if err == nil {
			break
		}
	}
	if err != nil {
		return "", nil, "", errors.New("missing file")
	}
	defer file.Close()

	limited := io.LimitReader(file, selfieMaxUploadSize+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return "", nil, "", err
	}
	if len(data) == 0 {
		return "", nil, "", errors.New("empty image data")
	}
	if len(data) > selfieMaxUploadSize {
		return "", nil, "", errors.New("file too large")
	}

	contentType := header.Header.Get("Content-Type")
	contentType = detectContentType(data, contentType)
	return caption, data, contentType, nil
}

func (rt *_router) readJSONSelfie(r *http.Request) (string, []byte, string, error) {
	var payload struct {
		Caption     string `json:"caption"`
		ImageBase64 string `json:"image_base64"`
		Image       string `json:"image"`
	}
	decoder := json.NewDecoder(io.LimitReader(r.Body, selfieMaxUploadSize*2))
	if err := decoder.Decode(&payload); err != nil {
		return "", nil, "", err
	}
	data, contentType, err := decodeBase64Image(payload.ImageBase64)
	if err != nil {
		if payload.Image != "" {
			data, contentType, err = decodeBase64Image(payload.Image)
		}
	}
	if err != nil {
		return "", nil, "", err
	}
	if len(data) > selfieMaxUploadSize {
		return "", nil, "", errors.New("file too large")
	}
	return sanitizeCaption(payload.Caption), data, contentType, nil
}

func (rt *_router) extractSelfiePayload(r *http.Request) (string, []byte, string, error) {
	contentType := strings.ToLower(r.Header.Get("Content-Type"))
	switch {
	case strings.HasPrefix(contentType, "multipart/form-data"):
		return rt.readMultipartSelfie(r)
	case strings.HasPrefix(contentType, "application/json") || strings.HasPrefix(contentType, "text/json"):
		return rt.readJSONSelfie(r)
	default:
		return rt.readMultipartSelfie(r)
	}
}

func buildSelfieResponsePayload(selfie database.Selfie) selfieResponse {
	return selfieResponse{
		ID:           selfie.ID,
		EventID:      selfie.EventID,
		Caption:      selfie.Caption,
		ImageURL:     selfie.ImageURL,
		ContentType:  selfie.ContentType,
		Approved:     selfie.Approved,
		ShowOnScreen: selfie.ShowOnScreen,
		SubmittedAt:  selfie.CreatedAt,
	}
}

func (rt *_router) getVoteStatus(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	eventID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil || eventID <= 0 {
		_ = writeJSONMessage(w, http.StatusBadRequest, "Identificativo evento non valido.")
		return
	}
	deviceID := rt.deviceIDFromRequest(r)
	if deviceID == "" {
		_ = writeJSONMessage(w, http.StatusBadRequest, "Identificativo dispositivo mancante.")
		return
	}
	hasVoted, err := rt.db.HasDeviceVoted(eventID, deviceID)
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot check vote status")
		_ = writeJSONMessage(w, http.StatusInternalServerError, "Impossibile verificare lo stato del voto.")
		return
	}
	resp := struct {
		HasVoted bool `json:"has_voted"`
	}{HasVoted: hasVoted}
	if err := writeJSON(w, http.StatusOK, resp); err != nil {
		ctx.Logger.WithError(err).Error("cannot write vote status response")
	}
}

func (rt *_router) postSelfie(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	eventID, err := strconv.Atoi(chi.URLParam(r, "eventId"))
	if err != nil || eventID <= 0 {
		_ = writeJSONMessage(w, http.StatusBadRequest, "Evento non valido.")
		return
	}
	deviceID := rt.deviceIDFromRequest(r)
	if deviceID == "" {
		_ = writeJSONMessage(w, http.StatusBadRequest, "Identificativo dispositivo mancante.")
		return
	}

	hasVoted, err := rt.db.HasDeviceVoted(eventID, deviceID)
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot verify device vote state")
		_ = writeJSONMessage(w, http.StatusInternalServerError, "Impossibile verificare lo stato del voto.")
		return
	}
	if !hasVoted {
		_ = writeJSONMessage(w, http.StatusForbidden, "Devi prima votare per inviare un selfie.")
		return
	}

	caption, data, contentType, err := rt.extractSelfiePayload(r)
	if err != nil {
		ctx.Logger.WithError(err).Warn("invalid selfie payload")
		_ = writeJSONMessage(w, http.StatusBadRequest, "Immagine non valida o troppo pesante.")
		return
	}
	ext, err := validateSelfieContentType(contentType)
	if err != nil {
		ctx.Logger.WithError(err).Warn("unsupported selfie content type")
		_ = writeJSONMessage(w, http.StatusBadRequest, "Formato immagine non supportato.")
		return
	}
	contentType = strings.ToLower(strings.TrimSpace(contentType))

	storageDir, err := rt.ensureSelfieDir(eventID)
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot ensure selfie directory")
		_ = writeJSONMessage(w, http.StatusInternalServerError, "Servizio non disponibile al momento.")
		return
	}
	fileID, err := uuid.NewV4()
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot generate selfie identifier")
		_ = writeJSONMessage(w, http.StatusInternalServerError, "Servizio non disponibile al momento.")
		return
	}
	filename := fmt.Sprintf("%s%s", fileID.String(), ext)
	fullPath := filepath.Join(storageDir, filename)

	if err := os.WriteFile(fullPath, data, 0o644); err != nil {
		ctx.Logger.WithError(err).Error("cannot persist selfie image")
		_ = writeJSONMessage(w, http.StatusInternalServerError, "Impossibile salvare l'immagine.")
		return
	}

	existing, err := rt.db.GetSelfieForDevice(eventID, deviceID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		ctx.Logger.WithError(err).Error("cannot lookup previous selfie")
	}

	selfie, err := rt.db.SaveSelfie(eventID, deviceID, caption, fullPath, contentType)
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot store selfie metadata")
		_ = writeJSONMessage(w, http.StatusInternalServerError, "Impossibile salvare il selfie.")
		_ = os.Remove(fullPath)
		return
	}

	selfie, err = rt.ensureSelfieURL(selfie)
	if err != nil {
		ctx.Logger.WithError(err).Warn("cannot ensure selfie url")
	}

	if existing.ID > 0 && existing.ImagePath != "" && existing.ImagePath != selfie.ImagePath {
		_ = os.Remove(existing.ImagePath)
	}

	response := buildSelfieResponsePayload(selfie)
	if err := writeJSON(w, http.StatusCreated, response); err != nil {
		ctx.Logger.WithError(err).Error("cannot encode selfie response")
	}
}

func (rt *_router) getOwnSelfie(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	eventID, err := strconv.Atoi(chi.URLParam(r, "eventId"))
	if err != nil || eventID <= 0 {
		_ = writeJSONMessage(w, http.StatusBadRequest, "Evento non valido.")
		return
	}
	deviceID := rt.deviceIDFromRequest(r)
	if deviceID == "" {
		_ = writeJSONMessage(w, http.StatusBadRequest, "Identificativo dispositivo mancante.")
		return
	}
	selfie, err := rt.db.GetSelfieForDevice(eventID, deviceID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		ctx.Logger.WithError(err).Error("cannot fetch device selfie")
		_ = writeJSONMessage(w, http.StatusInternalServerError, "Impossibile recuperare il selfie.")
		return
	}
	selfie, err = rt.ensureSelfieURL(selfie)
	if err != nil {
		ctx.Logger.WithError(err).Warn("cannot ensure selfie url")
	}
	if err := writeJSON(w, http.StatusOK, buildSelfieResponsePayload(selfie)); err != nil {
		ctx.Logger.WithError(err).Error("cannot write selfie response")
	}
}

func (rt *_router) listApprovedSelfies(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	eventID, err := strconv.Atoi(chi.URLParam(r, "eventId"))
	if err != nil || eventID <= 0 {
		_ = writeJSONMessage(w, http.StatusBadRequest, "Evento non valido.")
		return
	}
	selfies, err := rt.db.ListApprovedSelfies(eventID)
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot list approved selfies")
		_ = writeJSONMessage(w, http.StatusInternalServerError, "Impossibile recuperare i selfie approvati.")
		return
	}
	responses := make([]selfieResponse, 0, len(selfies))
	for _, selfie := range selfies {
		selfie, err = rt.ensureSelfieURL(selfie)
		if err != nil {
			ctx.Logger.WithError(err).Warn("cannot ensure selfie url")
		}
		responses = append(responses, buildSelfieResponsePayload(selfie))
	}
	if err := writeJSON(w, http.StatusOK, responses); err != nil {
		ctx.Logger.WithError(err).Error("cannot encode approved selfies")
	}
}

func (rt *_router) listAdminSelfies(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	eventID, err := strconv.Atoi(chi.URLParam(r, "eventId"))
	if err != nil || eventID <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	selfies, err := rt.db.ListEventSelfies(eventID)
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot list selfies")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	responses := make([]adminSelfieResponse, 0, len(selfies))
	for _, selfie := range selfies {
		selfie, err = rt.ensureSelfieURL(selfie)
		if err != nil {
			ctx.Logger.WithError(err).Warn("cannot ensure selfie url")
		}
		responses = append(responses, adminSelfieResponse{
			selfieResponse: buildSelfieResponsePayload(selfie),
			DeviceToken:    selfie.DeviceID,
		})
	}
	if err := writeJSON(w, http.StatusOK, responses); err != nil {
		ctx.Logger.WithError(err).Error("cannot encode admin selfie list")
	}
}

func (rt *_router) updateSelfieModeration(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	selfieID, err := strconv.Atoi(chi.URLParam(r, "selfieId"))
	if err != nil || selfieID <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	var payload struct {
		Approved     *bool `json:"approved"`
		ShowOnScreen *bool `json:"show_on_screen"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	selfie, err := rt.db.GetSelfieByID(selfieID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		ctx.Logger.WithError(err).Error("cannot load selfie")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	approved := selfie.Approved
	showOnScreen := selfie.ShowOnScreen
	if payload.Approved != nil {
		approved = *payload.Approved
	}
	if payload.ShowOnScreen != nil {
		showOnScreen = *payload.ShowOnScreen
	}
	if showOnScreen && !approved {
		approved = true
	}
	if err := rt.db.UpdateSelfieStatus(selfieID, approved, showOnScreen); err != nil {
		ctx.Logger.WithError(err).Error("cannot update selfie status")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	updated, err := rt.db.GetSelfieByID(selfieID)
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot reload selfie")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	updated, err = rt.ensureSelfieURL(updated)
	if err != nil {
		ctx.Logger.WithError(err).Warn("cannot ensure selfie url")
	}

	response := adminSelfieResponse{
		selfieResponse: buildSelfieResponsePayload(updated),
		DeviceToken:    updated.DeviceID,
	}
	if err := writeJSON(w, http.StatusOK, response); err != nil {
		ctx.Logger.WithError(err).Error("cannot encode moderation response")
	}
}

func (rt *_router) getSelfieImage(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	eventID, err := strconv.Atoi(chi.URLParam(r, "eventId"))
	if err != nil || eventID <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	selfieID, err := strconv.Atoi(chi.URLParam(r, "selfieId"))
	if err != nil || selfieID <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	selfie, err := rt.db.GetSelfieByID(selfieID)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if selfie.EventID != eventID {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	allowed := selfie.Approved
	if !allowed {
		deviceID := rt.deviceIDFromRequest(r)
		if deviceID != "" && deviceID == selfie.DeviceID {
			allowed = true
		}
	}
	if !allowed {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	rt.serveSelfieFile(w, r, ctx, selfie, selfie.Approved)
}

func (rt *_router) getAdminSelfieImage(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	selfieID, err := strconv.Atoi(chi.URLParam(r, "selfieId"))
	if err != nil || selfieID <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	selfie, err := rt.db.GetSelfieByID(selfieID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		ctx.Logger.WithError(err).Error("cannot fetch selfie")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	rt.serveSelfieFile(w, r, ctx, selfie, false)
}

func (rt *_router) serveSelfieFile(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext, selfie database.Selfie, allowCache bool) {
	if strings.TrimSpace(selfie.ImagePath) == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	absPath, err := filepath.Abs(selfie.ImagePath)
	if err != nil {
		ctx.Logger.WithError(err).Warn("invalid selfie path")
		w.WriteHeader(http.StatusNotFound)
		return
	}
	baseDir, err := filepath.Abs(filepath.Join("tmp", "selfies"))
	if err != nil {
		ctx.Logger.WithError(err).Warn("cannot resolve selfies directory")
		w.WriteHeader(http.StatusNotFound)
		return
	}
	prefix := baseDir + string(os.PathSeparator)
	if absPath != baseDir && !strings.HasPrefix(absPath, prefix) {
		ctx.Logger.Warn("selfie path outside storage directory")
		w.WriteHeader(http.StatusNotFound)
		return
	}
	file, err := os.Open(absPath)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	contentType := strings.TrimSpace(selfie.ContentType)
	if contentType == "" {
		buf := make([]byte, 512)
		n, _ := file.Read(buf)
		if n > 0 {
			contentType = http.DetectContentType(buf[:n])
			_, _ = file.Seek(0, io.SeekStart)
		}
	} else {
		_, _ = file.Seek(0, io.SeekStart)
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	if allowCache {
		w.Header().Set("Cache-Control", "public, max-age=60")
	} else {
		w.Header().Set("Cache-Control", "no-store")
	}
	w.Header().Set("Content-Type", contentType)

	http.ServeContent(w, r, filepath.Base(absPath), info.ModTime(), file)
}
