package main

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	_ "github.com/mattn/go-sqlite3"
)

var (
	db         *sql.DB
	hmacSecret = "secret-key"
)

func main() {
	var err error
	db, err = sql.Open("sqlite3", "./votes.db")
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	if err = createTable(); err != nil {
		log.Fatalf("create table: %v", err)
	}

	http.HandleFunc("/vote", voteHandler)

	log.Println("Server running on :3000")
	log.Fatal(http.ListenAndServe(":3000", nil))
}

func createTable() error {
	query := `CREATE TABLE IF NOT EXISTS votes (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        player_id INTEGER NOT NULL,
        code TEXT NOT NULL,
        signature TEXT NOT NULL
    );`
	_, err := db.Exec(query)
	return err
}

type voteRequest struct {
	PlayerID int `json:"player_id"`
}

type voteResponse struct {
	Code      string `json:"code"`
	Signature string `json:"signature"`
	QRData    string `json:"qr_data"`
}

func voteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req voteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	code := randomString(8)
	mac := hmac.New(sha256.New, []byte(hmacSecret))
	mac.Write([]byte(code))
	sig := hex.EncodeToString(mac.Sum(nil))

	if _, err := db.Exec(`INSERT INTO votes(player_id, code, signature) VALUES(?, ?, ?)`, req.PlayerID, code, sig); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	qrData := fmt.Sprintf("%s:%s", code, sig)
	resp := voteResponse{Code: code, Signature: sig, QRData: qrData}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func randomString(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	for i := range b {
		b[i] = letters[int(b[i])%len(letters)]
	}
	return string(b)
}
