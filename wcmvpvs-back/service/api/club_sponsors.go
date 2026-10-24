package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"unicode"
)

const maxClubSponsorLogoBytes = 300 * 1024
const maxClubSponsorBodyBytes = maxClubSponsorLogoBytes + 32*1024

type ClubSponsor struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	LogoURL     string `json:"logoUrl"`
	Website     string `json:"website"`
	Description string `json:"description"`
	Position    int    `json:"position"`
	Active      bool   `json:"active"`
}

func validateClubSponsor(v *ClubSponsor) error {
	v.Name = strings.TrimSpace(v.Name)
	v.LogoURL = strings.TrimSpace(v.LogoURL)
	v.Website = strings.TrimSpace(v.Website)
	v.Description = strings.TrimSpace(v.Description)
	if v.Name == "" || len(v.Name) > 200 {
		return fmt.Errorf("invalid_sponsor_name")
	}
	if len(v.Description) > 3000 {
		return fmt.Errorf("invalid_sponsor_description")
	}
	if v.Position < 0 || v.Position > 1000000 {
		return fmt.Errorf("invalid_sponsor_position")
	}
	if !validClubSponsorURL(v.Website) {
		return fmt.Errorf("invalid_sponsor_website")
	}
	if !validClubSponsorLogo(v.LogoURL) {
		return fmt.Errorf("invalid_sponsor_logo")
	}
	return nil
}

func validClubSponsorURL(value string) bool {
	if value == "" || len(value) > 2000 || strings.ContainsAny(value, "\\") || strings.IndexFunc(value, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) }) >= 0 {
		return false
	}
	u, err := url.Parse(value)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Hostname() != "" && u.User == nil
}

func validClubSponsorLogo(value string) bool {
	if strings.HasPrefix(value, "data:") {
		if len(value) > maxClubSponsorLogoBytes {
			return false
		}
		parts := strings.SplitN(value, ",", 2)
		if len(parts) != 2 {
			return false
		}
		mimeType, ok := map[string]string{
			"data:image/png;base64": "image/png", "data:image/jpeg;base64": "image/jpeg",
			"data:image/webp;base64": "image/webp", "data:image/gif;base64": "image/gif",
		}[parts[0]]
		if !ok {
			return false
		}
		data, err := base64.StdEncoding.DecodeString(parts[1])
		return err == nil && len(data) > 0 && http.DetectContentType(data) == mimeType
	}
	if strings.HasPrefix(value, "/") && !strings.HasPrefix(value, "//") && len(value) <= 2000 && !strings.ContainsAny(value, "\\") && strings.IndexFunc(value, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) }) < 0 {
		u, err := url.Parse(value)
		return err == nil && u.Host == "" && u.Scheme == "" && u.Path != ""
	}
	return validClubSponsorURL(value)
}

func (s *Store) ListClubSponsors(ctx context.Context, siteID int64, activeOnly bool) ([]ClubSponsor, error) {
	query := `SELECT id,name,logo_url,website,description,position,active FROM club_sponsors WHERE site_id=?`
	if activeOnly {
		query += ` AND active=1`
	}
	rows, err := s.db.QueryContext(ctx, query+` ORDER BY position,id`, siteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []ClubSponsor{}
	for rows.Next() {
		var v ClubSponsor
		var active int
		if err := rows.Scan(&v.ID, &v.Name, &v.LogoURL, &v.Website, &v.Description, &v.Position, &active); err != nil {
			return nil, err
		}
		v.Active = active == 1
		items = append(items, v)
	}
	return items, rows.Err()
}

func (s *Store) SaveClubSponsor(ctx context.Context, siteID int64, v ClubSponsor) (int64, error) {
	if v.ID > 0 {
		res, err := s.db.ExecContext(ctx, `UPDATE club_sponsors SET name=?,logo_url=?,website=?,description=?,position=?,active=? WHERE id=? AND site_id=?`, v.Name, v.LogoURL, v.Website, v.Description, v.Position, boolInt(v.Active), v.ID, siteID)
		if err != nil {
			return 0, err
		}
		n, err := res.RowsAffected()
		if err != nil {
			return 0, err
		}
		if n == 0 {
			return 0, errClubNotFound
		}
		return v.ID, nil
	}
	res, err := s.db.ExecContext(ctx, `INSERT INTO club_sponsors(site_id,name,logo_url,website,description,position,active) VALUES(?,?,?,?,?,?,?)`, siteID, v.Name, v.LogoURL, v.Website, v.Description, v.Position, boolInt(v.Active))
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) DeleteClubSponsor(ctx context.Context, siteID, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM club_sponsors WHERE id=? AND site_id=?`, id, siteID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return errClubNotFound
	}
	return nil
}

func (rt *_router) clubAdminSponsors(w http.ResponseWriter, r *http.Request, siteID int64) {
	items, err := rt.store.ListClubSponsors(r.Context(), siteID, false)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, 500)
		return
	}
	writeJSON(w, 200, map[string]interface{}{"sponsors": items})
}

func (rt *_router) clubAdminSaveSponsor(w http.ResponseWriter, r *http.Request, siteID int64) {
	v := ClubSponsor{Active: true}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxClubSponsorBodyBytes))
	if decoder.Decode(&v) != nil || decoder.Decode(&struct{}{}) != io.EOF {
		http.Error(w, `{"error":"bad_json"}`, 400)
		return
	}
	// The route alone chooses the record: a body ID must never turn a POST into an update.
	v.ID = 0
	if r.Method == http.MethodPut {
		var err error
		v.ID, err = parseClubID(r, "id")
		if err != nil || v.ID <= 0 {
			http.Error(w, `{"error":"bad_id"}`, 400)
			return
		}
	}
	if err := validateClubSponsor(&v); err != nil {
		writeJSON(w, 400, map[string]string{"error": err.Error()})
		return
	}
	id, err := rt.store.SaveClubSponsor(r.Context(), siteID, v)
	if errors.Is(err, errClubNotFound) {
		http.Error(w, `{"error":"not_found"}`, 404)
		return
	}
	if err != nil {
		http.Error(w, `{"error":"save_failed"}`, 500)
		return
	}
	writeJSON(w, 200, map[string]interface{}{"id": id, "ok": true})
}

func (rt *_router) clubAdminDeleteSponsor(w http.ResponseWriter, r *http.Request, siteID int64) {
	id, err := parseClubID(r, "id")
	if err != nil || id <= 0 {
		http.Error(w, `{"error":"bad_id"}`, 400)
		return
	}
	err = rt.store.DeleteClubSponsor(r.Context(), siteID, id)
	if errors.Is(err, errClubNotFound) {
		http.Error(w, `{"error":"not_found"}`, 404)
		return
	}
	if err != nil {
		http.Error(w, `{"error":"delete_failed"}`, 500)
		return
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}
