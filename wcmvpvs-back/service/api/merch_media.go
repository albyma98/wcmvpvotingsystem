package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/binary"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
	"github.com/go-chi/chi/v5"
)

const merchMaxImageBytes = 5 << 20

func validateMerchImage(data []byte) (string, error) {
	if len(data) == 0 || len(data) > merchMaxImageBytes {
		return "", merchInvalid("L'immagine deve avere dimensione massima di 5 MiB")
	}
	mime := http.DetectContentType(data)
	var width, height int
	switch mime {
	case "image/jpeg", "image/png":
		info, _, err := image.DecodeConfig(bytes.NewReader(data))
		if err != nil {
			return "", merchInvalid("Immagine non valida")
		}
		width, height = info.Width, info.Height
	case "image/webp":
		width, height = merchWebPDimensions(data)
	default:
		return "", merchInvalid("Carica un'immagine JPEG, PNG o WebP")
	}
	if width <= 0 || height <= 0 || width > 6000 || height > 6000 || width*height > 20000000 {
		return "", merchInvalid("Dimensioni immagine non valide (massimo 6000 pixel per lato e 20 megapixel)")
	}
	if mime != "image/webp" {
		if _, _, err := image.Decode(bytes.NewReader(data)); err != nil {
			return "", merchInvalid("Immagine non valida")
		}
	}
	return mime, nil
}

func merchWebPDimensions(data []byte) (int, int) {
	if len(data) < 25 || uint64(binary.LittleEndian.Uint32(data[4:8]))+8 != uint64(len(data)) {
		return 0, 0
	}
	size := uint64(binary.LittleEndian.Uint32(data[16:20]))
	if size > uint64(len(data)-20) {
		return 0, 0
	}
	body := data[20 : 20+int(size)]
	switch string(data[12:16]) {
	case "VP8X":
		if len(body) != 10 || body[0]&0x02 != 0 {
			return 0, 0
		} // static images only
		readDimension := func(v []byte) int { return 1 + int(v[0]) + (int(v[1]) << 8) + (int(v[2]) << 16) }
		// An extended canvas still requires an actual image chunk.
		for offset := 30; offset+8 <= len(data); {
			chunkSize := uint64(binary.LittleEndian.Uint32(data[offset+4 : offset+8]))
			if chunkSize > uint64(len(data)-offset-8) {
				return 0, 0
			}
			kind := string(data[offset : offset+4])
			if (kind == "VP8 " && chunkSize >= 10) || (kind == "VP8L" && chunkSize >= 5) {
				return readDimension(body[4:7]), readDimension(body[7:10])
			}
			offset += 8 + int(chunkSize) + int(chunkSize%2)
		}
	case "VP8L":
		if len(body) < 5 || body[0] != 0x2f {
			return 0, 0
		}
		bits := binary.LittleEndian.Uint32(body[1:5])
		if bits>>29 != 0 {
			return 0, 0
		}
		return int(bits&0x3fff) + 1, int((bits>>14)&0x3fff) + 1
	case "VP8 ":
		if len(body) < 10 || !bytes.Equal(body[3:6], []byte{0x9d, 0x01, 0x2a}) {
			return 0, 0
		}
		return int(binary.LittleEndian.Uint16(body[6:8]) & 0x3fff), int(binary.LittleEndian.Uint16(body[8:10]) & 0x3fff)
	}
	return 0, 0
}

func (rt *_router) postMerchMedia(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	r.Body = http.MaxBytesReader(w, r.Body, merchMaxImageBytes+(64<<10))
	reader, err := r.MultipartReader()
	if err != nil {
		rt.writeMerchError(w, merchInvalid("Caricamento multipart non valido"))
		return
	}
	part, err := reader.NextPart()
	if err != nil || part.FormName() != "file" || part.FileName() == "" {
		rt.writeMerchError(w, merchInvalid("Seleziona un'immagine"))
		return
	}
	defer part.Close()
	data, err := io.ReadAll(io.LimitReader(part, merchMaxImageBytes+1))
	if err != nil {
		rt.writeMerchError(w, merchInvalid("Caricamento immagine non riuscito"))
		return
	}
	mime, err := validateMerchImage(data)
	if err != nil {
		rt.writeMerchError(w, err)
		return
	}
	if _, err = reader.NextPart(); err != io.EOF {
		rt.writeMerchError(w, merchInvalid("Carica una sola immagine per richiesta"))
		return
	}
	id, err := merchID()
	if err == nil {
		err = rt.merchWrite(r.Context(), func(tx *sql.Tx) error {
			_, err := tx.ExecContext(r.Context(), "INSERT INTO merch_media (id,organization_id,mime,content,created_at) VALUES (?,?,?,?,?)", id, ctx.OrganizationID, mime, data, time.Now().Unix())
			return err
		})
	}
	if err != nil {
		rt.writeMerchError(w, err)
		return
	}
	_ = writeJSON(w, http.StatusCreated, merchMedia{ID: id, URL: merchMediaURL(ctx.OrganizationSlug, id)})
}

func (rt *_router) getMerchMedia(w http.ResponseWriter, r *http.Request, org database.Organization) {
	conn := rt.db.SQLConn()
	id := chi.URLParam(r, "id")
	token := parseBearerToken(r.Header.Get("Authorization"))
	if cookie, err := r.Cookie(adminCookieName()); err == nil {
		token = cookie.Value
	}
	session, ok := rt.getAdminSession(token)
	superadmin := strings.EqualFold(session.Role, "superadmin")
	admin := ok && (superadmin && session.OrganizationID == 0 || session.OrganizationID == org.ID && (superadmin || strings.EqualFold(session.Role, "admin")))
	query := "SELECT mime,content FROM merch_media WHERE organization_id=? AND id=?"
	if !admin {
		settings, err := loadMerchSettings(r.Context(), conn, org.ID)
		if err != nil {
			rt.writeMerchError(w, err)
			return
		}
		if !settings.Enabled {
			rt.writeMerchError(w, sql.ErrNoRows)
			return
		}
		query += ` AND EXISTS(SELECT 1 FROM merch_product_media pm JOIN merch_products p ON p.organization_id=pm.organization_id AND p.id=pm.product_id
			WHERE pm.organization_id=merch_media.organization_id AND pm.media_id=merch_media.id AND p.published=1 AND p.archived=0)`
	}
	var mime string
	var content []byte
	if err := conn.QueryRowContext(r.Context(), query, org.ID, id).Scan(&mime, &content); err != nil {
		rt.writeMerchError(w, err)
		return
	}
	w.Header().Set("Content-Type", mime)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Length", strconv.Itoa(len(content)))
	if !admin {
		w.Header().Set("Cache-Control", "public, max-age=60")
	}
	_, _ = w.Write(content)
}

func cleanUnattachedMerchMedia(ctx context.Context, conn *sql.DB, now int64) error {
	_, err := conn.ExecContext(ctx, `DELETE FROM merch_media WHERE created_at<? AND NOT EXISTS
		(SELECT 1 FROM merch_product_media pm WHERE pm.organization_id=merch_media.organization_id AND pm.media_id=merch_media.id)`, now-7*86400)
	return err
}
