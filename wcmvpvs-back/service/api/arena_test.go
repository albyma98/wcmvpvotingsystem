package api

import (
	"encoding/json"
	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
	"github.com/sirupsen/logrus"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type arenaAPIDB struct {
	fanEconomyAPIDB
	requestedOrg int
}

func (db *arenaAPIDB) GetArenaFans(org int) ([]database.ArenaFan, error) {
	db.requestedOrg = org
	return []database.ArenaFan{{ID: 9, Nickname: "Fan"}}, nil
}
func TestArenaTenantIdentityAndPublicFields(t *testing.T) {
	for _, tc := range []struct {
		name, token     string
		org             int
		pending, wantMe bool
	}{{"own", "valid", 7, false, true}, {"guest", "", 7, false, false}, {"wrong organization", "valid", 8, false, false}, {"invalid", "invalid", 7, false, false}, {"pending", "valid", 7, true, false}} {
		t.Run(tc.name, func(t *testing.T) {
			db := &arenaAPIDB{fanEconomyAPIDB: fanEconomyAPIDB{basketballAPIDB: basketballAPIDB{fanOrg: tc.org}, registrationPending: tc.pending}}
			rt := &_router{db: db}
			req := httptest.NewRequest("GET", "/fan/arena", nil)
			req.Header.Set("X-Fan-Session", tc.token)
			req.Header.Set("X-Device-ID", "device")
			w := httptest.NewRecorder()
			rt.getArenaFans(w, req, reqcontext.RequestContext{OrganizationID: 7, Logger: logrus.New()})
			var data struct {
				Fans []database.ArenaFan `json:"fans"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &data); err != nil {
				t.Fatal(err)
			}
			if w.Code != http.StatusOK || db.requestedOrg != 7 || len(data.Fans) != 1 || data.Fans[0].IsMe != tc.wantMe {
				t.Fatalf("scope/identity: %s", w.Body.String())
			}
			for _, secret := range []string{"phone", "session_token", "accepted_terms", "device_id"} {
				if strings.Contains(w.Body.String(), secret) {
					t.Fatal("private field", secret)
				}
			}
		})
	}
	rt := &_router{}
	w := httptest.NewRecorder()
	rt.getArenaFans(w, httptest.NewRequest("GET", "/fan/arena", nil), reqcontext.RequestContext{})
	if w.Code != 400 {
		t.Fatal(w.Code)
	}
}
