package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
	"github.com/go-chi/chi/v5"
)

type quizAnswerRequest struct {
	QuestionID    int    `json:"questionId"`
	SelectedIndex int    `json:"selectedIndex"`
	ResponseMs    int    `json:"responseMs"`
	DeviceID      string `json:"deviceId"`
}

func quizInWindow(cfg database.EventQuizConfig) bool {
	now := time.Now().UTC()
	if cfg.ActiveFrom != "" {
		if t, err := time.Parse(time.RFC3339, cfg.ActiveFrom); err == nil && now.Before(t) {
			return false
		}
	}
	if cfg.ActiveTo != "" {
		if t, err := time.Parse(time.RFC3339, cfg.ActiveTo); err == nil && now.After(t) {
			return false
		}
	}
	return true
}

func (rt *_router) getPublicEventQuiz(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	eventID, err := strconv.Atoi(chi.URLParam(r, "eventId"))
	if err != nil || eventID <= 0 {
		_ = writeJSONMessage(w, http.StatusBadRequest, "Evento non valido.")
		return
	}
	cfg, err := rt.db.GetEventQuizConfig(eventID)
	if err != nil {
		if err == sql.ErrNoRows {
			_ = writeJSONMessage(w, http.StatusNotFound, "Quiz non configurato.")
			return
		}
		ctx.Logger.WithError(err).Error("quiz config")
		_ = writeJSONMessage(w, http.StatusInternalServerError, "Errore quiz.")
		return
	}
	if !cfg.Enabled {
		_ = writeJSONMessage(w, http.StatusNotFound, "Quiz disabilitato.")
		return
	}
	if !quizInWindow(cfg) {
		_ = writeJSONMessage(w, http.StatusConflict, "Quiz non disponibile ora.")
		return
	}
	questions, err := rt.db.ListEventQuizQuestions(eventID)
	if err != nil {
		ctx.Logger.WithError(err).Error("quiz questions")
		_ = writeJSONMessage(w, http.StatusInternalServerError, "Errore quiz.")
		return
	}
	if len(questions) > cfg.QuestionsPerSession {
		questions = questions[:cfg.QuestionsPerSession]
	}
	type pubQ struct {
		ID      int      `json:"id"`
		Text    string   `json:"text"`
		Answers []string `json:"answers"`
	}
	out := make([]pubQ, 0, len(questions))
	for _, q := range questions {
		out = append(out, pubQ{ID: q.ID, Text: q.QuestionText, Answers: q.Answers})
	}
	_ = writeJSON(w, http.StatusOK, map[string]interface{}{"enabled": cfg.Enabled, "config": cfg, "questions": out})
}

func (rt *_router) postPublicEventQuizAnswer(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	eventID, err := strconv.Atoi(chi.URLParam(r, "eventId"))
	if err != nil || eventID <= 0 {
		_ = writeJSONMessage(w, http.StatusBadRequest, "Evento non valido.")
		return
	}
	var req quizAnswerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		_ = writeJSONMessage(w, http.StatusBadRequest, "Payload non valido.")
		return
	}
	q, err := rt.db.GetEventQuizQuestion(eventID, req.QuestionID)
	if err != nil {
		_ = writeJSONMessage(w, http.StatusBadRequest, "Domanda non valida.")
		return
	}
	isCorrect := req.SelectedIndex == q.CorrectIndex
	cfg, _ := rt.db.GetEventQuizConfig(eventID)
	coins := 0
	if isCorrect {
		coins = cfg.BaseReward
	}
	if isCorrect && req.ResponseMs > 0 && req.ResponseMs <= 1500 {
		coins += cfg.StreakBonus
	}
	_ = writeJSON(w, http.StatusOK, map[string]interface{}{"isCorrect": isCorrect, "coinsEarned": coins})
}

func (rt *_router) getAdminEventQuiz(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	eventID, _ := strconv.Atoi(chi.URLParam(r, "eventId"))
	if !rt.ensureEventInOrganization(w, ctx, eventID) {
		return
	}
	cfg, err := rt.db.GetEventQuizConfig(eventID)
	if err == sql.ErrNoRows {
		cfg = database.EventQuizConfig{EventID: eventID, QuestionsPerSession: 5, SecondsPerQuestion: 8, BaseReward: 3, CompletionBonus: 5, StreakBonus: 1}
	} else if err != nil {
		_ = writeJSONMessage(w, http.StatusInternalServerError, "Errore quiz.")
		return
	}
	questions, _ := rt.db.ListEventQuizQuestions(eventID)
	_ = writeJSON(w, http.StatusOK, map[string]interface{}{"config": cfg, "questions_count": len(questions)})
}

func (rt *_router) putAdminEventQuiz(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	eventID, _ := strconv.Atoi(chi.URLParam(r, "eventId"))
	if !rt.ensureEventInOrganization(w, ctx, eventID) {
		return
	}
	var cfg database.EventQuizConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		_ = writeJSONMessage(w, http.StatusBadRequest, "Payload non valido.")
		return
	}
	cfg.EventID = eventID
	saved, err := rt.db.UpsertEventQuizConfig(cfg)
	if err != nil {
		_ = writeJSONMessage(w, http.StatusInternalServerError, "Salvataggio quiz fallito.")
		return
	}
	_ = writeJSON(w, http.StatusOK, saved)
}

func (rt *_router) getAdminEventQuizQuestions(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	eventID, _ := strconv.Atoi(chi.URLParam(r, "eventId"))
	if !rt.ensureEventInOrganization(w, ctx, eventID) {
		return
	}
	items, err := rt.db.ListEventQuizQuestions(eventID)
	if err != nil {
		_ = writeJSONMessage(w, http.StatusInternalServerError, "Errore.")
		return
	}
	_ = writeJSON(w, http.StatusOK, items)
}
func (rt *_router) postAdminEventQuizQuestion(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	eventID, _ := strconv.Atoi(chi.URLParam(r, "eventId"))
	if !rt.ensureEventInOrganization(w, ctx, eventID) {
		return
	}
	if _, err := rt.db.GetEventQuizConfig(eventID); err != nil {
		if err == sql.ErrNoRows {
			_, err = rt.db.UpsertEventQuizConfig(database.EventQuizConfig{
				EventID:             eventID,
				QuestionsPerSession: 5,
				SecondsPerQuestion:  8,
				BaseReward:          3,
				CompletionBonus:     5,
				StreakBonus:         1,
			})
		}
		if err != nil {
			_ = writeJSONMessage(w, http.StatusInternalServerError, "Errore.")
			return
		}
	}
	var q database.EventQuizQuestion
	if err := json.NewDecoder(r.Body).Decode(&q); err != nil {
		_ = writeJSONMessage(w, http.StatusBadRequest, "Payload non valido.")
		return
	}
	created, err := rt.db.CreateEventQuizQuestion(eventID, q)
	if err != nil {
		_ = writeJSONMessage(w, http.StatusInternalServerError, "Errore.")
		return
	}
	_ = writeJSON(w, http.StatusCreated, created)
}
func (rt *_router) putAdminEventQuizQuestion(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	eventID, _ := strconv.Atoi(chi.URLParam(r, "eventId"))
	if !rt.ensureEventInOrganization(w, ctx, eventID) {
		return
	}
	qid, _ := strconv.Atoi(chi.URLParam(r, "id"))
	var q database.EventQuizQuestion
	if err := json.NewDecoder(r.Body).Decode(&q); err != nil {
		_ = writeJSONMessage(w, http.StatusBadRequest, "Payload non valido.")
		return
	}
	updated, err := rt.db.UpdateEventQuizQuestion(eventID, qid, q)
	if err != nil {
		_ = writeJSONMessage(w, http.StatusInternalServerError, "Errore.")
		return
	}
	_ = writeJSON(w, http.StatusOK, updated)
}
func (rt *_router) deleteAdminEventQuizQuestion(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	eventID, _ := strconv.Atoi(chi.URLParam(r, "eventId"))
	if !rt.ensureEventInOrganization(w, ctx, eventID) {
		return
	}
	qid, _ := strconv.Atoi(chi.URLParam(r, "id"))
	if err := rt.db.DeleteEventQuizQuestion(eventID, qid); err != nil {
		_ = writeJSONMessage(w, http.StatusInternalServerError, "Errore.")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
