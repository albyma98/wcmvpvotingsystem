package api

import (
        "encoding/json"
        "net/http"
        "strconv"
        "strings"

        "github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
        "github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
        "github.com/go-chi/chi/v5"
)

func (rt *_router) listSponsors(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
        sponsors, err := rt.db.ListSponsors()
        if err != nil {
                ctx.Logger.WithError(err).Error("cannot list sponsors")
                w.WriteHeader(http.StatusInternalServerError)
                return
        }

        type sponsorResponse struct {
                ID        int    `json:"id"`
                Slot      int    `json:"slot"`
                ImageData string `json:"image_data"`
                LinkURL   string `json:"link_url"`
        }

        response := make([]sponsorResponse, 0, len(sponsors))
        for _, sponsor := range sponsors {
                response = append(response, sponsorResponse{
                        ID:        sponsor.ID,
                        Slot:      sponsor.Slot,
                        ImageData: sponsor.ImageData,
                        LinkURL:   sponsor.LinkURL,
                })
        }

        _ = json.NewEncoder(w).Encode(response)
        ctx.Logger.WithField("count", len(response)).Info("listed sponsors")
}

func (rt *_router) upsertSponsor(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
        slot, err := strconv.Atoi(chi.URLParam(r, "slot"))
        if err != nil || slot < 1 || slot > 4 {
                w.WriteHeader(http.StatusBadRequest)
                return
        }

        var payload struct {
                ImageData string `json:"image_data"`
                LinkURL   string `json:"link_url"`
        }

        if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
                ctx.Logger.WithError(err).Warn("invalid payload while saving sponsor")
                w.WriteHeader(http.StatusBadRequest)
                return
        }

        payload.ImageData = strings.TrimSpace(payload.ImageData)

        if payload.ImageData == "" {
                ctx.Logger.Warn("missing image data while saving sponsor")
                w.WriteHeader(http.StatusBadRequest)
                return
        }

        link := strings.TrimSpace(payload.LinkURL)
        if link != "" {
                lower := strings.ToLower(link)
                if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
                        // keep link as-is
                } else if strings.Contains(link, "://") {
                        link = ""
                } else {
                        link = "https://" + link
                }
        }

        sponsor := database.Sponsor{
                Slot:      slot,
                ImageData: payload.ImageData,
                LinkURL:   link,
        }

        if err := rt.db.UpsertSponsor(sponsor); err != nil {
                ctx.Logger.WithError(err).Error("cannot upsert sponsor")
                w.WriteHeader(http.StatusInternalServerError)
                return
        }

        w.WriteHeader(http.StatusNoContent)
        ctx.Logger.WithField("slot", slot).Info("sponsor saved")
}

func (rt *_router) deleteSponsor(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
        slot, err := strconv.Atoi(chi.URLParam(r, "slot"))
        if err != nil || slot < 1 || slot > 4 {
                w.WriteHeader(http.StatusBadRequest)
                return
        }

        if err := rt.db.DeleteSponsorBySlot(slot); err != nil {
                ctx.Logger.WithError(err).Error("cannot delete sponsor")
                w.WriteHeader(http.StatusInternalServerError)
                return
        }

        w.WriteHeader(http.StatusNoContent)
        ctx.Logger.WithField("slot", slot).Info("sponsor deleted")
}
