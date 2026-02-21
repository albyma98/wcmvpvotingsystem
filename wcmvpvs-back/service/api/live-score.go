package api

import (
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
	"github.com/go-chi/chi/v5"
)

const (
	defaultLiveScoreSource = "legavolley_html"
	liveScoreCacheTTL      = 4 * time.Second
)

type LiveScore struct {
	HomeName      string
	GuestName     string
	SetsHome      int
	SetsGuest     int
	CurrentSet    int
	HomePoints    int
	GuestPoints   int
	IsSetFinished bool
	UpdatedAt     time.Time
}

type liveScoreResponse struct {
	EventID       int    `json:"eventId"`
	Source        string `json:"source"`
	LiveScoreURL  string `json:"liveScoreUrl"`
	HomeName      string `json:"homeName"`
	GuestName     string `json:"guestName"`
	SetsHome      int    `json:"setsHome"`
	SetsGuest     int    `json:"setsGuest"`
	CurrentSet    int    `json:"currentSet"`
	HomePoints    int    `json:"homePoints"`
	GuestPoints   int    `json:"guestPoints"`
	IsSetFinished bool   `json:"isSetFinished"`
	UpdatedAt     string `json:"updatedAt"`
}

type cachedLiveScore struct {
	Payload   liveScoreResponse
	ExpiresAt time.Time
}

var spanNumberRegex = regexp.MustCompile(`(?is)<span[^>]*>\s*(\d+)\s*</span>`)

func (rt *_router) getPublicEventLiveScore(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	eventID, err := strconv.Atoi(chi.URLParam(r, "eventId"))
	if err != nil || eventID <= 0 {
		_ = writeJSONMessage(w, http.StatusBadRequest, "ID evento non valido")
		return
	}
	event, err := rt.db.GetEventByID(eventID)
	if err != nil {
		_ = writeJSONMessage(w, http.StatusNotFound, "Evento non trovato")
		return
	}
	liveScoreURL := strings.TrimSpace(event.LiveScoreURL)
	if liveScoreURL == "" {
		_ = writeJSONMessage(w, http.StatusConflict, "Live score non configurato")
		return
	}
	if cached, ok := rt.getCachedLiveScore(eventID); ok {
		_ = writeJSON(w, http.StatusOK, cached)
		return
	}
	score, err := FetchAndParseLegavolley(liveScoreURL)
	if err != nil {
		ctx.Logger.WithError(err).WithField("event_id", eventID).Warn("cannot fetch live score")
		_ = writeJSONMessage(w, http.StatusBadGateway, "Impossibile recuperare il live score")
		return
	}
	source := strings.TrimSpace(event.LiveScoreSource)
	if source == "" {
		source = defaultLiveScoreSource
	}
	payload := liveScoreResponse{EventID: eventID, Source: source, LiveScoreURL: liveScoreURL, HomeName: score.HomeName, GuestName: score.GuestName, SetsHome: score.SetsHome, SetsGuest: score.SetsGuest, CurrentSet: score.CurrentSet, HomePoints: score.HomePoints, GuestPoints: score.GuestPoints, IsSetFinished: score.IsSetFinished, UpdatedAt: score.UpdatedAt.UTC().Format(time.RFC3339)}
	rt.setCachedLiveScore(eventID, payload)
	_ = writeJSON(w, http.StatusOK, payload)
}

func (rt *_router) getCachedLiveScore(eventID int) (liveScoreResponse, bool) { /* unchanged */
	now := time.Now()
	rt.liveScoreCacheMu.RLock()
	entry, ok := rt.liveScoreCache[eventID]
	rt.liveScoreCacheMu.RUnlock()
	if !ok || now.After(entry.ExpiresAt) {
		if ok {
			rt.liveScoreCacheMu.Lock()
			delete(rt.liveScoreCache, eventID)
			rt.liveScoreCacheMu.Unlock()
		}
		return liveScoreResponse{}, false
	}
	return entry.Payload, true
}
func (rt *_router) setCachedLiveScore(eventID int, payload liveScoreResponse) {
	rt.liveScoreCacheMu.Lock()
	rt.liveScoreCache[eventID] = cachedLiveScore{Payload: payload, ExpiresAt: time.Now().Add(liveScoreCacheTTL)}
	rt.liveScoreCacheMu.Unlock()
}

func FetchAndParseLegavolley(url string) (LiveScore, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return LiveScore{}, err
	}
	req.Header.Set("User-Agent", "wcmvpvs-live-score-bot/1.0")
	res, err := client.Do(req)
	if err != nil {
		return LiveScore{}, err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode > 299 {
		return LiveScore{}, fmt.Errorf("unexpected status code %d", res.StatusCode)
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return LiveScore{}, err
	}
	html := string(body)

	homeName := extractTeamName(html, "column1")
	guestName := extractTeamName(html, "column2")
	setsHome, err := extractSetWon(html, "column1")
	if err != nil {
		return LiveScore{}, err
	}
	setsGuest, err := extractSetWon(html, "column2")
	if err != nil {
		return LiveScore{}, err
	}
	homeSetPoints := extractNumericSpansByID(html, "DIV_Home_SetResults")
	guestSetPoints := extractNumericSpansByID(html, "DIV_Guest_SetResults")
	if len(homeSetPoints) == 0 || len(guestSetPoints) == 0 {
		return LiveScore{}, errors.New("missing set points")
	}
	last := minInt(len(homeSetPoints), len(guestSetPoints)) - 1
	currentSet, homePoints, guestPoints := last+1, homeSetPoints[last], guestSetPoints[last]
	target := 25
	if currentSet == 5 {
		target = 15
	}
	isFinished := (homePoints >= target || guestPoints >= target) && absInt(homePoints-guestPoints) >= 2
	return LiveScore{HomeName: homeName, GuestName: guestName, SetsHome: setsHome, SetsGuest: setsGuest, CurrentSet: currentSet, HomePoints: homePoints, GuestPoints: guestPoints, IsSetFinished: isFinished, UpdatedAt: time.Now()}, nil
}

func extractTeamName(html, columnClass string) string {
	r := regexp.MustCompile(`(?is)<[^>]*id=["']PTS["'][^>]*>.*?<[^>]*class=["'][^"']*` + columnClass + `[^"']*["'][^>]*>.*?<[^>]*class=["'][^"']*team_name[^"']*["'][^>]*>.*?<strong[^>]*>\s*([^<]+)\s*</strong>`)
	m := r.FindStringSubmatch(html)
	if len(m) < 2 {
		return ""
	}
	return strings.TrimSpace(m[1])
}

func extractSetWon(html, columnClass string) (int, error) {
	r := regexp.MustCompile(`(?is)<[^>]*id=["']PTS["'][^>]*>.*?<[^>]*class=["'][^"']*` + columnClass + `[^"']*score[^"']*["'][^>]*>.*?<[^>]*class=["'][^"']*set_won[^"']*["'][^>]*>\s*(\d+)\s*</`)
	m := r.FindStringSubmatch(html)
	if len(m) < 2 {
		return 0, errors.New("set_won not found")
	}
	return strconv.Atoi(m[1])
}

func extractNumericSpansByID(html, id string) []int {
	r := regexp.MustCompile(`(?is)<[^>]*id=["']` + id + `["'][^>]*>(.*?)</[^>]+>`)
	m := r.FindStringSubmatch(html)
	if len(m) < 2 {
		return nil
	}
	vals := make([]int, 0)
	for _, sm := range spanNumberRegex.FindAllStringSubmatch(m[1], -1) {
		if len(sm) < 2 {
			continue
		}
		if v, err := strconv.Atoi(sm[1]); err == nil {
			vals = append(vals, v)
		}
	}
	return vals
}
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
func absInt(v int) int { return int(math.Abs(float64(v))) }
