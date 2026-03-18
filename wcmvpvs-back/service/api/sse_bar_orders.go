package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
)

func barOrdersSSEKey(organizationID, partnerID int) int {
	if organizationID < 0 {
		organizationID = 0
	}
	if partnerID < 0 {
		partnerID = 0
	}
	return organizationID*1000000 + partnerID
}

func (rt *_router) notifyBarOrdersChanged(organizationID, partnerID int) {
	if rt.barOrdersHub == nil || organizationID < 0 {
		return
	}
	rt.barOrdersHub.Broadcast(barOrdersSSEKey(organizationID, 0))
	if partnerID > 0 {
		rt.barOrdersHub.Broadcast(barOrdersSSEKey(organizationID, partnerID))
	}
}

func (rt *_router) streamAdminBarOrders(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	partnerID := rt.resolveBarPartnerScope(ctx, r)
	if partnerID < 0 {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		_ = writeJSONMessage(w, http.StatusInternalServerError, "Streaming non supportato.")
		return
	}

	clearSSEDeadline(w)

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, ": connected\n\n")
	flusher.Flush()

	ch, unsub := rt.barOrdersHub.Subscribe(barOrdersSSEKey(ctx.OrganizationID, partnerID))
	defer unsub()

	keepalive := time.NewTicker(25 * time.Second)
	defer keepalive.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ch:
			fmt.Fprintf(w, "data: {}\n\n")
			flusher.Flush()
		case <-keepalive.C:
			fmt.Fprintf(w, ": keepalive\n\n")
			flusher.Flush()
		}
	}
}
