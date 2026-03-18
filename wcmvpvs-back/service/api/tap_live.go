package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
	"github.com/go-chi/chi/v5"
)

const (
	tapLiveRoundDurationSec   = 10
	tapLiveCountdownSec       = 3
	tapLiveMatchWaitSec       = 20
	tapLiveWinCoins           = 30
	tapLiveLoseCoins          = 8
	tapLiveDrawCoins          = 15
	tapLiveForfeitWinnerCoins = 25
)

type tapLiveQueueEntry struct {
	MatchID         string `json:"match_id"`
	Status          string `json:"status"`
	Message         string `json:"message"`
	WaitingDeadline int64  `json:"waiting_deadline"`
}

type tapLiveSubmitPayload struct {
	MatchID string `json:"match_id"`
	Score   int    `json:"score"`
}

type tapLiveAbortPayload struct {
	MatchID string `json:"match_id"`
}

type tapLiveState struct {
	ID              int    `json:"id"`
	MatchID         string `json:"match_id"`
	Status          string `json:"status"`
	Message         string `json:"message"`
	CountdownStart  int64  `json:"countdown_start,omitempty"`
	StartAt         int64  `json:"start_at,omitempty"`
	DurationSeconds int    `json:"duration_seconds,omitempty"`
	OpponentNick    string `json:"opponent_nickname,omitempty"`
}

type tapLiveResult struct {
	MatchID       string `json:"match_id"`
	Outcome       string `json:"outcome"`
	Message       string `json:"message"`
	MyScore       int    `json:"my_score"`
	OpponentScore int    `json:"opponent_score"`
	CoinsEarned   int    `json:"coins_earned"`
	Status        string `json:"status"`
}

type tapLiveManager struct {
	mu      sync.Mutex
	waiting map[int]tapLiveWaitingFan
	active  map[int]int
}

type tapLiveWaitingFan struct {
	fanID      int
	nickname   string
	eventID    int
	orgID      int
	createdAt  time.Time
	notifiedID int
}

func newTapLiveManager() *tapLiveManager {
	return &tapLiveManager{waiting: map[int]tapLiveWaitingFan{}, active: map[int]int{}}
}

func (m *tapLiveManager) putWaiting(v tapLiveWaitingFan) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.waiting[v.fanID] = v
}

func (m *tapLiveManager) cancelWaiting(fanID int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.waiting, fanID)
}

func (m *tapLiveManager) matchFor(fanID int, eventID int, orgID int, now time.Time) (tapLiveWaitingFan, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, w := range m.waiting {
		if id == fanID {
			continue
		}
		if now.Sub(w.createdAt) > tapLiveMatchWaitSec*time.Second {
			delete(m.waiting, id)
			continue
		}
		if w.eventID == eventID && w.orgID == orgID {
			delete(m.waiting, id)
			return w, true
		}
	}
	return tapLiveWaitingFan{}, false
}

func (m *tapLiveManager) waitingFor(fanID int, eventID int, orgID int, now time.Time) (tapLiveWaitingFan, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	w, ok := m.waiting[fanID]
	if !ok {
		return tapLiveWaitingFan{}, false
	}
	if now.Sub(w.createdAt) > tapLiveMatchWaitSec*time.Second {
		delete(m.waiting, fanID)
		return tapLiveWaitingFan{}, false
	}
	if w.eventID != eventID || w.orgID != orgID {
		return tapLiveWaitingFan{}, false
	}
	return w, true
}

func (m *tapLiveManager) isBusy(fanID int, now time.Time) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.active[fanID]; ok {
		return true
	}
	w, waiting := m.waiting[fanID]
	if waiting && now.Sub(w.createdAt) > tapLiveMatchWaitSec*time.Second {
		delete(m.waiting, fanID)
		return false
	}
	return waiting
}

func (m *tapLiveManager) setActive(a int, b int, matchID int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.active[a] = matchID
	m.active[b] = matchID
}

func (m *tapLiveManager) clearActive(a int, b int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.active, a)
	delete(m.active, b)
}

func (rt *_router) getTapLiveStream(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	token := rt.fanSessionTokenFromRequest(r)
	if token == "" {
		_ = writeJSONMessage(w, http.StatusUnauthorized, "Sessione fan richiesta")
		return
	}
	me, err := rt.db.GetFanBySessionToken(token)
	if err != nil {
		_ = writeJSONMessage(w, http.StatusUnauthorized, "Sessione fan non valida")
		return
	}

	clearSSEDeadline(w)
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	flusher, ok := w.(http.Flusher)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	ch, unsub := rt.tapLiveHub.Subscribe(me.Profile.ID)
	defer unsub()
	keepalive := time.NewTicker(25 * time.Second)
	defer keepalive.Stop()
	_, _ = w.Write([]byte(": ok\n\n"))
	flusher.Flush()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-ch:
			_, _ = w.Write([]byte("event: update\ndata: {\"ok\":true}\n\n"))
			flusher.Flush()
		case <-keepalive.C:
			_, _ = w.Write([]byte(": keepalive\n\n"))
			flusher.Flush()
		}
	}
}

func (rt *_router) postTapLiveQueue(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	eventID, _ := strconv.Atoi(chiURLParam(r, "eventId"))
	if eventID <= 0 {
		_ = writeJSONMessage(w, http.StatusBadRequest, "Evento non valido")
		return
	}
	token := rt.fanSessionTokenFromRequest(r)
	if token == "" {
		_ = writeJSONMessage(w, http.StatusUnauthorized, "Solo utenti registrati")
		return
	}
	me, err := rt.db.GetFanBySessionToken(token)
	if err != nil {
		_ = writeJSONMessage(w, http.StatusUnauthorized, "Solo utenti registrati")
		return
	}
	fanID := me.Profile.ID
	now := time.Now().UTC()
	if rt.tapLive.isBusy(fanID, now) {
		_ = writeJSONMessage(w, http.StatusConflict, "Hai già una sfida live attiva o in ricerca")
		return
	}
	if existing, err := rt.db.GetOpenTapLiveMatchByFan(eventID, fanID); err == nil && existing.ID > 0 {
		_ = writeJSON(w, http.StatusOK, tapLiveQueueEntry{MatchID: existing.MatchID, Status: existing.Status, Message: "Cerchiamo un avversario…"})
		return
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		ctx.Logger.WithError(err).Error("tap live open match")
		_ = writeJSONMessage(w, http.StatusInternalServerError, "Errore interno")
		return
	}

	opp, found := rt.tapLive.matchFor(fanID, eventID, ctx.OrganizationID,now)
	if !found {
		deadline := now.Add(tapLiveMatchWaitSec * time.Second)
		rt.tapLive.putWaiting(tapLiveWaitingFan{fanID: fanID, nickname: me.Profile.Nickname, eventID: eventID, orgID: ctx.OrganizationID, createdAt: now})
		_ = writeJSON(w, http.StatusAccepted, tapLiveQueueEntry{Status: "searching", Message: "Cerchiamo un avversario…", WaitingDeadline: deadline.Unix()})
		return
	}

	matchID := fmt.Sprintf("taplive_%d_%d", time.Now().UnixNano(), fanID)
	countdownStart := now
	startAt := countdownStart.Add(tapLiveCountdownSec * time.Second)
	endAt := startAt.Add(tapLiveRoundDurationSec * time.Second)
	created, err := rt.db.CreateTapLiveMatch(eventID, ctx.OrganizationID, matchID, fanID, opp.fanID, countdownStart, startAt, endAt)
	if err != nil {
		ctx.Logger.WithError(err).Error("create tap live match")
		_ = writeJSONMessage(w, http.StatusInternalServerError, "Errore creazione sfida")
		return
	}
	rt.tapLive.setActive(fanID, opp.fanID, created.ID)
	rt.tapLiveHub.Broadcast(fanID)
	rt.tapLiveHub.Broadcast(opp.fanID)
	_ = writeJSON(w, http.StatusOK, tapLiveQueueEntry{MatchID: matchID, Status: "matched", Message: "Avversario trovato"})
}

func (rt *_router) deleteTapLiveQueue(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	token := rt.fanSessionTokenFromRequest(r)
	if token == "" {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	me, err := rt.db.GetFanBySessionToken(token)
	if err == nil {
		rt.tapLive.cancelWaiting(me.Profile.ID)
	}
	w.WriteHeader(http.StatusNoContent)
}

func (rt *_router) getTapLiveState(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	eventID, _ := strconv.Atoi(chiURLParam(r, "eventId"))
	token := rt.fanSessionTokenFromRequest(r)
	if token == "" {
		_ = writeJSONMessage(w, http.StatusUnauthorized, "Sessione fan richiesta")
		return
	}
	me, err := rt.db.GetFanBySessionToken(token)
	if err != nil {
		_ = writeJSONMessage(w, http.StatusUnauthorized, "Sessione fan non valida")
		return
	}
	m, err := rt.db.GetOpenTapLiveMatchByFan(eventID, me.Profile.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			if waiting, ok := rt.tapLive.waitingFor(me.Profile.ID, eventID, ctx.OrganizationID, time.Now().UTC()); ok {
				_ = writeJSON(w, http.StatusOK, tapLiveState{Status: "searching", Message: "Cerchiamo un avversario…", DurationSeconds: tapLiveRoundDurationSec, CountdownStart: waiting.createdAt.Unix()})
				return
			}
			_ = writeJSON(w, http.StatusOK, tapLiveState{Status: "idle"})
			return
		}
		_ = writeJSONMessage(w, http.StatusInternalServerError, "Errore interno")
		return
	}
	opNick := m.Fan1Nickname
	if m.Fan1ID == me.Profile.ID {
		opNick = m.Fan2Nickname
	}
	resp := tapLiveState{ID: m.ID, MatchID: m.MatchID, Status: m.Status, OpponentNick: opNick, DurationSeconds: tapLiveRoundDurationSec}
	if !m.CountdownStartAt.IsZero() {
		resp.CountdownStart = m.CountdownStartAt.Unix()
	}
	if !m.StartedAt.IsZero() {
		resp.StartAt = m.StartedAt.Unix()
	}
	switch m.Status {
	case "matched", "countdown":
		resp.Message = "La sfida inizia tra…"
	case "playing":
		resp.Message = "Tappa la palla più volte possibile"
	case "finished":
		resp.Message = "Tempo scaduto"
	}
	_ = writeJSON(w, http.StatusOK, resp)
}

func (rt *_router) postTapLiveSubmit(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	eventID, _ := strconv.Atoi(chiURLParam(r, "eventId"))
	token := rt.fanSessionTokenFromRequest(r)
	if token == "" {
		_ = writeJSONMessage(w, http.StatusUnauthorized, "Sessione fan richiesta")
		return
	}
	me, err := rt.db.GetFanBySessionToken(token)
	if err != nil {
		_ = writeJSONMessage(w, http.StatusUnauthorized, "Sessione fan non valida")
		return
	}
	var payload tapLiveSubmitPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		_ = writeJSONMessage(w, http.StatusBadRequest, "Payload non valido")
		return
	}
	match, err := rt.db.GetTapLiveMatchByID(payload.MatchID)
	if err != nil {
		_ = writeJSONMessage(w, http.StatusNotFound, "Sfida non trovata")
		return
	}
	if match.EventID != eventID {
		_ = writeJSONMessage(w, http.StatusForbidden, "Sfida non valida")
		return
	}
	if me.Profile.ID != match.Fan1ID && me.Profile.ID != match.Fan2ID {
		_ = writeJSONMessage(w, http.StatusForbidden, "Sfida non valida")
		return
	}
	score := payload.Score
	if score < 0 {
		score = 0
	}
	if err := rt.db.SubmitTapLiveScore(payload.MatchID, me.Profile.ID, score); err != nil {
		_ = writeJSONMessage(w, http.StatusConflict, "Punteggio già inviato")
		return
	}
	if err := rt.db.TryFinalizeTapLiveMatch(payload.MatchID); err != nil {
		ctx.Logger.WithError(err).Warn("finalize tap live")
	}
	rt.tapLiveHub.Broadcast(match.Fan1ID)
	rt.tapLiveHub.Broadcast(match.Fan2ID)
	_ = writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true})
}

func (rt *_router) postTapLiveAbort(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	token := rt.fanSessionTokenFromRequest(r)
	if token == "" {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	me, err := rt.db.GetFanBySessionToken(token)
	if err != nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	var payload tapLiveAbortPayload
	_ = json.NewDecoder(r.Body).Decode(&payload)
	_ = rt.db.AbortTapLiveMatch(payload.MatchID, me.Profile.ID)
	if m, err := rt.db.GetTapLiveMatchByID(payload.MatchID); err == nil {
		rt.tapLive.clearActive(m.Fan1ID, m.Fan2ID)
		rt.tapLiveHub.Broadcast(m.Fan1ID)
		rt.tapLiveHub.Broadcast(m.Fan2ID)
	}
	w.WriteHeader(http.StatusNoContent)
}

func (rt *_router) getTapLiveResult(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	eventID, _ := strconv.Atoi(chiURLParam(r, "eventId"))
	token := rt.fanSessionTokenFromRequest(r)
	if token == "" {
		_ = writeJSONMessage(w, http.StatusUnauthorized, "Sessione fan richiesta")
		return
	}
	me, err := rt.db.GetFanBySessionToken(token)
	if err != nil {
		_ = writeJSONMessage(w, http.StatusUnauthorized, "Sessione fan non valida")
		return
	}
	matchID := strings.TrimSpace(r.URL.Query().Get("match_id"))
	if matchID == "" {
		_ = writeJSONMessage(w, http.StatusBadRequest, "match_id mancante")
		return
	}
	m, err := rt.db.GetTapLiveMatchByID(matchID)
	if err != nil || m.EventID != eventID {
		_ = writeJSONMessage(w, http.StatusNotFound, "Sfida non trovata")
		return
	}
	if me.Profile.ID != m.Fan1ID && me.Profile.ID != m.Fan2ID {
		_ = writeJSONMessage(w, http.StatusForbidden, "Sfida non valida")
		return
	}
	myScore := m.Fan1Score
	opScore := m.Fan2Score
	myOutcome := m.Fan1Result
	msg := "Pareggio"
	if me.Profile.ID == m.Fan2ID {
		myScore = m.Fan2Score
		opScore = m.Fan1Score
		myOutcome = m.Fan2Result
	}
	coins := m.Fan1Coins
	if me.Profile.ID == m.Fan2ID {
		coins = m.Fan2Coins
	}
	switch myOutcome {
	case "win":
		msg = "Hai vinto!"
	case "lose":
		msg = "Hai perso"
	case "draw":
		msg = "Pareggio"
	case "forfeit_win":
		msg = "L’avversario ha abbandonato"
	case "forfeit_lose":
		msg = "Hai perso"
	}
	if m.Status == "finished" {
		rt.tapLive.clearActive(m.Fan1ID, m.Fan2ID)
	}
	_ = writeJSON(w, http.StatusOK, tapLiveResult{MatchID: m.MatchID, Outcome: myOutcome, Message: msg, MyScore: myScore, OpponentScore: opScore, CoinsEarned: coins, Status: m.Status})
}

func chiURLParam(r *http.Request, key string) string {
	if v := strings.TrimSpace(chi.URLParam(r, key)); v != "" {
		return v
	}
	return ""
}
