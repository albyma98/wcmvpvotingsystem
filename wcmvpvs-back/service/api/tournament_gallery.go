package api

// ============================================================================
// TOURNAMENT GALLERY — foto scattate dai tifosi durante il torneo.
// Chiunque apre la sezione Gallery vede le miniature e può pubblicare una foto
// ("SCATTA FOTO"). Le foto sono auto-pubblicate (esperienza social) e il delete
// resta lato admin (moderazione). Storage su disco come i selfie (non data-URL
// in DB: le foto sono tante e pesanti); il listing ritorna solo gli id e il
// client costruisce l'URL immagine. Ogni upload/delete fa broadcast SSE, così
// la gallery si popola in tempo reale per tutti i device connessi.
// ============================================================================

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
)

const maxGalleryImageBytes = 6 << 20 // 6 MiB per foto (dopo il decode)

func registerTournamentGalleryRoutes(rt *_router) {
	rt.router.Get("/v1/tournaments/{slug}/gallery", rt.galleryList)
	rt.router.Post("/v1/tournaments/{slug}/gallery", rt.galleryUpload)
	rt.router.Get("/v1/tournaments/{slug}/gallery/{id}/image", rt.galleryImage)
	// Moderazione: rimozione foto dal pannello admin torneo.
	rt.router.Delete("/v1/ta/{slug}/gallery/{id}", rt.wrapTA(rt.galleryAdminDelete))
}

// EnsureTournamentGalleryTables crea la tabella gallery (idempotente).
func (s *Store) EnsureTournamentGalleryTables() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS tournament_gallery (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			event_id INTEGER NOT NULL,
			path TEXT NOT NULL,
			content_type TEXT NOT NULL DEFAULT '',
			device_id TEXT NOT NULL DEFAULT '',
			approved INTEGER NOT NULL DEFAULT 1,
			created_at TEXT NOT NULL DEFAULT (datetime('now'))
		);`)
	if err != nil {
		return fmt.Errorf("tournament gallery table: %w", err)
	}
	return nil
}

// --- Store ------------------------------------------------------------------

type GalleryPhoto struct {
	ID        int64  `json:"id"`
	CreatedAt string `json:"createdAt"`
}

func (s *Store) InsertGalleryPhoto(eventID int64, path, contentType, deviceID string) (int64, error) {
	res, err := s.db.Exec(`
		INSERT INTO tournament_gallery (event_id, path, content_type, device_id)
		VALUES (?, ?, ?, ?)`, eventID, path, contentType, deviceID)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) ListGalleryPhotos(eventID int64) ([]GalleryPhoto, error) {
	rows, err := s.db.Query(`
		SELECT id, created_at FROM tournament_gallery
		WHERE event_id = ? AND approved = 1
		ORDER BY id DESC`, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]GalleryPhoto, 0, 32)
	for rows.Next() {
		var p GalleryPhoto
		if err := rows.Scan(&p.ID, &p.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// GetGalleryPhotoFile ritorna path+content-type di una foto, scopata sull'evento.
func (s *Store) GetGalleryPhotoFile(eventID, id int64) (string, string, error) {
	var path, ctype string
	err := s.db.QueryRow(`
		SELECT path, content_type FROM tournament_gallery WHERE id = ? AND event_id = ?`,
		id, eventID).Scan(&path, &ctype)
	if errors.Is(err, sql.ErrNoRows) {
		return "", "", errTANotFound
	}
	return path, ctype, err
}

// DeleteGalleryPhoto rimuove la riga e ritorna il path da cancellare su disco.
func (s *Store) DeleteGalleryPhoto(eventID, id int64) (string, error) {
	var path string
	err := s.db.QueryRow(`
		SELECT path FROM tournament_gallery WHERE id = ? AND event_id = ?`, id, eventID).Scan(&path)
	if errors.Is(err, sql.ErrNoRows) {
		return "", errTANotFound
	}
	if err != nil {
		return "", err
	}
	_, err = s.db.Exec(`DELETE FROM tournament_gallery WHERE id = ? AND event_id = ?`, id, eventID)
	return path, err
}

// --- Storage su disco -------------------------------------------------------

const galleryStorageRoot = "tmp/tournament_gallery"

func galleryEventDir(eventID int64) (string, error) {
	abs, err := filepath.Abs(filepath.Join(galleryStorageRoot, fmt.Sprintf("event_%d", eventID)))
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return "", err
	}
	return abs, nil
}

// --- Handlers ---------------------------------------------------------------

func (rt *_router) galleryList(w http.ResponseWriter, r *http.Request) {
	eventID, err := rt.store.EventIDBySlug(r.Context(), chi.URLParam(r, "slug"))
	if err != nil {
		http.Error(w, `{"error":"tournament_not_found"}`, http.StatusNotFound)
		return
	}
	photos, err := rt.store.ListGalleryPhotos(eventID)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"photos": photos})
}

func (rt *_router) galleryUpload(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	eventID, err := rt.store.EventIDBySlug(r.Context(), slug)
	if err != nil {
		http.Error(w, `{"error":"tournament_not_found"}`, http.StatusNotFound)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxGalleryImageBytes*2) // base64 gonfia ~33%
	var body struct {
		Image string `json:"image"` // data-URL immagine (ridimensionata lato client)
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, `{"error":"bad_json"}`, http.StatusBadRequest)
		return
	}
	data, contentType, err := decodeBase64Image(body.Image)
	if err != nil {
		http.Error(w, `{"error":"bad_image"}`, http.StatusBadRequest)
		return
	}
	if len(data) == 0 || len(data) > maxGalleryImageBytes {
		http.Error(w, `{"error":"image_too_large"}`, http.StatusBadRequest)
		return
	}
	ext, err := validateSelfieContentType(contentType)
	if err != nil {
		http.Error(w, `{"error":"bad_type"}`, http.StatusBadRequest)
		return
	}

	dir, err := galleryEventDir(eventID)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	nameBuf := make([]byte, 12)
	if _, err := rand.Read(nameBuf); err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	fullPath := filepath.Join(dir, base64.RawURLEncoding.EncodeToString(nameBuf)+ext)
	if err := os.WriteFile(fullPath, data, 0o644); err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}

	id, err := rt.store.InsertGalleryPhoto(eventID, fullPath, strings.ToLower(contentType), rt.deviceIDFromRequest(r))
	if err != nil {
		_ = os.Remove(fullPath)
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	rt.tournamentHub.Broadcast(int(eventID)) // gallery live per tutti
	writeJSON(w, http.StatusCreated, map[string]interface{}{"id": id})
}

func (rt *_router) galleryImage(w http.ResponseWriter, r *http.Request) {
	eventID, err := rt.store.EventIDBySlug(r.Context(), chi.URLParam(r, "slug"))
	if err != nil {
		http.Error(w, `{"error":"tournament_not_found"}`, http.StatusNotFound)
		return
	}
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	path, ctype, err := rt.store.GetGalleryPhotoFile(eventID, id)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	rt.serveGalleryFile(w, r, path, ctype)
}

func (rt *_router) galleryAdminDelete(w http.ResponseWriter, r *http.Request, eventID int64) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	path, err := rt.store.DeleteGalleryPhoto(eventID, id)
	if errors.Is(err, errTANotFound) {
		http.Error(w, `{"error":"not_found"}`, http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	if path != "" {
		_ = os.Remove(path)
	}
	rt.tournamentHub.Broadcast(int(eventID))
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true})
}

// serveGalleryFile serve il file con guardia anti path-traversal (deve stare
// dentro la cartella di storage) e cache lunga (contenuto immutabile per id).
func (rt *_router) serveGalleryFile(w http.ResponseWriter, r *http.Request, path, ctype string) {
	if strings.TrimSpace(path) == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	base, err := filepath.Abs(galleryStorageRoot)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if abs != base && !strings.HasPrefix(abs, base+string(os.PathSeparator)) {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	f, err := os.Open(abs)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if ctype == "" {
		ctype = "image/jpeg"
	}
	w.Header().Set("Content-Type", ctype)
	w.Header().Set("Cache-Control", "public, max-age=86400")
	http.ServeContent(w, r, filepath.Base(abs), info.ModTime(), f)
}
