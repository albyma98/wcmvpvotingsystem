package api

import (
	"encoding/json"
	"net/http"
)

type jsonMessage struct {
	Message string `json:"message"`
}

func (rt *_router) respondWithJSON(w http.ResponseWriter, status int, payload interface{}) {
	if err := writeJSON(w, status, payload); err != nil {
		rt.baseLogger.WithError(err).Error("cannot encode json response")
	}
}

func (rt *_router) decodeJSONBody(w http.ResponseWriter, r *http.Request, dest interface{}) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dest); err != nil {
		_ = writeJSONMessage(w, http.StatusBadRequest, "Richiesta non valida")
		return err
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) error {
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(status)
	if payload == nil {
		return nil
	}
	return json.NewEncoder(w).Encode(payload)
}

func writeJSONMessage(w http.ResponseWriter, status int, message string) error {
	return writeJSON(w, status, jsonMessage{Message: message})
}
