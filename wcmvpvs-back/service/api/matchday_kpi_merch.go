package api

import (
	"context"
	"database/sql"
	"sort"
)

type matchdayMerchFacts struct {
	observed, enabled                bool
	opens, unidentifiedOpens         int
	paid, reservations, unpaid       int
	gmvCents, refundCents, netCents  int64
	paymentPaid, paymentResolved     int
	observedPayment                  bool
	visitIdentities, buyerIdentities []matchdayIdentity
	products                         map[string]matchdayMerchProduct
}

type matchdayMerchProduct struct {
	name         string
	views, units int
}

func loadMatchdayMerch(ctx context.Context, q matchdayQueryer, orgID int, data map[int]*matchdayFacts) error {
	ids, args := matchdayIDs(data)
	args = append(args, orgID)
	if err := matchdayRead(ctx, q, `SELECT event_id,enabled FROM merch_match_coverage WHERE event_id IN (`+ids+`) AND organization_id=?`, args, func(rows *sql.Rows) error {
		var eventID int
		var enabled bool
		if err := rows.Scan(&eventID, &enabled); err != nil {
			return err
		}
		m := &data[eventID].Operations.merch
		m.observed, m.enabled = true, enabled
		return nil
	}); err != nil {
		return err
	}
	if err := matchdayRead(ctx, q, `SELECT a.event_id,COALESCE(f.id,0),a.device_key,COUNT(*)
		FROM merch_match_activity a JOIN merch_analytics_events e ON e.organization_id=a.organization_id AND e.event_id=a.analytics_id
		LEFT JOIN fan_profiles f ON f.id=a.fan_id AND f.organization_id=a.organization_id
		WHERE a.event_id IN (`+ids+`) AND a.organization_id=? AND e.type='merch_shop_opened'
		GROUP BY a.event_id,f.id,a.device_key`, args, func(rows *sql.Rows) error {
		var eventID, opens int
		var identity matchdayIdentity
		if err := rows.Scan(&eventID, &identity.FanID, &identity.DeviceKey, &opens); err != nil {
			return err
		}
		m := &data[eventID].Operations.merch
		m.observed, m.enabled = true, true
		m.opens += opens
		if identity.FanID > 0 || identity.DeviceKey != "" {
			m.visitIdentities = append(m.visitIdentities, identity)
		} else {
			m.unidentifiedOpens += opens
		}
		return nil
	}); err != nil {
		return err
	}
	const eligible = `('CONFIRMED','PREPARING','READY_FOR_PICKUP','COMPLETED')`
	const paid = `o.payment_status IN ('PAID','PAID_OFFLINE') AND o.fulfillment_status IN ` + eligible
	if err := matchdayRead(ctx, q, `SELECT a.event_id,COALESCE(f.id,0),a.device_key,o.mode,o.payment_status,o.total_cents,
		`+paid+`,o.mode='offline' AND o.fulfillment_status IN `+eligible+`,
		COALESCE((SELECT SUM(p.amount_cents) FROM merch_payment_records p
			WHERE p.organization_id=o.organization_id AND p.order_id=o.id AND p.status='REFUNDED'),0)
		FROM merch_match_orders a JOIN merch_orders o ON o.organization_id=a.organization_id AND o.id=a.order_id
		LEFT JOIN fan_profiles f ON f.id=o.fan_id AND f.organization_id=a.organization_id
		WHERE a.event_id IN (`+ids+`) AND a.organization_id=?`, args, func(rows *sql.Rows) error {
		var eventID int
		var identity matchdayIdentity
		var mode, status string
		var total, refunds int64
		var paid, reserved bool
		if err := rows.Scan(&eventID, &identity.FanID, &identity.DeviceKey, &mode, &status, &total, &paid, &reserved, &refunds); err != nil {
			return err
		}
		f := data[eventID]
		m := &f.Operations.merch
		m.observed, m.enabled = true, true
		m.refundCents += refunds
		if paid {
			m.paid++
			m.gmvCents += total
			m.netCents += total - refunds
			m.buyerIdentities = append(m.buyerIdentities, identity)
		}
		if reserved {
			m.reservations++
			if status == "UNPAID_OFFLINE" {
				m.unpaid++
			}
		}
		if reserved || paid {
			feature := "purchase"
			if reserved {
				feature = "reservation"
			}
			f.Actions = append(f.Actions, matchdayAction{Identity: identity, Feature: feature, Count: 1})
		}
		if mode == "online" {
			m.observedPayment = true
			switch status {
			case "PAID", "REFUNDED":
				m.paymentPaid++
				m.paymentResolved++
			case "FAILED":
				m.paymentResolved++
			}
		}
		return nil
	}); err != nil {
		return err
	}
	productArgs := append(append([]interface{}{}, args...), args...)
	if err := matchdayRead(ctx, q, `SELECT observed.event_id,observed.product_id,MAX(p.name),SUM(observed.views),SUM(observed.units)
		FROM (
			SELECT a.event_id,a.organization_id,e.product_id,COUNT(*) AS views,0 AS units
			FROM merch_match_activity a JOIN merch_analytics_events e ON e.organization_id=a.organization_id AND e.event_id=a.analytics_id
			WHERE a.event_id IN (`+ids+`) AND a.organization_id=? AND e.type='merch_product_view'
			GROUP BY a.event_id,a.organization_id,e.product_id
			UNION ALL
			SELECT a.event_id,a.organization_id,i.product_id,0,SUM(i.quantity)
			FROM merch_match_orders a JOIN merch_orders o ON o.organization_id=a.organization_id AND o.id=a.order_id
			JOIN merch_order_items i ON i.organization_id=o.organization_id AND i.order_id=o.id
			WHERE a.event_id IN (`+ids+`) AND a.organization_id=? AND `+paid+`
			GROUP BY a.event_id,a.organization_id,i.product_id
		) observed JOIN merch_products p ON p.id=observed.product_id AND p.organization_id=observed.organization_id
		GROUP BY observed.event_id,observed.product_id`, productArgs, func(rows *sql.Rows) error {
		var eventID int
		var productID string
		var p matchdayMerchProduct
		if err := rows.Scan(&eventID, &productID, &p.name, &p.views, &p.units); err != nil {
			return err
		}
		m := &data[eventID].Operations.merch
		m.observed, m.enabled = true, true
		if m.products == nil {
			m.products = map[string]matchdayMerchProduct{}
		}
		m.products[productID] = p
		return nil
	}); err != nil {
		return err
	}
	for _, f := range data {
		m := f.Operations.merch
		if m.observed && m.enabled {
			net := float64(m.netCents) / 100
			f.MerchRevenue = &net
		}
	}
	return nil
}

func applyMatchdayMerch(f *matchdayFacts) {
	m := f.Operations.merch
	if !m.observed {
		return
	}
	const activitySource = "merch_match_activity, merch_analytics_events.type"
	const orderSource = "merch_match_orders, merch_orders.payment_status, fulfillment_status, total_cents"
	put := func(code string, value float64, sources ...string) matchdayMetric {
		metric := f.Metrics[code]
		metric.Sources, metric.Missing = sources, nil
		metric.Status, metric.Source, metric.Reason = "available", "db", ""
		metric.Value = &value
		metric.coverage = "merch-match"
		if !m.enabled {
			metric.Status, metric.Reason, metric.Value = "not_applicable", "Shop merchandising non attivo sulla home dell'evento.", nil
		}
		matchdayPut(f, metric)
		return metric
	}
	put("shop_merch_opens", float64(m.opens), activitySource)
	visitors, buyers := map[string]bool{}, map[string]bool{}
	if f.Resolve != nil {
		for _, visit := range m.visitIdentities {
			if identity := f.Resolve(visit); identity != "" {
				visitors[identity] = true
			}
		}
		for _, purchase := range m.buyerIdentities {
			if identity := f.Resolve(purchase); identity != "" {
				// Checkout is direct evidence of a shop visit if the opening signal was lost.
				buyers[identity], visitors[identity] = true, true
			}
		}
	}
	unique := put("shop_merch_visitors", float64(len(visitors)), activitySource, orderSource)
	unique.Formula = "Identità distinte osservate all'apertura o all'acquisto nello shop"
	if m.enabled && m.unidentifiedOpens > 0 {
		unique.Status, unique.Reason = "partial", "Alcune aperture sono prive di un identificativo pseudonimo."
	}
	matchdayPut(f, unique)
	conversion := put("shop_conversion_rate", 0, activitySource, orderSource)
	if m.enabled {
		conversion = matchdayRatio(conversion.Code, conversion.Name, conversion.Description, conversion.Formula, "percent", float64(len(buyers)), float64(len(visitors)), conversion.Sources...)
		if conversion.Value != nil && (m.unidentifiedOpens > 0 || hasUnidentifiedMerchBuyer(m.buyerIdentities)) {
			conversion.Status, conversion.Reason = "partial", "Alcune aperture o alcuni acquisti non hanno un'identità associata."
		}
		conversion.coverage = "merch-match"
	}
	matchdayPut(f, conversion)
	paid := put("shop_paid_orders", float64(m.paid), orderSource)
	gmv := put("shop_gmv", float64(m.gmvCents)/100, orderSource)
	reservations := put("shop_reservations", float64(m.reservations), orderSource)
	put("shop_unpaid_reservations", float64(m.unpaid), orderSource)
	put("shop_refunds", float64(m.refundCents)/100, orderSource, "merch_payment_records.status, amount_cents")
	aov := put("shop_aov", 0, orderSource)
	if m.enabled {
		aov = matchdayRatio(aov.Code, aov.Name, aov.Description, aov.Formula, "eur", *gmv.Value, *paid.Value, orderSource)
		aov.coverage = "merch-match"
	}
	matchdayPut(f, aov)
	executive := put("shop_conversions", 0, orderSource)
	executive.Value = nil
	executive.Rows = []matchdayRow{{Label: paid.Name, Values: map[string]*float64{"count": paid.Value}}, {Label: reservations.Name, Values: map[string]*float64{"count": reservations.Value}}}
	matchdayPut(f, executive)
	if !m.enabled {
		return
	}
	products := f.Metrics["shop_top_products"]
	products.Status, products.Source, products.Reason = "available", "db", ""
	products.Description = "Classifica dei prodotti del merchandising e degli oggetti virtuali osservati nella partita."
	products.Formula = "Visualizzazioni e quantità acquistate o riscattate per prodotto"
	products.Sources = append(products.Sources, activitySource, orderSource, "merch_order_items.quantity")
	products.Columns = []matchdayColumn{{ID: "views", Label: "Visualizzazioni", Unit: "count"}, {ID: "redemptions", Label: "Acquisti o riscatti", Unit: "count"}}
	ordered := make([]matchdayMerchProduct, 0, len(m.products))
	for _, product := range m.products {
		ordered = append(ordered, product)
	}
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].views != ordered[j].views {
			return ordered[i].views > ordered[j].views
		}
		if ordered[i].units != ordered[j].units {
			return ordered[i].units > ordered[j].units
		}
		return ordered[i].name < ordered[j].name
	})
	for _, product := range ordered {
		products.Rows = append(products.Rows, matchdayRow{Label: product.name, Group: "Merchandising", Values: map[string]*float64{
			"views": matchdayOperationsValue(product.views), "redemptions": matchdayOperationsValue(product.units),
		}})
	}
	if len(products.Rows) == 0 {
		products.Value = matchdayOperationsValue(0)
	}
	matchdayPut(f, products)
}

func hasUnidentifiedMerchBuyer(buyers []matchdayIdentity) bool {
	for _, buyer := range buyers {
		if buyer.FanID == 0 && buyer.DeviceKey == "" {
			return true
		}
	}
	return false
}
