package api

import (
	"context"
	"math"
	"strconv"
	"testing"
)

func TestMatchdayOperationsSeparatesChatAndBarActivity(t *testing.T) {
	f := newMatchChatFixture(t)
	db := f.rt.db.SQLConn()
	org, err := f.rt.db.GetEventOrganizationID(f.a)
	if err != nil {
		t.Fatal(err)
	}
	otherOrg, err := f.rt.db.GetEventOrganizationID(f.other)
	if err != nil {
		t.Fatal(err)
	}
	var fan1, fan2 int
	if err := db.QueryRow(`SELECT MIN(id),MAX(id) FROM fan_profiles WHERE organization_id=?`, org).Scan(&fan1, &fan2); err != nil {
		t.Fatal(err)
	}
	exec := func(query string, args ...interface{}) {
		t.Helper()
		if _, err := db.Exec(query, args...); err != nil {
			t.Fatal(err)
		}
	}
	exec(`INSERT INTO event_chat_messages(id,event_id,author_user_id,request_id,message_text,message_type,created_at,moderation_status)
		VALUES(1,?,?,'fan-1','Uno','fan',1,'removed'),(2,?,?,'fan-2','Due','fan',2,'visible'),
		(3,?,NULL,'official','Tre','official',3,'visible'),(4,?,NULL,'bot','Quattro','ai',4,'visible'),
		(5,?,NULL,'foreign','Esterno','ai',5,'visible')`, f.a, fan1, f.a, fan1, f.a, f.a, f.other)
	exec(`INSERT INTO event_chat_reports(event_id,message_id,reporter_user_id,reason,created_at,review_status)
		VALUES(?,1,?,'spam',1,'reviewed'),(?,1,?,'spam',1,'pending')`, f.a, fan1, f.a, fan2)
	exec(`INSERT INTO event_chat_reactions(event_id,message_id,fan_id,emoji) VALUES(?,2,?,'🔥'),(?,2,?,'🔥')`, f.a, fan1, f.a, fan2)

	for _, fixture := range []struct {
		ref, pay, status string
		org, cents       int
	}{
		{"valid", "paid", "completed", org, 725}, {"expired", "expired", "cancelled", org, 750},
		{"cancelled", "paid", "annullato", org, 800}, {"conflict", "paid", "completed", org, 825},
		{"foreign", "paid", "completed", otherOrg, 875}, {"no_signal", "paid", "completed", org, 900},
	} {
		exec(`INSERT INTO bar_orders(organization_id,products_json,quantities_json,total_cents,sector,row_label,seat,order_status,payment_status,stripe_reference)
			VALUES(?,'[]','[]',?,'','','',?,?,?)`, fixture.org, fixture.cents, fixture.status, fixture.pay, fixture.ref)
	}
	for _, observation := range []struct {
		event     int
		name, ref string
	}{
		{f.a, "bar.checkout_redirected", "valid"}, {f.a, "bar.order_completed", "valid"},
		{f.a, "bar.checkout_redirected", "expired"}, {f.a, "bar.checkout_redirected", "cancelled"},
		{f.a, "bar.checkout_redirected", "conflict"}, {f.b, "bar.checkout_redirected", "conflict"},
		{f.a, "bar.checkout_redirected", "foreign"},
	} {
		exec(`INSERT INTO tracking_events(event_id,organization_id,event_name,metadata_json) VALUES(?,?,?,?)`, observation.event, org, observation.name, `{"session_id":"`+observation.ref+`"}`)
	}
	exec(`INSERT INTO tracking_events(event_id,organization_id,event_name,metadata_json) VALUES(?,?,?,?)`, f.a, org, "bar.order_completed", "{malformed")
	identity := "fan:" + strconv.Itoa(fan1)
	facts := &matchdayFacts{
		Metrics: map[string]matchdayMetric{}, Reach: map[string]bool{identity: true, "device:guest": true},
		Resolve: func(id matchdayIdentity) string {
			if id.FanID == fan1 || id.DeviceKey == "linked" {
				return identity
			}
			return "device:" + id.DeviceKey
		},
		Signals:  []reportSignal{{Name: "match_chat_opened", DeviceID: "linked"}, {Name: "match_chat_opened", FanID: fan1}},
		Tracking: []matchdayTracking{{Name: "bar.checkout_redirected", Device: "device1", Metadata: map[string]interface{}{"session_id": "valid"}}},
	}
	disabled := &matchdayFacts{Metrics: map[string]matchdayMetric{}}
	if err := loadMatchdayOperations(context.Background(), db, org, map[int]*matchdayFacts{f.a: facts, f.b: disabled}); err != nil {
		t.Fatal(err)
	}
	calculateMatchdayOperations(facts)
	calculateMatchdayOperations(disabled)
	for code, want := range map[string]float64{
		"chat_openers": 1, "chat_participants": 1, "chat_adoption": 50, "chat_total_messages": 3,
		"chat_average_messages": 2, "chat_bot_messages": 1, "chat_fan_reactions": 1, "chat_reported": 1,
		"chat_removed": 1, "chat_moderation_rate": 50, "shop_bar_paid_orders": 1, "tech_payment_success": 50,
	} {
		matchdayOperationsAssertValue(t, facts.Metrics[code], want)
	}
	if got := disabled.Metrics["chat_total_messages"]; got.Value != nil || got.Status != "not_applicable" || got.Reason != "Funzione non attiva" {
		t.Fatalf("disabled chat: %+v", got)
	}
	if facts.BarRevenue == nil || *facts.BarRevenue != 7.25 || len(facts.Actions) != 2 {
		t.Fatalf("revenue %v, actions %+v", facts.BarRevenue, facts.Actions)
	}
	if facts.Actions[1].Feature != "purchase" || facts.Actions[1].Identity.Device != "device1" {
		t.Fatalf("purchase identity: %+v", facts.Actions[1])
	}
}

func TestMatchdayOperationsKeepsMissingAndManualZerosDistinct(t *testing.T) {
	facts := &matchdayFacts{Metrics: map[string]matchdayMetric{}, Manual: map[string]matchdayManualField{},
		Signals: []reportSignal{{Name: "vote_completed"}, {Name: "vote_completed"}, {Name: "vote_failed", Metadata: map[string]interface{}{"reason": "rate_limited"}}}}
	calculateMatchdayOperations(facts)
	matchdayOperationsAssertValue(t, facts.Metrics["tech_vote_success"], 200.0/3)
	if paid := facts.Metrics["shop_paid_orders"]; paid.Value != nil || paid.Status != "missing" {
		t.Fatalf("unmeasured paid orders: %+v", paid)
	}
	for code, amount := range map[string]int{"shop_paid_orders": 0, "shop_gmv": 0, "shop_reservations": 2} {
		facts.Manual[code] = matchdayManualField{matchdayManualRevision: matchdayManualRevision{Value: matchdayOperationsValue(amount)}}
	}
	calculateMatchdayOperations(facts)
	matchdayOperationsAssertValue(t, facts.Metrics["shop_paid_orders"], 0)
	if average := facts.Metrics["shop_aov"]; average.Value != nil || average.Status != "not_applicable" {
		t.Fatalf("zero-denominator average: %+v", average)
	}
	if conversions := facts.Metrics["shop_conversions"]; conversions.Value != nil || conversions.Source != "manual" || conversions.Status != "available" || len(conversions.Rows) != 2 {
		t.Fatalf("conversion details: %+v", conversions)
	}
	if p95 := facts.Metrics["tech_api_p95"]; p95.Value != nil || p95.Status != "not_measurable" {
		t.Fatalf("unmeasured API p95: %+v", p95)
	}
}

func matchdayOperationsAssertValue(t *testing.T, metric matchdayMetric, want float64) {
	t.Helper()
	if metric.Value == nil || math.Abs(*metric.Value-want) > 0.01 {
		t.Errorf("%s: want %v, got %+v", metric.Code, want, metric)
	}
}
