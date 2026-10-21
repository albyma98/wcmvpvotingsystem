package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
	"github.com/sirupsen/logrus"
)

type fanNicknameDB struct {
	database.AppDatabase
	err   error
	calls int
}

func (db *fanNicknameDB) IsFanNicknameAvailable(org int, nickname string, exclude int) (bool, error) {
	db.calls++
	// All plain suggestion seeds are occupied; suffixed alternatives are free.
	return org != 7 || strings.ContainsAny(nickname, "0123456789"), db.err
}

func TestFanNicknameAvailabilityAndFilteredSuggestions(t *testing.T) {
	db := &fanNicknameDB{}
	router := &_router{db: db}
	read := func(org int, query string) (*httptest.ResponseRecorder, struct {
		Available   bool     `json:"available"`
		Suggestions []string `json:"suggestions"`
	}) {
		w := httptest.NewRecorder()
		router.getFanNicknameAvailability(w, httptest.NewRequest(http.MethodGet, "/fan/nickname"+query, nil), reqcontext.RequestContext{OrganizationID: org})
		var data struct {
			Available   bool     `json:"available"`
			Suggestions []string `json:"suggestions"`
		}
		if w.Code == http.StatusOK {
			if err := json.Unmarshal(w.Body.Bytes(), &data); err != nil {
				t.Fatal(err)
			}
		}
		return w, data
	}
	w, data := read(7, "?nickname=CurvaNord")
	if w.Code != 200 || data.Available || len(data.Suggestions) != 4 || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("occupied nickname: %d %+v", w.Code, data)
	}
	seen := map[string]bool{}
	for _, nickname := range data.Suggestions {
		available, err := db.IsFanNicknameAvailable(7, nickname, 0)
		if err != nil || !available || seen[nickname] {
			t.Fatalf("suggestion not free/unique: %q", nickname)
		}
		if _, message := validateFanNickname(nickname); message != "" {
			t.Fatalf("invalid suggestion %q: %s", nickname, message)
		}
		seen[nickname] = true
	}
	w, data = read(8, "?nickname=CurvaNord")
	if w.Code != 200 || !data.Available {
		t.Fatal("availability must be organization scoped")
	}
	w, data = read(7, "")
	if w.Code != 200 || len(data.Suggestions) != 4 {
		t.Fatal("suggestions must load before typing")
	}
	before := db.calls
	if w, _ := read(0, ""); w.Code != 400 {
		t.Fatal("missing organization accepted")
	}
	if w, _ := read(7, "?nickname=ab"); w.Code != 400 {
		t.Fatal("invalid nickname accepted")
	}
	if before != db.calls {
		t.Fatal("invalid requests should not check database")
	}
	db.err = errors.New("offline")
	if w, _ := read(7, "?nickname=CurvaNord"); w.Code != 500 {
		t.Fatal("database errors must not report availability")
	}
}

func TestFanNicknameRegistrationConflict(t *testing.T) {
	db := &fanEconomyAPIDB{registerErr: database.ErrFanNicknameTaken}
	router := &_router{db: db}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/fan/register", strings.NewReader(withFanRegistrationAcceptance(`{"phone":"+393331234567","nickname":"CurvaNord","accepted_terms":true}`)))
	r.Header.Set("X-Fan-Session", "valid")
	router.postRegisterFan(w, r, reqcontext.RequestContext{OrganizationID: 7, Logger: logrus.New()})
	if w.Code != http.StatusConflict || !strings.Contains(w.Body.String(), "NICKNAME_TAKEN") {
		t.Fatalf("conflict must allow frontend recovery: %d %s", w.Code, w.Body.String())
	}
}
