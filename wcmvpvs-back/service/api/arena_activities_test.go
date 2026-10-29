package api

import (
	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
	"github.com/sirupsen/logrus"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type arenaActivityAPIDB struct {
	fanEconomyAPIDB
	org, fan, calls int
	cmd             database.ArenaCommand
}

func (db *arenaActivityAPIDB) ArenaActivities(org, fan int, cmd database.ArenaCommand, now time.Time) (database.ArenaActivityState, error) {
	db.org = org
	db.fan = fan
	db.cmd = cmd
	db.calls++
	return database.ArenaActivityState{}, nil
}
func TestArenaActivityIdentityAndAdminBoundary(t *testing.T) {
	for _, tc := range []struct {
		name, method, token, body string
		org, admin, status        int
	}{{"guest read", "GET", "", "", 7, 0, 200}, {"guest mutation", "POST", "", `{"action":"claim"}`, 7, 0, 401}, {"valid owner", "POST", "valid", `{"action":"buy","key":"cap"}`, 7, 0, 200}, {"cross club", "POST", "valid", `{"action":"buy","key":"cap"}`, 8, 0, 403}, {"forged owner", "POST", "valid", `{"action":"claim","fan_id":123}`, 7, 0, 400}, {"forged clock", "POST", "valid", `{"action":"claim","now":"2099-01-01"}`, 7, 0, 400}, {"public result", "POST", "valid", `{"action":"result","confirm":true,"score":"3-0"}`, 7, 0, 403}, {"admin result", "POST", "", `{"action":"result","confirm":true,"score":"3-0"}`, 7, 1, 200}} {
		t.Run(tc.name, func(t *testing.T) {
			db := &arenaActivityAPIDB{fanEconomyAPIDB: fanEconomyAPIDB{basketballAPIDB: basketballAPIDB{fanOrg: 7}}}
			rt := &_router{db: db}
			r := httptest.NewRequest(tc.method, "/fan/arena/activities", strings.NewReader(tc.body))
			r.Header.Set("X-Fan-Session", tc.token)
			r.Header.Set("X-Device-ID", "device")
			w := httptest.NewRecorder()
			rt.arenaActivities(w, r, reqcontext.RequestContext{OrganizationID: tc.org, AdminID: tc.admin, Logger: logrus.New()})
			if w.Code != tc.status {
				t.Fatalf("status %d: %s", w.Code, w.Body.String())
			}
			if tc.status != 200 && db.calls != 0 {
				t.Fatal("unauthorized mutation reached database")
			}
			if tc.name == "valid owner" && (db.fan != 9 || db.org != 7) {
				t.Fatalf("identity %d/%d", db.org, db.fan)
			}
		})
	}
}
