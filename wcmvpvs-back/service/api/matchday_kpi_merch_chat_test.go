package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
)

func TestMatchdayMerchAttributesCheckoutAndAnalyticsToOneMatch(t *testing.T) {
	f := newMerchTestFixture(t)
	ctx := context.Background()
	createEvent := func(org database.Organization) int {
		t.Helper()
		id, err := f.rt.db.CreateEvent(database.Event{OrganizationID: org.ID, Team1ID: org.TeamID, Team2ID: f.other.TeamID, StartDateTime: "2026-10-21T19:30:00Z"})
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	matchID, foreignMatch := createEvent(f.org), createEvent(f.other)
	if _, err := f.conn.Exec(`UPDATE events SET is_active=1 WHERE id=?`, matchID); err != nil {
		t.Fatal(err)
	}
	recorder := &matchdayRequestRecorder{db: f.conn, logger: f.rt.baseLogger, secret: "test-signer", pending: make(map[matchdayRequestKey]int)}
	f.rt.matchdayRequests = recorder
	grant := recorder.grant(f.org.ID, matchID, time.Now())
	product := f.product(t, 4)
	path := "/v1/clubs/" + f.org.Slug + "/merch"
	guest := map[string]string{"X-Device-ID": "guest-matchday-device", matchdayGrantHeader: grant}
	otherGuest := map[string]string{"X-Device-ID": "other-matchday-device", matchdayGrantHeader: grant}
	merchRequest(t, f.rt.router, http.MethodGet, fmt.Sprintf("%s?summary=1&match_id=%d", path, matchID), nil, guest, http.StatusOK)
	for _, headers := range []map[string]string{guest, otherGuest} {
		id, err := merchID()
		if err != nil {
			t.Fatal(err)
		}
		opened := merchAnalyticsInput{EventID: id, MatchID: matchID, Type: "merch_shop_opened"}
		for i := 0; i < 2; i++ {
			merchRequest(t, f.rt.router, http.MethodPost, path+"/events", opened, headers, http.StatusNoContent)
		}
		opened.MatchID = foreignMatch
		merchRequest(t, f.rt.router, http.MethodPost, path+"/events", opened, headers, http.StatusNoContent)
	}
	viewID, err := merchID()
	if err != nil {
		t.Fatal(err)
	}
	merchRequest(t, f.rt.router, http.MethodPost, path+"/events", merchAnalyticsInput{
		EventID: viewID, MatchID: matchID, Type: "merch_product_view", ProductID: product.ID,
	}, guest, http.StatusNoContent)
	input := f.order(t, product, "offline")
	input.MatchID = matchID
	var response merchCheckoutResponse
	merchDecode(t, merchRequest(t, f.rt.router, http.MethodPost, path+"/orders", input, guest, http.StatusCreated), &response)
	var retry merchCheckoutResponse
	merchDecode(t, merchRequest(t, f.rt.router, http.MethodPost, path+"/orders", input, map[string]string{
		"X-Device-ID": guest["X-Device-ID"], matchdayGrantHeader: recorder.grant(f.org.ID, matchID, time.Now().Add(-25*time.Hour)),
	}, http.StatusCreated), &retry)
	if retry.Order.ID != response.Order.ID {
		t.Fatalf("expired grant created a different order: %s, expected %s", retry.Order.ID, response.Order.ID)
	}
	if _, err := f.conn.Exec(`UPDATE merch_orders SET payment_status='PAID_OFFLINE',fulfillment_status='CONFIRMED' WHERE id=?`, response.Order.ID); err != nil {
		t.Fatal(err)
	}
	facts := &matchdayFacts{Metrics: map[string]matchdayMetric{}, Resolve: func(id matchdayIdentity) string { return id.DeviceKey }}
	foreign := &matchdayFacts{Metrics: map[string]matchdayMetric{}}
	if err := loadMatchdayMerch(ctx, f.conn, f.org.ID, map[int]*matchdayFacts{matchID: facts, foreignMatch: foreign}); err != nil {
		t.Fatal(err)
	}
	calculateMatchdayShop(facts)
	for metric, want := range map[string]float64{
		"shop_merch_opens": 2, "shop_merch_visitors": 2, "shop_conversion_rate": 50, "shop_paid_orders": 1,
		"shop_reservations": 1, "shop_gmv": 32, "shop_aov": 32, "shop_refunds": 0, "shop_unpaid_reservations": 0,
	} {
		matchdayOperationsAssertValue(t, facts.Metrics[metric], want)
	}
	if len(facts.Actions) != 1 || facts.Actions[0].Identity.DeviceKey != reportOperationalDeviceKey(f.org.ID, guest["X-Device-ID"]) || foreign.Operations.merch.observed {
		t.Fatalf("wrong match attribution or duplicate order: %+v, foreign %+v", facts.Actions, foreign.Operations.merch)
	}
	if products := facts.Metrics["shop_top_products"].Rows; len(products) != 1 || products[0].Label != product.Name || *products[0].Values["views"] != 1 || *products[0].Values["redemptions"] != 1 {
		t.Fatalf("wrong product metrics: %+v", products)
	}
	recorder.flush()
	if err := loadMatchdayTechnology(ctx, f.conn, f.org.ID, map[int]*matchdayFacts{matchID: facts, foreignMatch: foreign}); err != nil {
		t.Fatal(err)
	}
	if facts.Technology.requests != 7 || facts.Technology.errors != 0 || foreign.Technology.requests != 0 {
		t.Fatalf("wrong shop HTTP observation: %+v, foreign %+v", facts.Technology, foreign.Technology)
	}
}

func TestMatchdayMerchSeparatesVisitsPaymentAndNetReceipts(t *testing.T) {
	f := newMerchTestFixture(t)
	ctx := context.Background()
	matchID, err := f.rt.db.CreateEvent(database.Event{
		OrganizationID: f.org.ID, Team1ID: f.org.TeamID, Team2ID: f.other.TeamID, StartDateTime: "2026-10-21T19:30:00Z",
	})
	if err != nil {
		t.Fatal(err)
	}
	p := f.product(t, 20)
	recorder := &matchdayRequestRecorder{db: f.conn, logger: f.rt.baseLogger, secret: "test-signer", pending: make(map[matchdayRequestKey]int)}
	f.rt.matchdayRequests = recorder
	grant := recorder.grant(f.org.ID, matchID, time.Now())
	path := "/v1/clubs/" + f.org.Slug + "/merch/events"
	for _, device := range []string{"device-confirmed", "device-product-only", "device-product-only"} {
		viewID, err := merchID()
		if err != nil {
			t.Fatal(err)
		}
		merchRequest(t, f.rt.router, http.MethodPost, path, merchAnalyticsInput{
			EventID: viewID, MatchID: matchID, Type: "merch_product_view", ProductID: p.ID,
		}, map[string]string{"X-Device-ID": device, matchdayGrantHeader: grant}, http.StatusNoContent)
	}
	for _, fixture := range []struct {
		name, status string
		refund       int64
		late         bool
	}{
		{name: "confirmed", status: "PAID", refund: 500},
		{name: "fully-refunded", status: "PAID", refund: 3200},
		{name: "late-capture", status: "PAID", late: true},
		{name: "declined", status: "FAILED"},
		{name: "expired", status: "EXPIRED"},
		{name: "waiting", status: "PENDING_ONLINE"},
	} {
		input := f.order(t, p, "online")
		input.MatchID, input.matchDeviceKey = matchID, reportOperationalDeviceKey(f.org.ID, "device-"+fixture.name)
		order, err := f.rt.createMerchOrder(ctx, f.org.ID, "acct_test", input, database.FanProfile{}, "test")
		if err != nil {
			t.Fatal(err)
		}
		if fixture.status == "PENDING_ONLINE" {
			continue
		}
		event := merchStripeEvent{
			EventID: "evt_" + fixture.name, OrderID: order.ID, AccountID: "acct_test", SessionID: "cs_" + fixture.name,
			PaymentIntent: "pi_" + fixture.name, Status: fixture.status, Currency: "EUR", TotalCents: order.TotalCents,
		}
		if fixture.late {
			event.OccurredAt = order.ExpiresAt + 1
		}
		if err := f.rt.applyMerchStripeEvent(ctx, event); err != nil {
			t.Fatal(err)
		}
		if fixture.refund > 0 {
			event.EventID, event.Status, event.RefundID, event.RefundedCents = "evt_ref_"+fixture.name, "REFUNDED", "re_"+fixture.name, fixture.refund
			if err := f.rt.applyMerchStripeEvent(ctx, event); err != nil {
				t.Fatal(err)
			}
		}
	}
	facts := &matchdayFacts{Metrics: map[string]matchdayMetric{}, Resolve: func(id matchdayIdentity) string { return id.DeviceKey }}
	if err := loadMatchdayMerch(ctx, f.conn, f.org.ID, map[int]*matchdayFacts{matchID: facts}); err != nil {
		t.Fatal(err)
	}
	calculateMatchdayOperations(facts)
	calculateMatchdayRevenue(facts)
	for code, expected := range map[string]float64{
		"shop_merch_opens": 0, "shop_merch_visitors": 7, "shop_conversion_rate": 100.0 / 7,
		"shop_paid_orders": 1, "shop_gmv": 32, "shop_refunds": 37, "tech_payment_success": 60, "direction_revenue": 59,
	} {
		matchdayOperationsAssertValue(t, facts.Metrics[code], expected)
	}
	var capturedMinusRefunded int64
	if err := f.conn.QueryRow(`SELECT SUM(CASE WHEN p.status='REFUNDED' THEN -p.amount_cents ELSE p.amount_cents END)
		FROM merch_payment_records p JOIN merch_match_orders a ON p.order_id=a.order_id AND p.organization_id=a.organization_id
		WHERE a.event_id=? AND p.status IN ('PAID','PAID_OFFLINE','REFUNDED')`, matchID).Scan(&capturedMinusRefunded); err != nil {
		t.Fatal(err)
	}
	matchdayRequireValue(t, facts.Metrics["direction_revenue"].Rows[0].Values["amount"], float64(capturedMinusRefunded)/100)
}

func TestMatchdayChatMeasuresOpenersRepliesAndBlockedAttemptsAfterDisable(t *testing.T) {
	f := newMatchChatFixture(t)
	f.enable(t, f.a, true)
	f.enable(t, f.b, true)
	open := chatTestPath(f.a, "/messages?matchday_open=1")
	for _, token := range []string{"session1", "session1", "session2"} {
		f.request(t, "GET", open, f.slug, token, "", 200)
	}
	path := chatTestPath(f.a, "/messages")
	var first struct{ Message database.ChatMessage }
	decodeChatTest(t, f.request(t, "POST", path, f.slug, "session1", `{"text":"Forza squadra","request_id":"matchday-reply-parent"}`, 200), &first)
	reply := fmt.Sprintf(`{"text":"Dai!","request_id":"matchday-reply-child","reply_to_id":%d}`, first.Message.ID)
	f.request(t, "POST", chatTestPath(f.b, "/messages"), f.slug, "session2", reply, 409)
	for i := 0; i < 2; i++ {
		f.request(t, "POST", path, f.slug, "session2", reply, 200)
	}
	f.request(t, "POST", path, f.slug, "session2", `{"text":"   ","request_id":"matchday-blocked-attempt"}`, 400)
	f.enable(t, f.a, false)
	orgID, err := f.rt.db.GetEventOrganizationID(f.a)
	if err != nil {
		t.Fatal(err)
	}
	facts := &matchdayFacts{Metrics: map[string]matchdayMetric{}, Resolve: func(id matchdayIdentity) string { return strconv.Itoa(id.FanID) }}
	if err := loadMatchdayChatMeasurement(context.Background(), f.rt.db.SQLConn(), orgID, map[int]*matchdayFacts{f.a: facts}); err != nil {
		t.Fatal(err)
	}
	calculateMatchdayChat(facts)
	for metric, want := range map[string]float64{"chat_openers": 2, "chat_replies": 1, "chat_blocked": 1} {
		matchdayOperationsAssertValue(t, facts.Metrics[metric], want)
		if facts.Metrics[metric].Status != "available" {
			t.Fatalf("%s: expected full coverage from first chat configuration", metric)
		}
	}
	if _, err := f.rt.db.SQLConn().Exec(`UPDATE event_chat_settings SET created_at=0 WHERE event_id=?`, f.a); err != nil {
		t.Fatal(err)
	}
	if err := loadMatchdayChatMeasurement(context.Background(), f.rt.db.SQLConn(), orgID, map[int]*matchdayFacts{f.a: facts}); err != nil {
		t.Fatal(err)
	}
	calculateMatchdayChat(facts)
	for _, code := range []string{"chat_openers", "chat_replies", "chat_blocked"} {
		if facts.Metrics[code].Status != "partial" {
			t.Fatalf("%s: expected partial coverage for a previously configured chat", code)
		}
	}
}
