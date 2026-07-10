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
	rt.router.Get("/v1/tournaments/{slug}/gallery/{id}/thumb", rt.galleryThumb)
	// Moderazione: rimozione foto dal pannello admin torneo.
	rt.router.Delete("/v1/ta/{slug}/gallery/{id}", rt.wrapTA(rt.galleryAdminDelete))
}

// EnsureTournamentGalleryTables crea/riconcilia la tabella gallery (idempotente).
// I DB predatanti il modello "torneo = event" hanno una `tournament_gallery`
// legacy (colonna `tournament_id NOT NULL`, niente `event_id`/`path`) dalla
// vecchia feature tornei: CREATE IF NOT EXISTS la lascerebbe com'è e gli INSERT
// event-based fallirebbero. Come per teams/sponsors: reconcile + ALTER idempotenti.
func (s *Store) EnsureTournamentGalleryTables() error {
	const createSQL = `CREATE TABLE tournament_gallery (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		event_id INTEGER NOT NULL,
		path TEXT NOT NULL DEFAULT '',
		thumb_path TEXT NOT NULL DEFAULT '',
		content_type TEXT NOT NULL DEFAULT '',
		device_id TEXT NOT NULL DEFAULT '',
		approved INTEGER NOT NULL DEFAULT 1,
		created_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`
	if _, err := s.db.Exec(`CREATE TABLE IF NOT EXISTS tournament_gallery (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		event_id INTEGER NOT NULL,
		path TEXT NOT NULL DEFAULT '',
		thumb_path TEXT NOT NULL DEFAULT '',
		content_type TEXT NOT NULL DEFAULT '',
		device_id TEXT NOT NULL DEFAULT '',
		approved INTEGER NOT NULL DEFAULT 1,
		created_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`); err != nil {
		return fmt.Errorf("tournament gallery table: %w", err)
	}
	if err := s.reconcileTournamentTable("tournament_gallery", createSQL,
		[]string{"id", "event_id", "path", "thumb_path", "content_type", "device_id", "approved", "created_at"}); err != nil {
		return err
	}
	// ALTER idempotenti: copre DB con event_id ma colonne mancanti.
	for _, alter := range []string{
		`ALTER TABLE tournament_gallery ADD COLUMN path TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE tournament_gallery ADD COLUMN thumb_path TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE tournament_gallery ADD COLUMN content_type TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE tournament_gallery ADD COLUMN device_id TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE tournament_gallery ADD COLUMN approved INTEGER NOT NULL DEFAULT 1`,
	} {
		if _, err := s.db.Exec(alter); err != nil && !strings.Contains(err.Error(), "duplicate column") {
			return fmt.Errorf("gallery column ensure (%s): %w", alter, err)
		}
	}
	return nil
}

// --- Store ------------------------------------------------------------------

type GalleryPhoto struct {
	ID        int64  `json:"id"`
	CreatedAt string `json:"createdAt"`
}

func (s *Store) InsertGalleryPhoto(eventID int64, path, thumbPath, contentType, deviceID string) (int64, error) {
	res, err := s.db.Exec(`
		INSERT INTO tournament_gallery (event_id, path, thumb_path, content_type, device_id)
		VALUES (?, ?, ?, ?, ?)`, eventID, path, thumbPath, contentType, deviceID)
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

// GetGalleryPhotoFile ritorna path (full)+content-type di una foto, scopata sull'evento.
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

// GetGalleryThumbFile ritorna il path della miniatura (se presente) o, in
// fallback, quello della foto full (righe legacy senza thumbnail).
func (s *Store) GetGalleryThumbFile(eventID, id int64) (string, string, error) {
	var path, thumb, ctype string
	err := s.db.QueryRow(`
		SELECT path, COALESCE(thumb_path,''), content_type FROM tournament_gallery WHERE id = ? AND event_id = ?`,
		id, eventID).Scan(&path, &thumb, &ctype)
	if errors.Is(err, sql.ErrNoRows) {
		return "", "", errTANotFound
	}
	if err != nil {
		return "", "", err
	}
	if strings.TrimSpace(thumb) != "" {
		return thumb, ctype, nil
	}
	return path, ctype, nil // legacy: nessuna miniatura, si serve la full
}

// DeleteGalleryPhoto rimuove la riga e ritorna i path (full, thumb) da cancellare su disco.
func (s *Store) DeleteGalleryPhoto(eventID, id int64) (string, string, error) {
	var path, thumb string
	err := s.db.QueryRow(`
		SELECT path, COALESCE(thumb_path,'') FROM tournament_gallery WHERE id = ? AND event_id = ?`,
		id, eventID).Scan(&path, &thumb)
	if errors.Is(err, sql.ErrNoRows) {
		return "", "", errTANotFound
	}
	if err != nil {
		return "", "", err
	}
	_, err = s.db.Exec(`DELETE FROM tournament_gallery WHERE id = ? AND event_id = ?`, id, eventID)
	return path, thumb, err
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

	r.Body = http.MaxBytesReader(w, r.Body, maxGalleryImageBytes*2) // full + thumb, base64 gonfia ~33%
	var body struct {
		Image string `json:"image"` // data-URL foto full (qualità alta, per l'ingrandimento)
		Thumb string `json:"thumb"` // data-URL miniatura (piccola, per la griglia) — facoltativa
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, `{"error":"bad_json"}`, http.StatusBadRequest)
		return
	}

	dir, err := galleryEventDir(eventID)
	if err != nil {
		rt.baseLogger.WithError(err).Error("gallery: cannot ensure storage dir")
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	// writeImg: decodifica+valida+scrive una data-URL immagine su disco.
	writeImg := func(dataURL string) (path, ctype string, err error) {
		data, contentType, err := decodeBase64Image(dataURL)
		if err != nil {
			return "", "", fmt.Errorf("bad_image")
		}
		if len(data) == 0 || len(data) > maxGalleryImageBytes {
			return "", "", fmt.Errorf("image_too_large")
		}
		ext, err := validateSelfieContentType(contentType)
		if err != nil {
			return "", "", fmt.Errorf("bad_type")
		}
		nameBuf := make([]byte, 12)
		if _, err := rand.Read(nameBuf); err != nil {
			return "", "", err
		}
		p := filepath.Join(dir, base64.RawURLEncoding.EncodeToString(nameBuf)+ext)
		if err := os.WriteFile(p, data, 0o644); err != nil {
			return "", "", err
		}
		return p, strings.ToLower(contentType), nil
	}

	fullPath, contentType, err := writeImg(body.Image)
	if err != nil {
		code := err.Error()
		if code != "bad_image" && code != "image_too_large" && code != "bad_type" {
			rt.baseLogger.WithError(err).Error("gallery: cannot write full")
			code = "internal"
		}
		http.Error(w, `{"error":"`+code+`"}`, http.StatusBadRequest)
		return
	}
	// Miniatura (facoltativa): se manca o è invalida, non blocca l'upload — la
	// griglia farà fallback alla full via GetGalleryThumbFile.
	var thumbPath string
	if strings.TrimSpace(body.Thumb) != "" {
		if tp, _, terr := writeImg(body.Thumb); terr == nil {
			thumbPath = tp
		} else {
			rt.baseLogger.WithError(terr).Warn("gallery: thumb scartata, uso la full")
		}
	}

	id, err := rt.store.InsertGalleryPhoto(eventID, fullPath, thumbPath, contentType, rt.deviceIDFromRequest(r))
	if err != nil {
		rt.baseLogger.WithError(err).WithField("eventID", eventID).Error("gallery: cannot insert photo")
		_ = os.Remove(fullPath)
		if thumbPath != "" {
			_ = os.Remove(thumbPath)
		}
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

// galleryThumb serve la miniatura (griglia). Fallback alla full per righe legacy.
func (rt *_router) galleryThumb(w http.ResponseWriter, r *http.Request) {
	eventID, err := rt.store.EventIDBySlug(r.Context(), chi.URLParam(r, "slug"))
	if err != nil {
		http.Error(w, `{"error":"tournament_not_found"}`, http.StatusNotFound)
		return
	}
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	path, ctype, err := rt.store.GetGalleryThumbFile(eventID, id)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	rt.serveGalleryFile(w, r, path, ctype)
}

func (rt *_router) galleryAdminDelete(w http.ResponseWriter, r *http.Request, eventID int64) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	path, thumb, err := rt.store.DeleteGalleryPhoto(eventID, id)
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
	if thumb != "" && thumb != path {
		_ = os.Remove(thumb)
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
