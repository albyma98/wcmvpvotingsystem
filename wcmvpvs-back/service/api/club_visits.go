package api

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/gofrs/uuid"
)

// Collection never returns counters. Only the superadmin club list exposes them.
func (rt *_router) clubPublicVisit(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	var body struct {
		VisitorID string `json:"visitorId"`
		VisitID   string `json:"visitId"`
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&body) != nil || decoder.Decode(&struct{}{}) != io.EOF {
		http.Error(w, `{"error":"bad_json"}`, http.StatusBadRequest)
		return
	}
	visitor, visitorErr := uuid.FromString(body.VisitorID)
	visit, visitErr := uuid.FromString(body.VisitID)
	if visitorErr != nil || visitErr != nil || visitor == uuid.Nil || visit == uuid.Nil || len(body.VisitorID) != 36 || len(body.VisitID) != 36 {
		http.Error(w, `{"error":"invalid_visit"}`, http.StatusBadRequest)
		return
	}
	site, err := rt.store.ClubSiteBySlug(r.Context(), chi.URLParam(r, "slug"))
	if err != nil || !site.Active {
		http.Error(w, `{"error":"club_not_found"}`, http.StatusNotFound)
		return
	}
	// Scope the anonymous browser identifier to the club; store no IP or user agent.
	key := sha256.Sum256([]byte(fmt.Sprintf("%d:%s", site.ID, visitor.String())))
	if err := rt.store.RecordClubVisit(r.Context(), site.ID, visit.String(), fmt.Sprintf("%x", key)); err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Store) RecordClubVisit(ctx context.Context, siteID int64, visitID, visitorKey string) error {
	// A repeated delivery of the same page opening must not increment the total.
	_, err := s.db.ExecContext(ctx, `INSERT INTO club_visits (site_id,visit_id,visitor_key)
		SELECT id,?,? FROM club_sites WHERE id=? AND active=1
		ON CONFLICT(site_id,visit_id) DO NOTHING`, visitID, visitorKey, siteID)
	return err
}
