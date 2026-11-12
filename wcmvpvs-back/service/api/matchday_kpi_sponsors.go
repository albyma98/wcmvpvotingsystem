package api

import (
	"context"
	"database/sql"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
)

const (
	matchdaySponsorImpressions = "sponsor_measured_exposures.impression, method"
	matchdaySponsorViewability = "sponsor_measured_exposures.impression, method, viewable"
	matchdaySponsorReach       = "sponsor_measured_exposures.impression, audience_key"
	matchdaySponsorTime        = "sponsor_measured_exposures.impression, ended, visible_ms"
	matchdaySponsorClicks      = "sponsor_measured_exposures.clickable, clicks"
	matchdaySponsorPlacement   = "sponsor_measured_exposures.placement_id, game_id, method"
	matchdaySponsorLegacyReach = "sponsor_exposures.device_id, exposure_type"
	matchdaySponsorLegacyClick = "sponsor_clicks.device_id, sponsor_id"
	matchdaySponsorCoupon      = "user_coupons.match_id, coupon_id, used_at; coupons.sponsor_id, organization_id"
)

type matchdaySponsorCouponCount struct {
	id, redeemed int
	name         string
}

type matchdaySponsorExtras struct {
	legacyReach, legacyClicks int
	coupons                   []matchdaySponsorCouponCount
	measuredActions           []matchdayAction
	legacyActions             []matchdayAction
}

func loadMatchdaySponsors(ctx context.Context, q matchdayQueryer, orgID int, data map[int]*matchdayFacts, voteSecret string) error {
	in, ids := matchdayIDs(data)
	eventIDs := make([]int, 0, len(ids))
	extras := make(map[int]*matchdaySponsorExtras, len(data))
	for id := range data {
		eventIDs = append(eventIDs, id)
		extras[id] = &matchdaySponsorExtras{}
	}
	reports, err := database.ReadMeasuredSponsorMatches(ctx, q, orgID, eventIDs)
	if err != nil {
		return fmt.Errorf("matchday measured sponsors: %w", err)
	}
	for _, read := range []struct {
		name, query string
		scan        func(*sql.Rows) error
	}{
		{"legacy exposures", `
			SELECT x.event_id, COUNT(DISTINCT NULLIF(x.device_id,''))
			FROM sponsor_exposures x JOIN sponsors s ON s.id=x.sponsor_id
			WHERE s.organization_id=? AND x.event_id IN (` + in + `)
				AND x.exposure_type IN ('seen','watched')
			GROUP BY x.event_id`, func(rows *sql.Rows) error {
			var id, reach int
			if err := rows.Scan(&id, &reach); err != nil {
				return err
			}
			extras[id].legacyReach = reach
			return nil
		}},
		{"legacy clicks", `
			SELECT x.event_id, COALESCE(x.device_id,''), COUNT(*)
			FROM sponsor_clicks x JOIN sponsors s ON s.id=x.sponsor_id
			WHERE s.organization_id=? AND x.event_id IN (` + in + `)
			GROUP BY x.event_id, x.device_id`, func(rows *sql.Rows) error {
			var id, count int
			var device string
			if err := rows.Scan(&id, &device, &count); err != nil {
				return err
			}
			x := extras[id]
			x.legacyClicks += count
			x.legacyActions = append(x.legacyActions, matchdayAction{Identity: matchdayIdentity{Device: device}, Feature: "sponsor_click", Count: count})
			return nil
		}},
		{"coupons", `
			SELECT uc.match_id, c.sponsor_id, COALESCE(NULLIF(MAX(s.report_name),''),MAX(s.name)), COUNT(CASE WHEN uc.used_at IS NOT NULL THEN 1 END)
			FROM user_coupons uc JOIN coupons c ON c.id=uc.coupon_id
			JOIN sponsors s ON s.id=c.sponsor_id AND s.organization_id=c.organization_id
			WHERE c.organization_id=? AND uc.match_id IN (` + in + `)
			GROUP BY uc.match_id, c.sponsor_id ORDER BY uc.match_id, c.sponsor_id`, func(rows *sql.Rows) error {
			var id int
			var count matchdaySponsorCouponCount
			if err := rows.Scan(&id, &count.id, &count.name, &count.redeemed); err != nil {
				return err
			}
			extras[id].coupons = append(extras[id].coupons, count)
			return nil
		}},
	} {
		if err := matchdayReadSponsorRows(ctx, q, read.query, append([]interface{}{orgID}, ids...), read.scan); err != nil {
			return fmt.Errorf("matchday sponsor %s: %w", read.name, err)
		}
	}

	devices := map[int]map[string]string{}
	if voteSecret != "" {
		for id, report := range reports {
			if report.Total.Clicks != nil && *report.Total.Clicks > 0 {
				devices[id] = matchdaySponsorAudienceDevices(orgID, voteSecret, data[id], extras[id].legacyActions)
			}
		}
	}
	if err := matchdayReadSponsorRows(ctx, q, `
		SELECT match_id, audience_key, SUM(clicks) FROM sponsor_measured_exposures
		WHERE club_id=? AND match_id IN (`+in+`) AND clicks>0 GROUP BY match_id, audience_key`,
		append([]interface{}{orgID}, ids...), func(rows *sql.Rows) error {
			var id, count int
			var key string
			if err := rows.Scan(&id, &key, &count); err != nil {
				return err
			}
			x := extras[id]
			x.measuredActions = append(x.measuredActions, matchdayAction{Identity: matchdayIdentity{Device: devices[id][key]}, Feature: "sponsor_click", Count: count})
			return nil
		}); err != nil {
		return fmt.Errorf("matchday measured sponsor clicks: %w", err)
	}
	for id, f := range data {
		extras[id].apply(f, reports[id])
	}
	return nil
}

func matchdayReadSponsorRows(ctx context.Context, q matchdayQueryer, query string, args []interface{}, scan func(*sql.Rows) error) error {
	rows, err := q.QueryContext(ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		if err := scan(rows); err != nil {
			return err
		}
	}
	return rows.Err()
}

func matchdaySponsorAudienceDevices(orgID int, secret string, f *matchdayFacts, legacy []matchdayAction) map[string]string {
	out := map[string]string{}
	seen := map[string]bool{}
	add := func(device string) {
		if seen[device] || !sponsorDeviceID.MatchString(device) {
			return
		}
		seen[device] = true
		value := strconv.Itoa(orgID) + "\x00" + device
		key := base64.RawURLEncoding.EncodeToString(sponsorMAC(secret, "audience", value))
		out[key] = device
	}
	for _, tracking := range f.Tracking {
		add(tracking.Device)
	}
	for _, actions := range [][]matchdayAction{f.Actions, legacy} {
		for _, action := range actions {
			add(action.Identity.Device)
		}
	}
	return out
}

func (x *matchdaySponsorExtras) apply(f *matchdayFacts, report *database.SponsorAnalyticsReport) {
	const noMeasurement = "Nessuna misurazione sponsor associata alla partita."
	const viewableFormula = "Esposizioni HTML con almeno il 50% visibile per almeno 1.000 ms continui."
	const durationFormula = "Somma dei millisecondi sopra soglia del 50% nelle esposizioni concluse."
	const impressionsFormula = "Somma delle impression HTML e geometriche effettivamente registrate."
	const reachFormula = "Dispositivi distinti con almeno un’impression misurata."
	const frequencyFormula = "Impression registrate / dispositivi distinti raggiunti."
	const ctrFormula = "Click misurati / impression registrate × 100."
	impressions := matchdayUnavailable("sponsor_impressions", "Impression sponsor", "Impression misurate, con dettaglio per sponsor.", impressionsFormula, "count", "not_measurable", noMeasurement, matchdaySponsorImpressions)
	viewable := matchdayUnavailable("sponsor_viewable", "Viewable impressions sponsor", "Impression HTML che hanno superato la soglia di visibilità.", viewableFormula, "count", "not_measurable", "Nessuna impression HTML con misurazione di viewability.", matchdaySponsorViewability)
	viewability := matchdayUnavailable("sponsor_viewability", "Tasso di viewability", "Quota delle impression HTML che supera la soglia di visibilità.", "Viewable impressions / impression HTML misurate × 100.", "percent", "not_measurable", viewable.Reason, matchdaySponsorViewability)
	reach := matchdayUnavailable("sponsor_reach", "Reach degli sponsor", "Copertura deduplicata tra sponsor.", reachFormula, "count", "not_measurable", noMeasurement, matchdaySponsorReach)
	frequency := matchdayUnavailable("sponsor_frequency", "Frequency", "Impression registrate per dispositivo raggiunto.", frequencyFormula, "ratio", "not_measurable", noMeasurement, matchdaySponsorImpressions, matchdaySponsorReach)
	totalTime := matchdayUnavailable("sponsor_visible_time", "Tempo visibile totale", "Tempo misurato sopra la soglia di visibilità.", durationFormula, "milliseconds", "not_measurable", "Nessuna esposizione conclusa con campioni di durata.", matchdaySponsorTime)
	avgTime := matchdayUnavailable("sponsor_avg_visible_time", "Tempo visibile medio", "Durata media delle esposizioni concluse con tempo visibile positivo.", "Tempo visibile totale / esposizioni concluse con tempo visibile positivo.", "milliseconds", "not_measurable", totalTime.Reason, matchdaySponsorTime)
	clicks := matchdayUnavailable("sponsor_clicks", "Click agli sponsor", "Click registrati su placement cliccabili.", "Somma dei click misurati.", "count", "not_measurable", "Nessuna copertura di tracciamento per placement cliccabili.", matchdaySponsorClicks)
	ctr := matchdayUnavailable("sponsor_ctr", "CTR sponsor", "Click in relazione a tutte le impression misurate.", ctrFormula, "percent", "not_measurable", "Mancano click e impression da una popolazione comparabile.", matchdaySponsorClicks, matchdaySponsorImpressions)
	conversions := matchdayUnavailable("sponsor_conversions", "Conversioni sponsor", "Riscatti di coupon attribuiti al match e al relativo sponsor.", "Conteggio dei coupon con riscatto registrato.", "count", "not_measurable", "Nessun coupon con attribuzione al match osservata.", matchdaySponsorCoupon)
	placements := matchdayUnavailable("sponsor_placements", "Performance per posizione", "Dettaglio delle posizioni pubblicitarie effettivamente misurate.", "Aggregazione per posizione, gioco e metodo di misura.", "count", "not_measurable", noMeasurement, matchdaySponsorPlacement)

	count := func(metric matchdayMetric, value int) matchdayMetric {
		return matchdayCount(metric.Code, metric.Name, metric.Description, metric.Formula, float64(value), metric.Sources...)
	}
	ratio := func(metric matchdayMetric, numerator, denominator int) matchdayMetric {
		return matchdayRatio(metric.Code, metric.Name, metric.Description, metric.Formula, metric.Unit, float64(numerator), float64(denominator), metric.Sources...)
	}
	if report != nil {
		m := report.Total
		impressions = count(impressions, m.Impressions)
		if m.Viewable != nil {
			viewable = count(viewable, *m.Viewable)
			viewability = ratio(viewability, *m.Viewable, m.HTMLImpressions)
			if m.GeometricImpressions > 0 {
				viewable = matchdayDomainPartial(viewable, "I placement geometrici non hanno una misura di viewability HTML.")
			}
		}
		if m.Reach != nil {
			reach = count(reach, *m.Reach)
			frequency = ratio(frequency, m.Impressions, *m.Reach)
		} else if m.Impressions > 0 {
			reach.Reason = "Alcune impression non hanno un identificativo di dispositivo, necessario per una reach deduplicata."
			frequency.Reason = reach.Reason
		}
		if m.TotalDuration != nil {
			totalTime = matchdayCount(totalTime.Code, totalTime.Name, totalTime.Description, totalTime.Formula, float64(*m.TotalDuration), totalTime.Sources...)
			totalTime.Unit = "milliseconds"
			if m.DurationSamples < m.Impressions {
				totalTime = matchdayDomainPartial(totalTime, "Durata disponibile soltanto per le esposizioni con chiusura registrata.")
			}
		}
		if m.AverageDuration != nil {
			avgTime = matchdayRatio(avgTime.Code, avgTime.Name, avgTime.Description, avgTime.Formula, avgTime.Unit, float64(*m.TotalDuration), float64(m.PositiveDurationSamples), avgTime.Sources...)
			if m.DurationSamples < m.Impressions {
				avgTime = matchdayDomainPartial(avgTime, totalTime.Reason)
			}
		}
		if m.Clicks != nil {
			clicks = count(clicks, *m.Clicks)
			if m.Impressions > 0 {
				ctr = ratio(ctr, *m.Clicks, m.Impressions)
			}
		}
		if len(report.Sponsors) > 0 {
			impressions.Columns = matchdaySponsorColumns()
			for _, sponsor := range report.Sponsors {
				impressions.Rows = append(impressions.Rows, matchdaySponsorRow(sponsor.SponsorName, "", sponsor.SponsorAnalyticsMetrics))
			}
		}
		if len(report.Placements) > 0 {
			placements = count(placements, len(report.Placements))
			placements.Columns = matchdaySponsorColumns()
			for _, placement := range report.Placements {
				label, method := matchdayPlacementName(placement)
				placements.Rows = append(placements.Rows, matchdaySponsorRow(label, method, placement.SponsorAnalyticsMetrics))
			}
		}
	}
	if reach.Value == nil && (report == nil || report.Total.Impressions == 0) && x.legacyReach > 0 {
		reach = matchdayDomainPartial(matchdayCount(reach.Code, reach.Name, reach.Description, "Dispositivi distinti con esposizione legacy vista o guardata.", float64(x.legacyReach), matchdaySponsorLegacyReach), "Copertura disponibile dal tracciamento precedente, senza misura pubblicitaria per placement.")
		reach.coverage = "legacy"
	}
	if clicks.Value == nil && x.legacyClicks > 0 {
		clicks = matchdayDomainPartial(matchdayCount(clicks.Code, clicks.Name, clicks.Description, "Conteggio dei click salvati dal tracciamento precedente.", float64(x.legacyClicks), matchdaySponsorLegacyClick), "Il tracciamento precedente non fornisce impression comparabili per il CTR.")
		clicks.coverage = "legacy"
	}
	if len(x.coupons) > 0 {
		total := 0
		for _, coupon := range x.coupons {
			total += coupon.redeemed
		}
		conversions = matchdayDomainPartial(count(conversions, total), "Attribuzione tramite coupon riscattati. Gli acquisti senza attribuzione sponsor non fanno parte del dato.")
		conversions.Columns = []matchdayColumn{{ID: "redeemed", Label: "Coupon riscattati", Unit: "count"}}
		for _, coupon := range x.coupons {
			conversions.Rows = append(conversions.Rows, matchdayRow{Label: coupon.name, Values: map[string]*float64{"redeemed": matchdayDomainValue(float64(coupon.redeemed))}})
		}
	}
	switch {
	case report != nil && report.Total.Clicks != nil:
		f.Actions = append(f.Actions, x.measuredActions...)
	case len(x.legacyActions) > 0:
		f.Actions = append(f.Actions, x.legacyActions...)
	default:
		for _, signal := range f.Signals {
			if signal.Name == "sponsor_click" {
				f.Actions = append(f.Actions, matchdayAction{Identity: matchdayReportIdentity(signal), Feature: "sponsor_click", Count: 1})
			}
		}
	}
	matchdayPut(f, impressions, viewable, viewability, reach, totalTime, avgTime, clicks, ctr, conversions, frequency, placements)
}

func matchdaySponsorColumns() []matchdayColumn {
	return []matchdayColumn{
		{ID: "impressions", Label: "Impression", Unit: "count"}, {ID: "viewable", Label: "Viewable", Unit: "count"},
		{ID: "viewability", Label: "Viewability", Unit: "percent"}, {ID: "reach", Label: "Reach", Unit: "count"},
		{ID: "total_time", Label: "Tempo visibile", Unit: "milliseconds"}, {ID: "avg_time", Label: "Tempo medio", Unit: "milliseconds"},
		{ID: "clicks", Label: "Click", Unit: "count"}, {ID: "ctr", Label: "CTR", Unit: "percent"},
		{ID: "frequency", Label: "Frequency", Unit: "ratio"},
	}
}

func matchdaySponsorRow(label, group string, m database.SponsorAnalyticsMetrics) matchdayRow {
	values := map[string]*float64{"impressions": matchdayDomainValue(float64(m.Impressions)), "viewability": m.Viewability, "avg_time": m.AverageDuration}
	if m.Viewable != nil {
		values["viewable"] = matchdayDomainValue(float64(*m.Viewable))
	}
	if m.Reach != nil {
		values["reach"] = matchdayDomainValue(float64(*m.Reach))
		values["frequency"] = matchdayDomainRatio(m.Impressions, *m.Reach, 1)
	}
	if m.TotalDuration != nil {
		values["total_time"] = matchdayDomainValue(float64(*m.TotalDuration))
	}
	if m.Clicks != nil {
		values["clicks"] = matchdayDomainValue(float64(*m.Clicks))
		values["ctr"] = matchdayDomainRatio(*m.Clicks, m.Impressions, 100)
	}
	return matchdayRow{Label: label, Group: group, Values: values}
}

func matchdayPlacementName(row database.SponsorAnalyticsRow) (string, string) {
	labels := map[string]string{
		"home_footer": "Home tifosi", "post_vote_left": "Conferma voto MVP, sinistra", "post_vote_right": "Conferma voto MVP, destra",
		"game_wall_left": "Tabellone di gioco, sinistra", "game_wall_right": "Tabellone di gioco, destra",
		"sponsor_rush_entry": "Accesso a Sponsor Rush", "sponsor_rush_logo": "Logo in Sponsor Rush",
	}
	label := labels[row.Placement]
	if label == "" {
		label = row.Placement
	}
	method := map[string]string{database.SponsorHTML: "Display HTML", database.SponsorSVG: "Geometria SVG", database.SponsorInGame: "Geometria di gioco"}[row.Method]
	return label, strings.Trim(strings.Join([]string{method, row.Game}, " · "), " ·")
}
