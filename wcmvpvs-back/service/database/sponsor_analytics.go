package database

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
)

const (
	SponsorImpression       = "sponsor_impression"
	SponsorViewable         = "sponsor_viewable"
	SponsorGeometryViewable = "sponsor_geometry_viewable"
	SponsorExposureSample   = "sponsor_exposure_sample"
	SponsorExposureEnd      = "sponsor_exposure_end"
	SponsorClick            = "sponsor_click"
	SponsorFullView         = "sponsor_full_view"
	SponsorHTML             = "html_display"
	SponsorSVG              = "svg_geometry"
	SponsorInGame           = "game_geometry"
	SponsorPostVoteDuration = 4000
)

type SponsorAnalyticsEvent struct {
	ID         string `json:"event_id"`
	Type       string `json:"type"`
	ClubID     int    `json:"club_id"`
	MatchID    int    `json:"match_id"`
	SponsorID  int    `json:"sponsor_id"`
	CreativeID string `json:"creative_id"`
	Placement  string `json:"placement_id"`
	Game       string `json:"game_id"`
	SessionID  string `json:"session_id"`
	ExposureID string `json:"exposure_id"`
	Timestamp  int64  `json:"timestamp"`
	Duration   int    `json:"visible_duration_ms"`
	Continuous int    `json:"continuous_visible_ms"`
	Percentage int    `json:"visibility_percentage"`
	Method     string `json:"measurement_method"`
	Grant      string `json:"grant"`
	GeometrySampled bool `json:"geometry_sampled,omitempty"`

	// Populated only after validating the signed catalog.
	SponsorName  string `json:"-"`
	Clickable    bool   `json:"-"`
	FullDuration int    `json:"-"`
	ReportDeviceKey string `json:"-"`
}

func ensureSponsorAnalytics(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS sponsor_measured_exposures (
			id TEXT PRIMARY KEY,
			club_id INTEGER NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
			match_id INTEGER REFERENCES events(id) ON DELETE CASCADE,
			sponsor_id INTEGER NOT NULL,
			sponsor_name TEXT NOT NULL,
			creative_id TEXT NOT NULL,
			placement_id TEXT NOT NULL,
			game_id TEXT NOT NULL,
			method TEXT NOT NULL,
			session_id TEXT NOT NULL,
			audience_key TEXT NOT NULL,
			start_ms INTEGER NOT NULL,
			last_ms INTEGER NOT NULL,
			visible_ms INTEGER NOT NULL DEFAULT 0,
			continuous_ms INTEGER NOT NULL DEFAULT 0,
			max_percentage INTEGER NOT NULL,
			impression INTEGER NOT NULL,
			viewable INTEGER NOT NULL DEFAULT 0,
			ended INTEGER NOT NULL DEFAULT 0,
			full_view INTEGER NOT NULL DEFAULT 0,
			full_duration INTEGER NOT NULL,
			clickable INTEGER NOT NULL,
			clicks INTEGER NOT NULL DEFAULT 0
		);
		CREATE INDEX IF NOT EXISTS idx_sponsor_measured_period ON sponsor_measured_exposures(club_id, start_ms);
		CREATE INDEX IF NOT EXISTS idx_sponsor_measured_sponsor ON sponsor_measured_exposures(club_id, sponsor_id, start_ms);
		CREATE INDEX IF NOT EXISTS idx_sponsor_measured_match ON sponsor_measured_exposures(match_id, club_id, start_ms);
		CREATE TABLE IF NOT EXISTS sponsor_measured_event_ids (
			id TEXT PRIMARY KEY,
			exposure_id TEXT NOT NULL REFERENCES sponsor_measured_exposures(id) ON DELETE CASCADE
		);
		CREATE INDEX IF NOT EXISTS idx_sponsor_measured_event_exposure ON sponsor_measured_event_ids(exposure_id);
	`)
	if err != nil {
		return err
	}
	for _, statement := range []string{
		`ALTER TABLE sponsor_measured_exposures ADD COLUMN report_device_key TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE sponsor_measured_exposures ADD COLUMN geometry_sampled INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE sponsor_measured_exposures ADD COLUMN geometry_viewable INTEGER NOT NULL DEFAULT 0`,
	} {
		if _, err := db.Exec(statement); err != nil && !strings.Contains(strings.ToLower(err.Error()), "duplicate column name") {
			return err
		}
	}
	return nil
}

type measuredSponsorState struct {
	club, match, sponsor, fullDuration                   int
	creative, placement, game, method, session, audience string
	start, last                                          int64
	duration, continuous, clicks                         int
	impression, viewable, ended, full, clickable         bool
	geometrySampled, geometryViewable                    bool
}

func (s measuredSponsorState) accepts(e SponsorAnalyticsEvent, audience string) bool {
	if s.club != e.ClubID || s.match != e.MatchID || s.sponsor != e.SponsorID ||
		s.creative != e.CreativeID || s.placement != e.Placement || s.game != e.Game ||
		s.method != e.Method || s.session != e.SessionID || s.audience != audience ||
		s.ended || !s.impression || e.Timestamp < s.last || e.Timestamp-s.start > 24*60*60*1000 ||
		e.Duration < s.duration || e.Continuous < s.continuous ||
		int64(e.Duration-s.duration) > e.Timestamp-s.last+1 || int64(e.Duration) > e.Timestamp-s.start+1 {
		return false
	}
	switch e.Type {
	case SponsorViewable:
		return s.method == SponsorHTML && !s.viewable && e.Continuous >= 1000 && e.Percentage >= 50
	case SponsorGeometryViewable:
		return s.geometrySampled && !s.geometryViewable && s.method != SponsorHTML && e.Continuous >= 1000 && e.Percentage >= 50
	case SponsorExposureSample:
		return true
	case SponsorFullView:
		return s.viewable && !s.full && s.fullDuration > 0 && e.Continuous >= s.fullDuration && e.Percentage >= 50
	case SponsorExposureEnd:
		return s.method != SponsorHTML || s.viewable || e.Continuous < 1000
	case SponsorClick:
		return s.clickable && s.clicks < 30
	}
	return false
}

// RecordSponsorAnalytics applies a batch atomically. Invalid transitions are
// discarded individually so one expired creative cannot strand valid events.
func (db *appdbimpl) RecordSponsorAnalytics(ctx context.Context, audience string, events []SponsorAnalyticsEvent) (int, error) {
	if len(events) == 0 {
		return 0, nil
	}
	// Keep simultaneous batches from competing to upgrade SQLite read
	// snapshots to write transactions during a match-wide burst.
	db.sponsorAnalyticsMu.Lock()
	defer db.sponsorAnalyticsMu.Unlock()

	tx, err := db.c.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	var scopeExists bool
	err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM organizations WHERE id=?
		AND (?=0 OR EXISTS(SELECT 1 FROM events WHERE id=? AND organization_id=organizations.id)))`,
		events[0].ClubID, events[0].MatchID, events[0].MatchID).Scan(&scopeExists)
	if err != nil || !scopeExists {
		return 0, err
	}
	accepted := 0
	for _, e := range events {
		if e.ClubID != events[0].ClubID || e.MatchID != events[0].MatchID {
			continue
		}
		var duplicate int
		err := tx.QueryRowContext(ctx, `SELECT 1 FROM sponsor_measured_event_ids WHERE id=?`, e.ID).Scan(&duplicate)
		if err == nil {
			continue
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return 0, err
		}

		var state measuredSponsorState
		err = tx.QueryRowContext(ctx, `
			SELECT club_id, COALESCE(match_id,0), sponsor_id, creative_id, placement_id, game_id, method,
				session_id, audience_key, start_ms, last_ms, visible_ms, continuous_ms, impression,
				viewable, ended, full_view, full_duration, clickable, clicks, geometry_sampled, geometry_viewable
			FROM sponsor_measured_exposures WHERE id=?`, e.ExposureID).Scan(
			&state.club, &state.match, &state.sponsor, &state.creative, &state.placement, &state.game,
			&state.method, &state.session, &state.audience, &state.start, &state.last,
			&state.duration, &state.continuous, &state.impression, &state.viewable, &state.ended,
			&state.full, &state.fullDuration, &state.clickable, &state.clicks, &state.geometrySampled, &state.geometryViewable)
		switch {
		case errors.Is(err, sql.ErrNoRows):
			impression := e.Type == SponsorImpression && e.Percentage > 0
			click := e.Type == SponsorClick && e.Clickable
			if (!impression && !click) || e.Duration != 0 || e.Continuous != 0 {
				continue
			}
			_, err = tx.ExecContext(ctx, `
				INSERT INTO sponsor_measured_exposures (
					id, club_id, match_id, sponsor_id, sponsor_name, creative_id, placement_id, game_id,
					method, session_id, audience_key, start_ms, last_ms, max_percentage, impression, full_duration, clickable, clicks,
					report_device_key, geometry_sampled
				) VALUES (?,?,NULLIF(?,0),?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
				e.ExposureID, e.ClubID, e.MatchID, e.SponsorID, e.SponsorName, e.CreativeID, e.Placement, e.Game,
				e.Method, e.SessionID, audience, e.Timestamp, e.Timestamp, e.Percentage, impression,
				e.FullDuration, e.Clickable, boolToInt(click), e.ReportDeviceKey, e.GeometrySampled && e.Method != SponsorHTML)
		case err != nil:
			return 0, err
		case state.accepts(e, audience):
			_, err = tx.ExecContext(ctx, `UPDATE sponsor_measured_exposures
				SET last_ms=?, visible_ms=?, continuous_ms=?, max_percentage=MAX(max_percentage,?),
					viewable=(viewable OR ?), ended=?, full_view=(full_view OR ?), clicks=clicks+?,
					report_device_key=CASE WHEN report_device_key='' THEN ? ELSE report_device_key END,
					geometry_viewable=(geometry_viewable OR ?)
				WHERE id=?`, e.Timestamp, e.Duration, e.Continuous, e.Percentage,
				e.Type == SponsorViewable, e.Type == SponsorExposureEnd, e.Type == SponsorFullView,
				boolToInt(e.Type == SponsorClick), e.ReportDeviceKey, e.Type == SponsorGeometryViewable, e.ExposureID)
		default:
			continue
		}
		if err != nil {
			return 0, err
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO sponsor_measured_event_ids (id, exposure_id) VALUES (?,?)`, e.ID, e.ExposureID); err != nil {
			return 0, err
		}
		accepted++
	}
	return accepted, tx.Commit()
}

type SponsorAnalyticsFilter struct {
	ClubID, MatchID, SponsorID int
	From, Until                time.Time
}

type SponsorAnalyticsMetrics struct {
	Impressions             int      `json:"impressions"`
	HTMLImpressions         int      `json:"html_impressions"`
	GeometricImpressions    int      `json:"geometric_impressions"`
	Viewable                *int     `json:"viewable_impressions"`
	Viewability             *float64 `json:"viewability_rate"`
	Reach                   *int     `json:"reach"`
	Frequency               *float64 `json:"frequency"`
	TotalDuration           *int64   `json:"total_visible_ms"`
	AverageDuration         *float64 `json:"average_visible_ms"`
	DurationSamples         int      `json:"duration_samples"`
	PositiveDurationSamples int      `json:"-"`
	GeometrySampled         int      `json:"-"`
	GeometryViewable        int      `json:"-"`
	ObservedDuration        *int64   `json:"-"`
	ObservedDurationSamples int      `json:"-"`
	Clicks                  *int     `json:"clicks"`
	ClickableImpressions    int      `json:"-"`
	ClicksOnImpressions     int      `json:"-"`
	UniqueClicks            *int     `json:"unique_clickers"`
	CTR                     *float64 `json:"ctr"`
	FullViews               *int     `json:"full_views"`
	FullRate                *float64 `json:"full_view_rate"`
	Conversions             *int     `json:"conversions"`
}

type SponsorAnalyticsRow struct {
	SponsorID   int    `json:"sponsor_id,omitempty"`
	SponsorName string `json:"sponsor_name,omitempty"`
	Creative    string `json:"creative_id,omitempty"`
	Placement   string `json:"placement_id,omitempty"`
	Game        string `json:"game_id,omitempty"`
	Method      string `json:"measurement_method,omitempty"`
	Day         string `json:"day,omitempty"`
	SponsorAnalyticsMetrics
}

type SponsorAnalyticsReport struct {
	From       string                  `json:"from"`
	Until      string                  `json:"until"`
	Total      SponsorAnalyticsMetrics `json:"total"`
	Sponsors   []SponsorAnalyticsRow   `json:"sponsors"`
	Placements []SponsorAnalyticsRow   `json:"placements"`
	Creatives  []SponsorAnalyticsRow   `json:"creatives"`
	Trend      []SponsorAnalyticsRow   `json:"trend"`
}

func sponsorRatio(numerator, denominator int, scale float64) *float64 {
	if denominator == 0 {
		return nil
	}
	value := float64(numerator) * scale / float64(denominator)
	return &value
}

// GetMeasuredSponsorAnalytics is also the aggregate data source for commercial
// reports. A period is defined by the UTC start of the exposure or standalone click.
func (db *appdbimpl) GetMeasuredSponsorAnalytics(ctx context.Context, f SponsorAnalyticsFilter) (SponsorAnalyticsReport, error) {
	report := SponsorAnalyticsReport{
		From: f.From.Format("2006-01-02"), Until: f.Until.Format("2006-01-02"),
		Sponsors: []SponsorAnalyticsRow{}, Placements: []SponsorAnalyticsRow{},
		Creatives: []SponsorAnalyticsRow{}, Trend: []SponsorAnalyticsRow{},
	}
	tx, err := db.c.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return report, err
	}
	defer tx.Rollback()

	args := []interface{}{f.ClubID, f.From.UnixMilli(), f.Until.UnixMilli()}
	filter := ` FROM sponsor_measured_exposures WHERE club_id=? AND start_ms>=? AND start_ms<?`
	if f.MatchID > 0 {
		filter += ` AND match_id=?`
		args = append(args, f.MatchID)
	}
	if f.SponsorID > 0 {
		filter += ` AND sponsor_id=?`
		args = append(args, f.SponsorID)
	}
	groups := []struct {
		columns, suffix string
		target          *[]SponsorAnalyticsRow
	}{
		{`0,'','','','','',''`, "", nil},
		{`sponsor_id, MAX(sponsor_name),'','','','', ''`, ` GROUP BY sponsor_id ORDER BY sponsor_id`, &report.Sponsors},
		{`0,'','',placement_id,game_id,method,''`, ` GROUP BY placement_id,game_id,method ORDER BY placement_id,game_id`, &report.Placements},
		{`sponsor_id, MAX(sponsor_name),creative_id,placement_id,game_id,method,''`, ` GROUP BY sponsor_id,creative_id,placement_id,game_id,method ORDER BY sponsor_id,placement_id,game_id,creative_id`, &report.Creatives},
		{`0,'','','','','',date(start_ms/1000,'unixepoch')`, ` GROUP BY date(start_ms/1000,'unixepoch') ORDER BY 7`, &report.Trend},
	}
	for _, group := range groups {
		rows, err := tx.QueryContext(ctx, `SELECT `+group.columns+measuredSponsorStatsSQL+filter+group.suffix, args...)
		if err != nil {
			return report, err
		}
		for rows.Next() {
			row, err := scanSponsorAnalyticsRow(rows)
			if err != nil {
				rows.Close()
				return report, err
			}
			if group.target == nil {
				report.Total = row.SponsorAnalyticsMetrics
			} else {
				*group.target = append(*group.target, row)
			}
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return report, err
		}
	}
	return report, tx.Commit()
}

const measuredSponsorStatsSQL = `, COALESCE(SUM(impression),0), COUNT(CASE WHEN impression AND method='html_display' THEN 1 END),
		COALESCE(SUM(viewable),0),
		COUNT(DISTINCT CASE WHEN impression THEN NULLIF(audience_key,'') END),
		COUNT(CASE WHEN impression AND audience_key='' THEN 1 END),
		SUM(CASE WHEN impression AND ended THEN visible_ms END),
		COUNT(CASE WHEN impression AND ended THEN 1 END),
		COUNT(CASE WHEN impression AND ended AND visible_ms>0 THEN 1 END),
		COALESCE(SUM(clicks),0), COUNT(CASE WHEN clickable THEN 1 END),
		COUNT(DISTINCT CASE WHEN clicks>0 THEN NULLIF(audience_key,'') END),
		COUNT(CASE WHEN clicks>0 AND audience_key='' THEN 1 END),
		COALESCE(SUM(full_view),0), COUNT(CASE WHEN full_duration>0 AND impression AND (ended OR full_view) THEN 1 END),
		COUNT(CASE WHEN impression AND geometry_sampled THEN 1 END),
		COUNT(CASE WHEN impression AND geometry_viewable THEN 1 END),
		SUM(CASE WHEN impression AND (ended OR visible_ms>0) THEN visible_ms END),
		COUNT(CASE WHEN impression AND (ended OR visible_ms>0) THEN 1 END),
		COUNT(CASE WHEN impression AND clickable THEN 1 END),
		COALESCE(SUM(CASE WHEN impression AND clickable THEN clicks END),0)`

// ReadMeasuredSponsorMatches groups the same measured data as the commercial
// report for several already scoped events in a single read.
func ReadMeasuredSponsorMatches(ctx context.Context, q interface {
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
}, clubID int, eventIDs []int) (map[int]*SponsorAnalyticsReport, error) {
	reports := make(map[int]*SponsorAnalyticsReport)
	if len(eventIDs) == 0 {
		return reports, nil
	}
	args := []interface{}{clubID}
	for _, id := range eventIDs {
		args = append(args, id)
	}
	in := strings.TrimSuffix(strings.Repeat("?,", len(eventIDs)), ",")
	rows, err := q.QueryContext(ctx, `WITH scoped AS (
		SELECT match_id, sponsor_id, sponsor_name, placement_id, game_id, method, impression, viewable,
			audience_key, ended, visible_ms, clicks, clickable, full_view, full_duration, geometry_sampled, geometry_viewable
		FROM sponsor_measured_exposures WHERE club_id=? AND match_id IN (`+in+`)
	), levels AS (SELECT 0 AS level UNION ALL SELECT 1 UNION ALL SELECT 2)
	SELECT match_id, level, CASE WHEN level=1 THEN sponsor_id ELSE 0 END AS grouped_sponsor,
		CASE WHEN level=1 THEN MAX(sponsor_name) ELSE '' END, '',
		CASE WHEN level=2 THEN placement_id ELSE '' END AS grouped_placement,
		CASE WHEN level=2 THEN game_id ELSE '' END AS grouped_game,
		CASE WHEN level=2 THEN method ELSE '' END AS grouped_method, ''
		`+measuredSponsorStatsSQL+`
	FROM scoped CROSS JOIN levels
	GROUP BY match_id, level, grouped_sponsor, grouped_placement, grouped_game, grouped_method
	ORDER BY match_id, level, grouped_sponsor, grouped_placement, grouped_game, grouped_method`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id, level int
		row, err := scanSponsorAnalyticsRow(rows, &id, &level)
		if err != nil {
			return nil, err
		}
		report := reports[id]
		if report == nil {
			report = &SponsorAnalyticsReport{}
			reports[id] = report
		}
		switch level {
		case 0:
			report.Total = row.SponsorAnalyticsMetrics
		case 1:
			report.Sponsors = append(report.Sponsors, row)
		case 2:
			report.Placements = append(report.Placements, row)
		}
	}
	return reports, rows.Err()
}

func scanSponsorAnalyticsRow(rows *sql.Rows, prefix ...interface{}) (SponsorAnalyticsRow, error) {
	var row SponsorAnalyticsRow
	m := &row.SponsorAnalyticsMetrics
	var viewable, reach, missingReach, clicks, clickable, uniqueClicks, missingClicks, full, fullDenominator, visibleSamples int
	var duration, observedDuration sql.NullInt64
	err := rows.Scan(append(prefix, &row.SponsorID, &row.SponsorName, &row.Creative, &row.Placement, &row.Game, &row.Method, &row.Day,
		&m.Impressions, &m.HTMLImpressions, &viewable, &reach, &missingReach, &duration, &m.DurationSamples, &visibleSamples,
		&clicks, &clickable, &uniqueClicks, &missingClicks, &full, &fullDenominator,
		&m.GeometrySampled, &m.GeometryViewable, &observedDuration, &m.ObservedDurationSamples,
		&m.ClickableImpressions, &m.ClicksOnImpressions)...)
	if err != nil {
		return row, err
	}
	m.GeometricImpressions = m.Impressions - m.HTMLImpressions
	if m.HTMLImpressions > 0 {
		m.Viewable = &viewable
		m.Viewability = sponsorRatio(viewable, m.HTMLImpressions, 100)
	}
	if m.Impressions > 0 && missingReach == 0 {
		m.Reach = &reach
		if m.Viewable != nil {
			m.Frequency = sponsorRatio(viewable, reach, 1)
		}
	}
	if duration.Valid {
		m.TotalDuration = &duration.Int64
		if visibleSamples > 0 {
			m.PositiveDurationSamples = visibleSamples
			average := float64(duration.Int64) / float64(visibleSamples)
			m.AverageDuration = &average
		}
	}
	if observedDuration.Valid {
		m.ObservedDuration = &observedDuration.Int64
	}
	if clickable > 0 {
		m.Clicks = &clicks
		if missingClicks == 0 {
			m.UniqueClicks = &uniqueClicks
		}
		m.CTR = sponsorRatio(clicks, viewable, 100)
	}
	if fullDenominator > 0 {
		m.FullViews = &full
		m.FullRate = sponsorRatio(full, fullDenominator, 100)
	}
	return row, nil
}
