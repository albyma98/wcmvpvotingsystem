package api

import (
	"encoding/json"
	"net/http"
)

type jsonMessage struct {
	Message string `json:"message"`
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
