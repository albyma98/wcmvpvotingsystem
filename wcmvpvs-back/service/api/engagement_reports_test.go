package api

import (
	"bytes"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	imagejpeg "image/jpeg"
	imagepng "image/png"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"
)

type engagementReportFixture struct {
	router *_router
	conn   *sql.DB
	org    database.Organization
	other  database.Organization
	event  int
	ctx    reqcontext.RequestContext
}

func newEngagementReportFixture(t *testing.T) engagementReportFixture {
	t.Helper()
	conn, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	conn.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = conn.Close() })
	db, err := database.New(conn)
	if err != nil {
		t.Fatal(err)
	}
	org, err := db.CreateOrganization(database.Organization{Name: "Report Home", Slug: "report-home", IsActive: true})
	if err != nil {
		t.Fatal(err)
	}
	other, err := db.CreateOrganization(database.Organization{Name: "Report Away", Slug: "report-away", IsActive: true})
	if err != nil {
		t.Fatal(err)
	}
	event, err := db.CreateEvent(database.Event{OrganizationID: org.ID, Team1ID: org.TeamID, Team2ID: other.TeamID, StartDateTime: "2026-10-21T19:30:00Z", Location: "Report Arena"})
	if err != nil {
		t.Fatal(err)
	}
	router := &_router{db: db, baseLogger: logrus.New()}
	if err := router.ensureEngagementReportTables(); err != nil {
		t.Fatal(err)
	}
	return engagementReportFixture{router, conn, org, other, event, reqcontext.RequestContext{OrganizationID: org.ID, AdminID: 77, AdminUsername: "report-reviewer", AdminRole: "admin", Logger: logrus.New()}}
}

func engagementReportRequest(t *testing.T, handler httpRouterHandler, method string, event int, payload interface{}, ctx reqcontext.RequestContext, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	var body []byte
	if payload != nil {
		var err error
		body, err = json.Marshal(payload)
		if err != nil {
			t.Fatal(err)
		}
	}
	request := httptest.NewRequest(method, "/"+strconv.Itoa(event), bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	for name, value := range headers {
		request.Header.Set(name, value)
	}
	mux := chi.NewRouter()
	wrapped := func(w http.ResponseWriter, r *http.Request) { handler(w, r, ctx) }
	switch method {
	case http.MethodGet:
		mux.Get("/{eventId}", wrapped)
	case http.MethodPost:
		mux.Post("/{eventId}", wrapped)
	case http.MethodPut:
		mux.Put("/{eventId}", wrapped)
	default:
		t.Fatalf("unsupported report test method %s", method)
	}
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	return response
}

func engagementReportRequireStatus(t *testing.T, response *httptest.ResponseRecorder, want int) {
	t.Helper()
	if response.Code != want {
		t.Fatalf("status=%d, want=%d, body=%s", response.Code, want, response.Body.String())
	}
}

func engagementReportDecodeSnapshot(t *testing.T, response *httptest.ResponseRecorder) engagementReportSnapshot {
	t.Helper()
	engagementReportRequireStatus(t, response, http.StatusOK)
	var snapshot engagementReportSnapshot
	if err := json.Unmarshal(response.Body.Bytes(), &snapshot); err != nil {
		t.Fatal(err)
	}
	return snapshot
}

func engagementReportSignalPayload(id, name string, metadata map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{"id": id, "name": name, "session_id": "report-session", "occurred_at": time.Now().UTC().Add(-time.Minute).Format(time.RFC3339Nano), "metadata": metadata}
}

func engagementReportFloat(value float64) *float64 { return &value }

func engagementReportImage(t *testing.T, format string, width, height int) string {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	img.Set(0, 0, color.RGBA{R: 100, G: 40, B: 200, A: 255})
	var out bytes.Buffer
	var err error
	if format == "jpeg" {
		err = imagejpeg.Encode(&out, img, &imagejpeg.Options{Quality: 85})
	} else {
		err = imagepng.Encode(&out, img)
	}
	if err != nil {
		t.Fatal(err)
	}
	return "data:image/" + format + ";base64," + base64.StdEncoding.EncodeToString(out.Bytes())
}

func TestEngagementReportPrepareExcludesLegacyHistoryAggregatesAndPreservesMissingData(t *testing.T) {
	f := newEngagementReportFixture(t)
	// A populated legacy history must not manufacture observations for /new.
	if _, err := f.conn.Exec(`INSERT INTO tracking_events(event_id,organization_id,device_id,event_name,metadata_json) VALUES(?,?,?,?,?)`, f.event, f.org.ID, "legacy-browser", "page_view", `{"seconds":900}`); err != nil {
		t.Fatal(err)
	}
	snapshot := engagementReportDecodeSnapshot(t, engagementReportRequest(t, f.router.prepareEngagementReport, http.MethodGet, f.event, nil, f.ctx, nil))
	if len(snapshot.KPIs) != 30 || len(snapshot.SelectedKPIIDs) != 20 || len(snapshot.Facts) == 0 || snapshot.SchemaVersion != 1 {
		t.Fatalf("incomplete report: %+v", snapshot)
	}
	for key, value := range snapshot.Facts {
		if value != nil {
			t.Errorf("missing fact %s fabricated value %v", key, *value)
		}
	}
	for _, kpi := range snapshot.KPIs {
		if kpi.Value != nil {
			t.Errorf("missing KPI %s must remain unavailable", kpi.ID)
		}
	}
	if snapshot.MeasurementStartedAt != "" || snapshot.Branding.CompanyName != "ArenaBoostX" || snapshot.Event.OrganizationName != f.org.Name || !strings.Contains(snapshot.Event.Title, f.org.Name) || snapshot.Event.Location != "Report Arena" {
		t.Fatalf("wrong empty-report metadata: %+v", snapshot)
	}
	var stored string
	if err := f.conn.QueryRow(`SELECT payload FROM engagement_report_calculations WHERE id=? AND admin_id=?`, snapshot.CalculationID, f.ctx.AdminID).Scan(&stored); err != nil || !strings.Contains(stored, snapshot.CalculationID) {
		t.Fatalf("calculation not persisted: %s, %v", stored, err)
	}
}

func TestEngagementReportBrandingRequiresRealBoundedImagesAndMasterRole(t *testing.T) {
	f := newEngagementReportFixture(t)
	master := f.ctx
	master.AdminRole = "superadmin"
	for _, handler := range []httpRouterHandler{f.router.getReportBranding, f.router.putReportBranding} {
		response := engagementReportRequest(t, handler, http.MethodPut, f.event, reportBranding{CompanyName: "Changed"}, f.ctx, nil)
		engagementReportRequireStatus(t, response, http.StatusForbidden)
	}
	for _, format := range []string{"png", "jpeg"} {
		logo := engagementReportImage(t, format, 12, 6)
		response := engagementReportRequest(t, f.router.putReportBranding, http.MethodPut, f.event, reportBranding{CompanyName: "  ArenaBoostX Italia  ", LogoData: logo}, master, nil)
		engagementReportRequireStatus(t, response, http.StatusOK)
		var saved reportBranding
		if err := json.Unmarshal(response.Body.Bytes(), &saved); err != nil || saved.CompanyName != "ArenaBoostX Italia" || saved.LogoData != logo {
			t.Fatalf("branding round trip: %+v, %v", saved, err)
		}
	}
	validPNG := engagementReportImage(t, "png", 8, 8)
	pngBytes, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(validPNG, "data:image/png;base64,"))
	if err != nil {
		t.Fatal(err)
	}
	truncatedPNG := "data:image/png;base64," + base64.StdEncoding.EncodeToString(pngBytes[:len(pngBytes)-16])
	for _, tc := range []struct {
		name string
		data reportBranding
	}{
		{"empty name", reportBranding{CompanyName: " "}},
		{"multiline name", reportBranding{CompanyName: "Arena\nBoostX"}},
		{"long name", reportBranding{CompanyName: strings.Repeat("x", 121)}},
		{"not image", reportBranding{CompanyName: "ArenaBoostX", LogoData: "data:image/png;base64,aGVsbG8="}},
		{"svg", reportBranding{CompanyName: "ArenaBoostX", LogoData: "data:image/svg+xml;base64,PHN2Zz4="}},
		{"mismatched MIME", reportBranding{CompanyName: "ArenaBoostX", LogoData: strings.Replace(validPNG, "image/png", "image/jpeg", 1)}},
		{"bad base64", reportBranding{CompanyName: "ArenaBoostX", LogoData: "data:image/png;base64,!!!!"}},
		{"damaged PNG body", reportBranding{CompanyName: "ArenaBoostX", LogoData: truncatedPNG}},
		{"dimension limit", reportBranding{CompanyName: "ArenaBoostX", LogoData: engagementReportImage(t, "png", 2049, 1)}},
		{"encoded size", reportBranding{CompanyName: "ArenaBoostX", LogoData: "data:image/png;base64," + strings.Repeat("A", 512*1024)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			engagementReportRequireStatus(t, engagementReportRequest(t, f.router.putReportBranding, http.MethodPut, f.event, tc.data, master, nil), http.StatusBadRequest)
		})
	}
	engagementReportRequireStatus(t, engagementReportRequest(t, f.router.putReportBranding, http.MethodPut, f.event, reportBranding{CompanyName: "ArenaBoostX", LogoData: ""}, master, nil), http.StatusOK)
	brand, err := f.router.loadReportBranding()
	if err != nil || brand.LogoData != "" {
		t.Fatalf("explicit logo removal was not persisted: %+v, %v", brand, err)
	}
}

func TestEngagementReportSignalsAreAtomicIdempotentAndExcludeUnknownMetadata(t *testing.T) {
	f := newEngagementReportFixture(t)
	headers := map[string]string{"X-Device-ID": "private-browser-id"}
	view := engagementReportSignalPayload("view-one", "page_view", map[string]interface{}{"vote_available": true, "email": "private@example.com", "phone": "+391234567", "fan_id": 999, "arbitrary": "discard", "source": "arena"})
	payload := map[string]interface{}{"events": []interface{}{view}}
	for i := 0; i < 2; i++ {
		engagementReportRequireStatus(t, engagementReportRequest(t, f.router.recordReportSignals, http.MethodPost, f.event, payload, f.ctx, headers), http.StatusNoContent)
	}
	var count int
	var metadata, device string
	var fanID int
	if err := f.conn.QueryRow(`SELECT COUNT(*),metadata,device_key,fan_id FROM engagement_report_signals WHERE event_id=?`, f.event).Scan(&count, &metadata, &device, &fanID); err != nil {
		t.Fatal(err)
	}
	if count != 1 || device == headers["X-Device-ID"] || len(device) != 64 || fanID != 0 || strings.Contains(metadata, "private") || strings.Contains(metadata, "fan_id") || strings.Contains(metadata, "arbitrary") || !strings.Contains(metadata, `"vote_available":true`) {
		t.Fatalf("wrong signal storage: count=%d device=%q fan=%d metadata=%s", count, device, fanID, metadata)
	}
	invalid := engagementReportSignalPayload("invalid", "unknown_signal", nil)
	batch := map[string]interface{}{"events": []interface{}{engagementReportSignalPayload("atomic-valid", "page_view", nil), invalid}}
	engagementReportRequireStatus(t, engagementReportRequest(t, f.router.recordReportSignals, http.MethodPost, f.event, batch, f.ctx, headers), http.StatusBadRequest)
	if err := f.conn.QueryRow(`SELECT COUNT(*) FROM engagement_report_signals WHERE event_id=?`, f.event).Scan(&count); err != nil || count != 1 {
		t.Fatalf("invalid batch was partially persisted: count=%d, %v", count, err)
	}
	spoof := engagementReportSignalPayload("spoof", "page_view", nil)
	spoof["fan_id"] = 999
	engagementReportRequireStatus(t, engagementReportRequest(t, f.router.recordReportSignals, http.MethodPost, f.event, map[string]interface{}{"events": []interface{}{spoof}}, f.ctx, headers), http.StatusBadRequest)
	engagementReportRequireStatus(t, engagementReportRequest(t, f.router.recordReportSignals, http.MethodPost, f.event, payload, f.ctx, nil), http.StatusBadRequest)
	foreignSponsor, err := f.conn.Exec(`INSERT INTO sponsors(organization_id,position,name,logo_data) VALUES(?,1,'Foreign sponsor','logo')`, f.other.ID)
	if err != nil {
		t.Fatal(err)
	}
	sponsorID, err := foreignSponsor.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	sponsorView := map[string]interface{}{"events": []interface{}{engagementReportSignalPayload("foreign-sponsor", "sponsor_view", map[string]interface{}{"sponsor_id": sponsorID})}}
	engagementReportRequireStatus(t, engagementReportRequest(t, f.router.recordReportSignals, http.MethodPost, f.event, sponsorView, f.ctx, headers), http.StatusBadRequest)
}

func TestEngagementReportRejectsCrossTenantAccessAndPendingRegistration(t *testing.T) {
	f := newEngagementReportFixture(t)
	foreign := f.ctx
	foreign.OrganizationID = f.other.ID
	for _, handler := range []httpRouterHandler{f.router.prepareEngagementReport, f.router.approveEngagementReport, f.router.recordReportSignals} {
		engagementReportRequireStatus(t, engagementReportRequest(t, handler, http.MethodPost, f.event, map[string]interface{}{}, foreign, map[string]string{"X-Device-ID": "browser"}), http.StatusNotFound)
	}
	pending, err := f.router.db.CreateFanWithPhoneE164(f.org.ID, "+393331234567")
	if err != nil {
		t.Fatal(err)
	}
	if err := f.router.db.UpsertFanSession("pending-report-token", pending.ID, "browser"); err != nil {
		t.Fatal(err)
	}
	headers := map[string]string{"X-Device-ID": "browser", "X-Fan-Session": "pending-report-token"}
	completed := map[string]interface{}{"events": []interface{}{engagementReportSignalPayload("registration", "registration_completed", nil)}}
	engagementReportRequireStatus(t, engagementReportRequest(t, f.router.recordReportSignals, http.MethodPost, f.event, completed, f.ctx, headers), http.StatusBadRequest)
	view := map[string]interface{}{"events": []interface{}{engagementReportSignalPayload("pending-view", "page_view", nil)}}
	engagementReportRequireStatus(t, engagementReportRequest(t, f.router.recordReportSignals, http.MethodPost, f.event, view, f.ctx, headers), http.StatusNoContent)
	var fanID int
	if err := f.conn.QueryRow(`SELECT fan_id FROM engagement_report_signals WHERE event_id=? AND signal_id='pending-view'`, f.event).Scan(&fanID); err != nil || fanID != 0 {
		t.Fatalf("pending profile counted as identified: fan=%d, %v", fanID, err)
	}
	foreignFan, err := f.router.db.RegisterFan(database.FanRegisterInput{OrganizationID: f.other.ID, Nickname: "ForeignReportFan", Phone: "+393331234568", AcceptedTerms: true, SessionToken: "foreign-report-token", DeviceID: "original-foreign-device"})
	if err != nil {
		t.Fatal(err)
	}
	if foreignFan.Profile.RegistrationPending {
		t.Fatal("fixture foreign fan must be fully registered")
	}
	foreignView := map[string]interface{}{"events": []interface{}{engagementReportSignalPayload("foreign-token-view", "page_view", nil)}}
	engagementReportRequireStatus(t, engagementReportRequest(t, f.router.recordReportSignals, http.MethodPost, f.event, foreignView, f.ctx, map[string]string{"X-Device-ID": "another-browser", "X-Fan-Session": "foreign-report-token"}), http.StatusNoContent)
	var foreignDevice string
	if err := f.conn.QueryRow(`SELECT device_id FROM fan_sessions WHERE token='foreign-report-token'`).Scan(&foreignDevice); err != nil || foreignDevice != "original-foreign-device" {
		t.Fatalf("wrong-organization token mutated a foreign session: device=%q, %v", foreignDevice, err)
	}
}

func TestEngagementReportApprovalRecalculatesAndStoresImmutableVersionedSnapshots(t *testing.T) {
	f := newEngagementReportFixture(t)
	master := f.ctx
	master.AdminRole = "superadmin"
	firstLogo := engagementReportImage(t, "png", 12, 6)
	engagementReportRequireStatus(t, engagementReportRequest(t, f.router.putReportBranding, http.MethodPut, f.event, reportBranding{CompanyName: "Original issuer", LogoData: firstLogo}, master, nil), http.StatusOK)
	payload := map[string]interface{}{"events": []interface{}{
		engagementReportSignalPayload("view", "page_view", map[string]interface{}{"vote_available": true}),
		engagementReportSignalPayload("vote", "vote_completed", nil),
	}}
	engagementReportRequireStatus(t, engagementReportRequest(t, f.router.recordReportSignals, http.MethodPost, f.event, payload, f.ctx, map[string]string{"X-Device-ID": "browser"}), http.StatusNoContent)
	draft := engagementReportDecodeSnapshot(t, engagementReportRequest(t, f.router.prepareEngagementReport, http.MethodGet, f.event, nil, f.ctx, nil))
	var originalDraft, originalSignals string
	if err := f.conn.QueryRow(`SELECT payload FROM engagement_report_calculations WHERE id=?`, draft.CalculationID).Scan(&originalDraft); err != nil {
		t.Fatal(err)
	}
	if err := f.conn.QueryRow(`SELECT group_concat(signal_id || metadata) FROM engagement_report_signals WHERE event_id=?`, f.event).Scan(&originalSignals); err != nil {
		t.Fatal(err)
	}
	// Changing the platform identity after calculation must not alter its signed version.
	engagementReportRequireStatus(t, engagementReportRequest(t, f.router.putReportBranding, http.MethodPut, f.event, reportBranding{CompanyName: "New issuer"}, master, nil), http.StatusOK)
	approval := map[string]interface{}{"calculation_id": draft.CalculationID, "format": "extended", "facts": map[string]*float64{"visitors": engagementReportFloat(2), "engagement_eligible_visitors": engagementReportFloat(2)}, "notes": "Presenze digitali verificate dalla società"}
	approved := engagementReportDecodeSnapshot(t, engagementReportRequest(t, f.router.approveEngagementReport, http.MethodPost, f.event, approval, f.ctx, nil))
	if approved.ID == "" || approved.ApprovedBy != f.ctx.AdminUsername || approved.SchemaVersion != 1 || approved.Branding.CompanyName != "Original issuer" || approved.Branding.LogoData != firstLogo || approved.OriginalFacts["visitors"] == nil || *approved.OriginalFacts["visitors"] != 1 || *approved.Facts["visitors"] != 2 {
		t.Fatalf("wrong approved snapshot: %+v", approved)
	}
	found := false
	for _, kpi := range approved.KPIs {
		if kpi.ID == "active_participation" {
			found = true
			if kpi.Value == nil || *kpi.Value != 50 || !kpi.Corrected || kpi.Source != "manual" || kpi.Note != approval["notes"] {
				t.Fatalf("ratio not recalculated and annotated: %+v", kpi)
			}
		}
	}
	if !found {
		t.Fatal("missing active participation KPI")
	}
	replay := engagementReportDecodeSnapshot(t, engagementReportRequest(t, f.router.approveEngagementReport, http.MethodPost, f.event, approval, f.ctx, nil))
	if replay.ID != approved.ID || replay.ApprovedAt != approved.ApprovedAt {
		t.Fatal("transport retry created another version")
	}
	approval["facts"] = map[string]*float64{"visitors": engagementReportFloat(3)}
	revision := engagementReportDecodeSnapshot(t, engagementReportRequest(t, f.router.approveEngagementReport, http.MethodPost, f.event, approval, f.ctx, nil))
	if revision.ID == approved.ID || *revision.Facts["visitors"] != 3 {
		t.Fatal("changed review did not create its own immutable version")
	}
	var storedDraft, storedSignals, storedApproved string
	if err := f.conn.QueryRow(`SELECT payload FROM engagement_report_calculations WHERE id=?`, draft.CalculationID).Scan(&storedDraft); err != nil {
		t.Fatal(err)
	}
	if err := f.conn.QueryRow(`SELECT group_concat(signal_id || metadata) FROM engagement_report_signals WHERE event_id=?`, f.event).Scan(&storedSignals); err != nil {
		t.Fatal(err)
	}
	if err := f.conn.QueryRow(`SELECT payload FROM engagement_report_snapshots WHERE id=?`, approved.ID).Scan(&storedApproved); err != nil {
		t.Fatal(err)
	}
	var original engagementReportSnapshot
	if err := json.Unmarshal([]byte(storedApproved), &original); err != nil || *original.Facts["visitors"] != 2 || original.Branding.CompanyName != "Original issuer" || storedDraft != originalDraft || storedSignals != originalSignals {
		t.Fatalf("approval mutated original data: %v", err)
	}
	latest := engagementReportDecodeSnapshot(t, engagementReportRequest(t, f.router.prepareEngagementReport, http.MethodGet, f.event, nil, f.ctx, nil))
	if latest.LatestReport == nil || latest.LatestReport.ID != revision.ID || latest.Branding.CompanyName != "New issuer" || *latest.Facts["visitors"] != 1 {
		t.Fatalf("reopen did not separate latest approved and freshly measured: %+v", latest)
	}
}

func TestEngagementReportApprovalRejectsInvalidCorrectionsAndOtherReviewers(t *testing.T) {
	f := newEngagementReportFixture(t)
	draft := engagementReportDecodeSnapshot(t, engagementReportRequest(t, f.router.prepareEngagementReport, http.MethodGet, f.event, nil, f.ctx, nil))
	for _, tc := range []struct {
		name      string
		facts     map[string]*float64
		overrides map[string]reportOverride
		notes     string
	}{
		{"unknown fact", map[string]*float64{"invented": engagementReportFloat(1)}, nil, "Review"},
		{"negative fact", map[string]*float64{"visitors": engagementReportFloat(-1)}, nil, "Review"},
		{"fractional fact", map[string]*float64{"visitors": engagementReportFloat(1.5)}, nil, "Review"},
		{"ratio audience mismatch", map[string]*float64{"visitors": engagementReportFloat(1), "active_visitors": engagementReportFloat(2)}, nil, "Review"},
		{"unknown KPI", nil, map[string]reportOverride{"invented": {Value: engagementReportFloat(1), Note: "Review"}}, ""},
		{"fractional count override", nil, map[string]reportOverride{"visitors": {Value: engagementReportFloat(1.5), Note: "Review"}}, ""},
		{"over 100 percent", nil, map[string]reportOverride{"active_participation": {Value: engagementReportFloat(101), Note: "Review"}}, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := map[string]interface{}{"calculation_id": draft.CalculationID, "format": "one_page", "facts": tc.facts, "overrides": tc.overrides, "notes": tc.notes}
			engagementReportRequireStatus(t, engagementReportRequest(t, f.router.approveEngagementReport, http.MethodPost, f.event, input, f.ctx, nil), http.StatusBadRequest)
		})
	}
	otherReviewer := f.ctx
	otherReviewer.AdminID++
	input := map[string]interface{}{"calculation_id": draft.CalculationID, "format": "one_page"}
	engagementReportRequireStatus(t, engagementReportRequest(t, f.router.approveEngagementReport, http.MethodPost, f.event, input, otherReviewer, nil), http.StatusNotFound)
	var count int
	if err := f.conn.QueryRow(`SELECT COUNT(*) FROM engagement_report_snapshots`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("invalid review persisted a snapshot: count=%d %v", count, err)
	}
	input["overrides"] = map[string]reportOverride{"active_participation": {Value: nil, Note: "Misurazione non confermata"}}
	approved := engagementReportDecodeSnapshot(t, engagementReportRequest(t, f.router.approveEngagementReport, http.MethodPost, f.event, input, f.ctx, nil))
	for _, kpi := range approved.KPIs {
		if kpi.ID == "active_participation" && (kpi.Value != nil || !kpi.Corrected || kpi.Note == "") {
			t.Fatalf("explicit unavailable override lost its audit note: %+v", kpi)
		}
	}
}

func TestEngagementReportSignalBatchLimitsAndUntrustedIdentityFields(t *testing.T) {
	f := newEngagementReportFixture(t)
	headers := map[string]string{"X-Device-ID": "browser"}
	for _, amount := range []int{0, 21} {
		var signals []interface{}
		for i := 0; i < amount; i++ {
			signals = append(signals, engagementReportSignalPayload(fmt.Sprintf("signal-%d", i), "page_view", nil))
		}
		engagementReportRequireStatus(t, engagementReportRequest(t, f.router.recordReportSignals, http.MethodPost, f.event, map[string]interface{}{"events": signals}, f.ctx, headers), http.StatusBadRequest)
	}
	for _, field := range []string{"fan_id", "device_id", "organization_id"} {
		input := map[string]interface{}{"events": []interface{}{engagementReportSignalPayload("identity", "page_view", nil)}, field: "spoofed"}
		engagementReportRequireStatus(t, engagementReportRequest(t, f.router.recordReportSignals, http.MethodPost, f.event, input, f.ctx, headers), http.StatusBadRequest)
	}
	for _, offset := range []time.Duration{-31 * 24 * time.Hour, 10 * time.Minute} {
		signal := engagementReportSignalPayload("invalid-clock", "page_view", nil)
		signal["occurred_at"] = time.Now().UTC().Add(offset).Format(time.RFC3339Nano)
		engagementReportRequireStatus(t, engagementReportRequest(t, f.router.recordReportSignals, http.MethodPost, f.event, map[string]interface{}{"events": []interface{}{signal}}, f.ctx, headers), http.StatusBadRequest)
	}
}

func TestEngagementReportPurgeRemovesDataWithoutForeignKeyCascades(t *testing.T) {
	f := newEngagementReportFixture(t)
	if _, err := f.conn.Exec(`PRAGMA foreign_keys=OFF`); err != nil {
		t.Fatal(err)
	}
	batch := map[string]interface{}{"events": []interface{}{engagementReportSignalPayload("purge-observation", "page_view", nil)}}
	engagementReportRequireStatus(t, engagementReportRequest(t, f.router.recordReportSignals, http.MethodPost, f.event, batch, f.ctx, map[string]string{"X-Device-ID": "purge-browser"}), http.StatusNoContent)
	draft := engagementReportDecodeSnapshot(t, engagementReportRequest(t, f.router.prepareEngagementReport, http.MethodGet, f.event, nil, f.ctx, nil))
	engagementReportRequireStatus(t, engagementReportRequest(t, f.router.approveEngagementReport, http.MethodPost, f.event, map[string]interface{}{"calculation_id": draft.CalculationID, "format": "one_page"}, f.ctx, nil), http.StatusOK)
	if err := f.router.db.PurgeEventData(f.event); err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"engagement_report_signals", "engagement_report_calculations", "engagement_report_snapshots"} {
		var count int
		if err := f.conn.QueryRow(`SELECT COUNT(*) FROM `+table+` WHERE event_id=?`, f.event).Scan(&count); err != nil || count != 0 {
			t.Fatalf("orphaned %s rows=%d err=%v", table, count, err)
		}
	}
}

func TestEngagementReportProportionalFactCorrectionsRemainExplicit(t *testing.T) {
	f := newEngagementReportFixture(t)
	batch := map[string]interface{}{"events": []interface{}{engagementReportSignalPayload("proportion-view", "page_view", nil), engagementReportSignalPayload("proportion-vote", "vote_completed", nil)}}
	engagementReportRequireStatus(t, engagementReportRequest(t, f.router.recordReportSignals, http.MethodPost, f.event, batch, f.ctx, map[string]string{"X-Device-ID": "browser"}), http.StatusNoContent)
	draft := engagementReportDecodeSnapshot(t, engagementReportRequest(t, f.router.prepareEngagementReport, http.MethodGet, f.event, nil, f.ctx, nil))
	input := map[string]interface{}{"calculation_id": draft.CalculationID, "format": "one_page", "facts": map[string]*float64{"visitors": engagementReportFloat(2), "engagement_eligible_visitors": engagementReportFloat(2), "active_visitors": engagementReportFloat(2)}, "notes": "Presenze e partecipanti confermati"}
	approved := engagementReportDecodeSnapshot(t, engagementReportRequest(t, f.router.approveEngagementReport, http.MethodPost, f.event, input, f.ctx, nil))
	for _, kpi := range approved.KPIs {
		if kpi.ID == "active_participation" && (kpi.Value == nil || *kpi.Value != 100 || !kpi.Corrected || kpi.Source != "manual" || kpi.Numerator == nil || *kpi.Numerator != 2) {
			t.Fatalf("proportional correction hidden: %+v", kpi)
		}
	}
	input["overrides"] = map[string]reportOverride{"active_participation": {Value: engagementReportFloat(70), Note: "Valore confermato dal referente"}}
	overridden := engagementReportDecodeSnapshot(t, engagementReportRequest(t, f.router.approveEngagementReport, http.MethodPost, f.event, input, f.ctx, nil))
	for _, kpi := range overridden.KPIs {
		if kpi.ID == "active_participation" && (kpi.Numerator != nil || kpi.Denominator != nil || *kpi.Value != 70) {
			t.Fatalf("misleading fraction beside direct correction: %+v", kpi)
		}
	}
}
