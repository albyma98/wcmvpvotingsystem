package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
)

func TestCRMLeadToRenewal(t *testing.T) {
	f := newEngagementReportFixture(t)
	rt := f.router
	rt.router = chi.NewRouter()
	rt.sessionTimeout = time.Hour
	actor, err := f.conn.Exec("INSERT INTO admins(username,password_hash,role) VALUES('crm-master','unused','superadmin')")
	if err != nil {
		t.Fatal(err)
	}
	actorID, err := actor.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().In(crmRome)
	rt.adminSessions = map[string]adminSession{
		"master": {AdminID: int(actorID), Username: "crm-master", Role: "superadmin", ExpiresAt: now.Add(time.Hour)},
		"club":   {AdminID: 999, Username: "club-admin", Role: "admin", OrganizationID: f.org.ID, ExpiresAt: now.Add(time.Hour)},
	}
	registerCRMRoutes(rt)
	if err := rt.ensureCRMTables(); err != nil {
		t.Fatalf("repeat migration: %v", err)
	}
	if _, err := rt.db.GetOrganization(f.org.ID); err != nil {
		t.Fatalf("migration changed existing organization: %v", err)
	}
	do := func(token, method, path string, input interface{}, status int, out interface{}) {
		t.Helper()
		body, err := json.Marshal(input)
		if err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest(method, crmPath+path, bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+token)
		response := httptest.NewRecorder()
		rt.router.ServeHTTP(response, req)
		if response.Code != status {
			t.Fatalf("%s %s = %d, want %d. %s", method, path, response.Code, status, response.Body.String())
		}
		if out != nil {
			if err := json.Unmarshal(response.Body.Bytes(), out); err != nil {
				t.Fatalf("decode %s: %v", path, err)
			}
		}
	}
	for _, route := range []struct{ method, path string }{
		{"GET", ""}, {"POST", "/import"}, {"POST", "/companies"}, {"PUT", "/companies/1"},
		{"GET", "/companies/1"}, {"GET", "/companies/1/performance"}, {"POST", "/companies/1/records"},
		{"PUT", "/companies/1/records/1"}, {"POST", "/companies/1/contracts/1/renewal"},
	} {
		do("", route.method, route.path, nil, 401, nil)
		do("club", route.method, route.path, nil, 403, nil)
	}

	type importResult struct {
		CompanyIDs     []int64 `json:"companyIds"`
		CompaniesAdded int     `json:"companiesAdded"`
		ContactsAdded  int     `json:"contactsAdded"`
	}
	csv := "codice_societa;societa;sport;codice_referente;referente;email;influenza\n" +
		"lead-01;Prospect Roma;Basket;decision;Ada;ada@example.test;decision\n" +
		"lead-01;Prospect Roma;Basket;ops;Luca;luca@example.test;operations"
	var imported, repeated importResult
	do("master", "POST", "/import", map[string]string{"csv": csv}, 200, &imported)
	do("master", "POST", "/import", map[string]string{"csv": strings.ReplaceAll(csv, "Prospect Roma", "Duplicato")}, 200, &repeated)
	if imported.CompaniesAdded != 1 || imported.ContactsAdded != 2 || len(imported.CompanyIDs) != 1 ||
		repeated.CompaniesAdded != 0 || repeated.ContactsAdded != 0 || len(repeated.CompanyIDs) != 1 || repeated.CompanyIDs[0] != imported.CompanyIDs[0] {
		t.Fatalf("invalid import/retry: %+v %+v", imported, repeated)
	}
	do("master", "POST", "/import", map[string]string{"csv": "codice_societa,societa,referente,email\nbad,Rollback,Ada,invalid"}, 400, nil)
	var rollback int
	if err := f.conn.QueryRow("SELECT COUNT(*) FROM crm_companies WHERE import_key='bad'").Scan(&rollback); err != nil || rollback != 0 {
		t.Fatalf("failed import wasn't rolled back: %d %v", rollback, err)
	}

	var profile struct {
		Company crmCompany
		Records []crmRecord
	}
	base := fmt.Sprintf("/companies/%d", imported.CompanyIDs[0])
	read := func() { do("master", "GET", base, nil, 200, &profile) }
	read()
	if profile.Company.Status != "lead" || profile.Company.Name != "Prospect Roma" || profile.Company.ABXOrgID != 0 {
		t.Fatalf("invalid lead: %+v", profile.Company)
	}
	contactIDs := map[string]int64{}
	for _, item := range profile.Records {
		if item.Kind == "contact" {
			contactIDs[item.Type] = item.ID
		}
	}
	if len(contactIDs) != 2 {
		t.Fatalf("missing contacts: %+v", contactIDs)
	}
	save := func(value crmRecord, status int) crmRecord {
		t.Helper()
		var out crmRecord
		if value.ID == 0 {
			do("master", "POST", base+"/records", value, status, &out)
		} else {
			do("master", "PUT", fmt.Sprintf("%s/records/%d", base, value.ID), value, status, &out)
		}
		return out
	}
	today := now.Format(crmDate)
	end := now.AddDate(0, 1, 0).Format(crmDate)
	company := profile.Company
	company.OwnerID = actorID
	do("master", "PUT", base, company, 200, &company)
	opportunity := save(crmRecord{Kind: "opportunity", ContactID: contactIDs["decision"], crmRecordDetails: crmRecordDetails{
		Title: "Prima trattativa", Type: "new", Status: "to_contact", Season: "2026/27", NextStep: "Demo",
	}}, 200)

	call := crmRecordRequest{
		crmRecord: crmRecord{Kind: "activity", ContactID: contactIDs["decision"], OpportunityID: opportunity.ID, crmRecordDetails: crmRecordDetails{
			Title: "Prima telefonata", Type: "call", Date: today + "T10:00", Result: "Demo concordata",
		}},
		FollowUp: &crmRecord{OwnerID: actorID, crmRecordDetails: crmRecordDetails{Title: "Fissare la demo", Priority: "high"}},
	}
	do("master", "POST", base+"/records", call, 400, nil)
	read()
	for _, item := range profile.Records {
		if item.Kind == "activity" || item.Kind == "task" {
			t.Fatal("a failed follow-up left a partial record")
		}
	}
	call.FollowUp.Due = today
	var logged crmRecord
	do("master", "POST", base+"/records", call, 200, &logged)
	read()
	var task crmRecord
	for _, item := range profile.Records {
		if item.Kind == "task" && item.Title == call.FollowUp.Title {
			task = item
		}
	}
	if logged.AuthorID != actorID || logged.AuthorName != "crm-master" || task.OwnerID != actorID || task.OpportunityID != opportunity.ID || task.ContactID != contactIDs["decision"] {
		t.Fatalf("audit or follow-up lost: %+v %+v", logged, task)
	}
	task.Status = "done"
	completed := save(task, 200)
	if completed.CompletedAt == "" {
		t.Fatal("completion was not dated")
	}
	do("master", "PUT", fmt.Sprintf("%s/records/%d", base, task.ID), task, 409, nil)

	opportunity.Status = "pilot"
	opportunity = save(opportunity, 200)
	pilot := save(crmRecord{Kind: "pilot", OpportunityID: opportunity.ID, ContactID: contactIDs["operations"], crmRecordDetails: crmRecordDetails{
		Title: "Pilota Roma", Status: "active", Start: today, End: end, Goal: "Attivare la prima gara",
	}}, 200)
	company.ABXOrgID = int64(f.org.ID)
	company.Status = "onboarding"
	do("master", "PUT", base, company, 200, &company)
	pilot.EventIDs = []int64{int64(f.event)}
	pilot = save(pilot, 200)
	checklist := save(crmRecord{Kind: "checklist", crmRecordDetails: crmRecordDetails{Type: "setup", Status: "done"}}, 200)
	if checklist.CompletedAt == "" {
		t.Fatal("checklist completion not dated")
	}
	var other crmCompany
	do("master", "POST", "/companies", crmCompany{Name: "Altra società", ABXOrgID: company.ABXOrgID}, 409, nil)
	do("master", "POST", "/companies", crmCompany{Name: "Altra società", ABXOrgID: int64(f.other.ID)}, 200, &other)
	do("master", "POST", fmt.Sprintf("/companies/%d/records", other.ID),
		crmRecord{Kind: "opportunity", ContactID: contactIDs["decision"], crmRecordDetails: crmRecordDetails{Title: "Riferimento errato", Type: "new", Status: "pilot"}}, 400, nil)
	do("master", "POST", fmt.Sprintf("/companies/%d/records", other.ID),
		crmRecord{Kind: "pilot", crmRecordDetails: crmRecordDetails{Title: "Gara altrui", Status: "planned", Start: today, End: end, EventIDs: pilot.EventIDs}}, 400, nil)

	// A closed match may have no report signals. Missing data and a measured
	// zero must remain distinct all the way through the CRM API response.
	matchTime := now.AddDate(0, 0, -1).UTC().Format(time.RFC3339)
	if _, err := f.conn.Exec("UPDATE events SET is_concluded=1,start_datetime=? WHERE id=?", matchTime, f.event); err != nil {
		t.Fatal(err)
	}
	var trend crmTrend
	do("master", "GET", base+"/performance", nil, 200, &trend)
	if trend.Activated != 1 || trend.LastActivity == nil || trend.LastActivity.ID != f.event || len(trend.Matches) != 1 ||
		trend.Matches[0].KPIs[0].Value != nil || trend.Matches[0].ReportPath == "" {
		t.Fatalf("missing data was fabricated or match was lost: %+v", trend)
	}
	if _, err := f.conn.Exec("INSERT INTO engagement_report_signals(event_id,signal_id,name,session_id,device_key,occurred_at,metadata) VALUES(?,?,?,?,?,?,?)",
		f.event, "crm-signal", "page_view", "s", "device", matchTime, `{"surface":"new"}`); err != nil {
		t.Fatal(err)
	}
	do("master", "GET", base+"/performance", nil, 200, &trend)
	for _, kpi := range trend.Matches[0].KPIs {
		if kpi.ID == "active_participation" && (kpi.Value == nil || *kpi.Value != 0 || kpi.Denominator == nil || *kpi.Denominator != 1) {
			t.Fatalf("zero and denominator were not preserved: %+v", kpi)
		}
		if kpi.ID == "sponsor_reach" && kpi.Value != nil {
			t.Fatalf("sponsor data absent but KPI set: %+v", kpi)
		}
	}

	pilot.Status, pilot.Result, pilot.NextStep = "completed", "Accettato", "Firma"
	pilot = save(pilot, 200)
	amount := int64(150000)
	opportunity.Status, opportunity.Amount = "won", &amount
	opportunity = save(opportunity, 200)
	company.Status = "active"
	do("master", "PUT", base, company, 200, &company)
	contract := save(crmRecord{Kind: "contract", OpportunityID: opportunity.ID, ContactID: contactIDs["decision"], crmRecordDetails: crmRecordDetails{
		Title: "Contratto Roma", Season: "2026/27", Status: "signed", Start: today, End: end, Due: today, Amount: &amount,
	}}, 200)
	deposit := int64(40000)
	save(crmRecord{Kind: "payment", ContractID: contract.ID, crmRecordDetails: crmRecordDetails{Title: "Acconto", Date: today, Amount: &deposit}}, 200)
	do("master", "POST", base+"/records", crmRecord{Kind: "payment", ContractID: contract.ID, crmRecordDetails: crmRecordDetails{
		Title: "Incasso oltre residuo", Date: today, Amount: &amount,
	}}, 400, nil)

	renewalURL := fmt.Sprintf("%s/contracts/%d/renewal", base, contract.ID)
	var firstRenewal, secondRenewal crmRecord
	do("master", "POST", renewalURL, nil, 200, &firstRenewal)
	do("master", "POST", renewalURL, nil, 200, &secondRenewal)
	if firstRenewal.Type != "renewal" || firstRenewal.ContractID != contract.ID || firstRenewal.ContactID != contactIDs["decision"] ||
		firstRenewal.Due != contract.End || firstRenewal.CompanyID != company.ID || firstRenewal.ID != secondRenewal.ID {
		t.Fatalf("invalid renewal: %+v %+v", firstRenewal, secondRenewal)
	}
	read()
	var gotCall, gotPilot, gotContract, gotLinkHistory, gotPhaseHistory bool
	renewalTasks := 0
	for _, item := range profile.Records {
		gotCall = gotCall || item.ID == logged.ID
		gotPilot = gotPilot || (item.ID == pilot.ID && item.Result == "Accettato" && len(item.EventIDs) == 1)
		gotContract = gotContract || item.ID == contract.ID
		gotLinkHistory = gotLinkHistory || (item.Kind == "history" && strings.Contains(item.Notes, "Collegamento ABX"))
		gotPhaseHistory = gotPhaseHistory || (item.Kind == "history" && item.OpportunityID == opportunity.ID && strings.Contains(item.Notes, "Pilota → Vinta"))
		if item.Kind == "task" && item.OpportunityID == firstRenewal.ID && item.Due == contract.Due {
			renewalTasks++
		}
	}
	if profile.Company.ID != company.ID || profile.Company.Status != "active" || profile.Company.ABXOrgID != int64(f.org.ID) ||
		!gotCall || !gotPilot || !gotContract || !gotLinkHistory || !gotPhaseHistory || renewalTasks != 1 {
		t.Fatalf("incomplete CRM lifecycle. company=%+v call=%t pilot=%t contract=%t link=%t phase=%t renewalTasks=%d",
			profile.Company, gotCall, gotPilot, gotContract, gotLinkHistory, gotPhaseHistory, renewalTasks)
	}
}
