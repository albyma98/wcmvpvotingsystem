package api

import "fmt"

// Engagement CTR counts devices that clicked and saw the same sponsor.
// Its denominator is unique reach. Commercial CTR divides clicks by viewable impressions.
type reportMeasuredSponsors struct {
	id                                                   int
	name                                                 string
	rows, htmlImpressions, viewable, reach, unknownReach int
	clickable, clicks, clickers, anonymousClicks         int
	exposedClickers                                      int
}

func (rt *_router) loadReportMeasuredSponsors(eventID, orgID, sponsorID int) (reportMeasuredSponsors, []reportMeasuredSponsors, error) {
	var total reportMeasuredSponsors
	var sponsors []reportMeasuredSponsors
	query := `WITH scoped AS (
		SELECT sponsor_id, sponsor_name, audience_key, method, impression, viewable, clickable, clicks
		FROM sponsor_measured_exposures WHERE match_id=? AND club_id=?`
	args := []interface{}{eventID, orgID}
	if sponsorID > 0 {
		query += ` AND sponsor_id=?`
		args = append(args, sponsorID)
	}
	query += `), by_scope AS (
			SELECT 0 AS scope_id, * FROM scoped
			UNION ALL SELECT sponsor_id, * FROM scoped
		), audience AS (
			SELECT scope_id, sponsor_id, audience_key, MAX(viewable) AS seen, MAX(clicks>0) AS clicked
			FROM by_scope WHERE audience_key<>'' GROUP BY scope_id, sponsor_id, audience_key
		), unique_audience AS (
			SELECT scope_id,
				COUNT(DISTINCT CASE WHEN seen THEN audience_key END) AS reach,
				COUNT(DISTINCT CASE WHEN clicked THEN audience_key END) AS clickers,
				COUNT(DISTINCT CASE WHEN seen AND clicked THEN audience_key END) AS exposed_clickers
			FROM audience GROUP BY scope_id
		)
		SELECT scope_id, MAX(sponsor_name), COUNT(*),
			COUNT(CASE WHEN impression=1 AND method='html_display' THEN 1 END),
			COUNT(CASE WHEN viewable=1 THEN 1 END),
			COALESCE(MAX(reach),0),
			COUNT(CASE WHEN viewable=1 AND audience_key='' THEN 1 END),
			COUNT(CASE WHEN clickable=1 THEN 1 END), COALESCE(SUM(clicks),0),
			COALESCE(MAX(clickers),0),
			COALESCE(SUM(CASE WHEN audience_key='' THEN clicks ELSE 0 END),0),
			COALESCE(MAX(exposed_clickers),0)
		FROM by_scope LEFT JOIN unique_audience USING (scope_id) GROUP BY scope_id ORDER BY scope_id`
	rows, err := rt.db.SQLConn().Query(query, args...)
	if err != nil {
		return total, nil, fmt.Errorf("report measured sponsor exposures: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var m reportMeasuredSponsors
		if err := rows.Scan(&m.id, &m.name, &m.rows, &m.htmlImpressions, &m.viewable, &m.reach,
			&m.unknownReach, &m.clickable, &m.clicks, &m.clickers, &m.anonymousClicks, &m.exposedClickers); err != nil {
			return total, nil, err
		}
		if m.id == 0 {
			total = m
		} else {
			sponsors = append(sponsors, m)
		}
	}
	return total, sponsors, rows.Err()
}

func (m reportMeasuredSponsors) apply(facts map[string]*float64, details *reportDetails, sponsors []reportMeasuredSponsors) map[string]string {
	details.Sources = append(details.Sources, reportSourceRows{Table: "sponsor_measured_exposures", Rows: m.rows})
	notes := map[string]string{}
	// A single population supplies reach and CTR. The historical signals
	// identify devices differently, so their counts cannot be added to these.
	if m.htmlImpressions > 0 {
		const exposureNote = "Esposizioni HTML qualificate registrate per l'evento in sponsor_measured_exposures."
		facts["sponsor_reach"], facts["sponsor_clickers"] = nil, nil
		facts["sponsor_impressions"] = reportMetricFloat(float64(m.viewable))
		notes["sponsor_impressions"] = exposureNote
		if m.unknownReach == 0 {
			facts["sponsor_reach"] = reportMetricFloat(float64(m.reach))
			notes["sponsor_reach"] = exposureNote + " Dispositivi deduplicati anche tra sponsor."
		}
		if m.unknownReach == 0 && m.anonymousClicks == 0 && m.clickable > 0 {
			facts["sponsor_clickers"] = reportMetricFloat(float64(m.exposedClickers))
			notes["sponsor_ctr"] = exposureNote + " Il numeratore conta i dispositivi con un clic sullo stesso sponsor visto."
		}
		details.CoverageNotes = append(details.CoverageNotes, "Copertura e CTR sugli esposti usano le esposizioni HTML qualificate. Il collegamento tra questi dispositivi e i round di gioco è indisponibile.")
		details.Sponsors = make([]reportSponsor, 0, len(sponsors))
		for _, sponsor := range sponsors {
			details.Sponsors = append(details.Sponsors, sponsor.detail())
		}
	}
	if existing := facts["total_sponsor_clicks"]; m.clickable > 0 && (m.clicks > 0 || existing == nil || *existing == 0) {
		facts["total_sponsor_clicks"] = reportMetricFloat(float64(m.clicks))
		notes["sponsor_clicks"] = "Clic salvati dalla misurazione sponsor in sponsor_measured_exposures, inclusi i clic senza identità. Il totale usa una sola sorgente."
		details.Operational.SponsorClicks = m.clicks
		details.Operational.SponsorClicksWithoutIdentity = m.anonymousClicks
	}
	if m.clicks > 0 && m.anonymousClicks == 0 && (facts["sponsor_interactors"] == nil || *facts["sponsor_interactors"] == 0) {
		facts["sponsor_interactors"] = reportMetricFloat(float64(m.clickers))
		notes["sponsor_interactors"] = "Dispositivi unici con clic salvati dalla misurazione sponsor. È un minimo osservato."
	}
	return notes
}

func (m reportMeasuredSponsors) detail() reportSponsor {
	detail := reportSponsor{ID: m.id, Name: m.name, Impressions: m.viewable, ExposureMeasured: m.htmlImpressions > 0}
	if detail.ExposureMeasured && m.unknownReach == 0 {
		detail.Reach = &m.reach
		if m.anonymousClicks == 0 && m.clickable > 0 {
			detail.Clickers = &m.exposedClickers
			detail.CTR = reportMetricRatio(reportMetricFloat(float64(m.exposedClickers)), reportMetricFloat(float64(m.reach)), true)
		}
	}
	return detail
}
