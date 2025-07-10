package main

import (
    "database/sql"
    "encoding/json"
    "log"
    "net/http"
    "strings"
    "strconv"
)

// VoteRequest represents the JSON body for a vote
// {"player_id": 1}
type VoteRequest struct {
    PlayerID int `json:"player_id"`
}

func VoteHandler(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        matchIDStr := r.URL.Query().Get("matchId")
        if matchIDStr == "" {
            // assume matchID is part of URL: /vote/:matchId -> using a router
            // e.g. with chi or mux. For simplicity, read from URL path.
            parts := splitPath(r.URL.Path)
            if len(parts) >= 2 {
                matchIDStr = parts[len(parts)-1]
            }
        }
        matchID, err := strconv.Atoi(matchIDStr)
        if err != nil {
            http.Error(w, "invalid matchId", http.StatusBadRequest)
            return
        }

        var vr VoteRequest
        if err := json.NewDecoder(r.Body).Decode(&vr); err != nil {
            http.Error(w, "invalid body", http.StatusBadRequest)
            return
        }

        ip := r.RemoteAddr
        if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
            ip = forwarded
        }

        uuidCookie, _ := r.Cookie("uuid")
        uuid := ""
        if uuidCookie != nil {
            uuid = uuidCookie.Value
        }

        ua := r.UserAgent()

        // Insert vote if not already present
        _, err = db.Exec(`INSERT INTO votes (match_id, player_id, ip_address, user_agent, uuid)
                          VALUES ($1, $2, $3, $4, $5)
                          ON CONFLICT DO NOTHING`, matchID, vr.PlayerID, ip, ua, uuid)
        if err != nil {
            log.Println("error inserting vote:", err)
            http.Error(w, "unable to vote", http.StatusInternalServerError)
            return
        }

        w.WriteHeader(http.StatusCreated)
    }
}

// splitPath splits path into parts ignoring empty elements
func splitPath(p string) []string {
    var parts []string
    for _, s := range strings.Split(p, "/") {
        if s != "" {
            parts = append(parts, s)
        }
    }
    return parts
}
