package database

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// SaveFanProfile stores a marketing profile and returns the stored record with timestamps and ID.
func (db *appdbimpl) SaveFanProfile(profile FanProfile) (FanProfile, error) {
	now := time.Now().Format(time.RFC3339)
	profile.CreatedAt = now
	res, err := db.c.Exec(`INSERT INTO fan_profiles (
                organization_id, event_id, vote_id, fan_id, device_id, first_name, last_name, email, age_range, location,
                attendance_frequency, discovery_channel, sponsor_preference, promo_opt_in, contact_channel, interests, created_at
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		profile.OrganizationID, profile.EventID, profile.VoteID, strings.TrimSpace(profile.FanID), strings.TrimSpace(profile.DeviceID), strings.TrimSpace(profile.FirstName), strings.TrimSpace(profile.LastName), strings.TrimSpace(profile.Email), strings.TrimSpace(profile.AgeRange), strings.TrimSpace(profile.Location),
		strings.TrimSpace(profile.AttendanceFrequency), strings.TrimSpace(profile.DiscoveryChannel), strings.TrimSpace(profile.SponsorPreference), profile.PromoOptIn, strings.TrimSpace(profile.ContactChannel), strings.TrimSpace(profile.Interests), now,
	)
	if err != nil {
		return profile, fmt.Errorf("error saving fan profile: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return profile, fmt.Errorf("error retrieving fan profile id: %w", err)
	}
	profile.ID = int(id)
	return profile, nil
}

// UpsertFanProfile creates or updates a profile matching the provided identifiers.
func (db *appdbimpl) UpsertFanProfile(profile FanProfile) (FanProfile, error) {
	profile.Email = strings.TrimSpace(profile.Email)
	profile.FanID = strings.TrimSpace(profile.FanID)
	profile.DeviceID = strings.TrimSpace(profile.DeviceID)

	var existing FanProfile
	row := db.c.QueryRow(`SELECT id, fan_id, device_id, first_name, last_name, email, age_range, location, attendance_frequency, discovery_channel, sponsor_preference, promo_opt_in, contact_channel, interests FROM fan_profiles WHERE organization_id = ? AND (
                (fan_id IS NOT NULL AND fan_id != '' AND fan_id = ?) OR
                (email IS NOT NULL AND email != '' AND email = ?) OR
                (device_id IS NOT NULL AND device_id != '' AND device_id = ?)
        ) ORDER BY created_at DESC LIMIT 1`, profile.OrganizationID, profile.FanID, profile.Email, profile.DeviceID)
	switch err := row.Scan(&existing.ID, &existing.FanID, &existing.DeviceID, &existing.FirstName, &existing.LastName, &existing.Email, &existing.AgeRange, &existing.Location, &existing.AttendanceFrequency, &existing.DiscoveryChannel, &existing.SponsorPreference, &existing.PromoOptIn, &existing.ContactChannel, &existing.Interests); err {
	case nil:
		profile.ID = existing.ID
	case sql.ErrNoRows:
		// insert new profile
		return db.SaveFanProfile(profile)
	default:
		return profile, fmt.Errorf("error searching existing fan profile: %w", err)
	}

	_, err := db.c.Exec(`UPDATE fan_profiles SET fan_id = COALESCE(NULLIF(?, ''), fan_id), device_id = COALESCE(NULLIF(?, ''), device_id), first_name = COALESCE(NULLIF(?, ''), first_name), last_name = COALESCE(NULLIF(?, ''), last_name), email = COALESCE(NULLIF(?, ''), email), age_range = COALESCE(NULLIF(?, ''), age_range), location = COALESCE(NULLIF(?, ''), location), attendance_frequency = COALESCE(NULLIF(?, ''), attendance_frequency), discovery_channel = COALESCE(NULLIF(?, ''), discovery_channel), sponsor_preference = COALESCE(NULLIF(?, ''), sponsor_preference), promo_opt_in = ?, contact_channel = COALESCE(NULLIF(?, ''), contact_channel), interests = COALESCE(NULLIF(?, ''), interests) WHERE id = ?`,
		profile.FanID, profile.DeviceID, strings.TrimSpace(profile.FirstName), strings.TrimSpace(profile.LastName), profile.Email, strings.TrimSpace(profile.AgeRange), strings.TrimSpace(profile.Location), strings.TrimSpace(profile.AttendanceFrequency), strings.TrimSpace(profile.DiscoveryChannel), strings.TrimSpace(profile.SponsorPreference), profile.PromoOptIn, strings.TrimSpace(profile.ContactChannel), strings.TrimSpace(profile.Interests), profile.ID)
	if err != nil {
		return profile, fmt.Errorf("error updating fan profile: %w", err)
	}
	return profile, nil
}

// ListFanProfilesWithStats returns all profiles for the organization enriched with aggregated points and consents.
func (db *appdbimpl) ListFanProfilesWithStats(organizationID int) ([]FanProfile, error) {
	rows, err := db.c.Query(`
SELECT fp.id, fp.organization_id, IFNULL(fp.event_id, 0), IFNULL(fp.vote_id, 0), IFNULL(fp.fan_id, ''), IFNULL(fp.device_id, ''), IFNULL(fp.first_name, ''), IFNULL(fp.last_name, ''), IFNULL(fp.email, ''),
       IFNULL(fp.age_range, ''), IFNULL(fp.location, ''), IFNULL(fp.attendance_frequency, ''), IFNULL(fp.discovery_channel, ''), IFNULL(fp.sponsor_preference, ''),
       fp.promo_opt_in, IFNULL(fp.contact_channel, ''), IFNULL(fp.interests, ''), IFNULL(fp.created_at, ''),
       COALESCE((SELECT SUM(points) FROM fan_gamification_events WHERE fan_profile_id = fp.id), 0) AS total_points,
       IFNULL(fc.consent_club, 0), IFNULL(fc.consent_sponsors, 0), IFNULL(fc.consent_analytics, 0),
       (SELECT source FROM fan_gamification_events WHERE fan_profile_id = fp.id ORDER BY created_at DESC LIMIT 1) AS last_activity,
       (SELECT created_at FROM fan_gamification_events WHERE fan_profile_id = fp.id ORDER BY created_at DESC LIMIT 1) AS last_activity_at,
       (SELECT GROUP_CONCAT(name, ',') FROM marketing_badges mb WHERE mb.organization_id = fp.organization_id AND mb.threshold <= COALESCE((SELECT SUM(points) FROM fan_gamification_events WHERE fan_profile_id = fp.id), 0)) AS badges
FROM fan_profiles fp
LEFT JOIN fan_consents fc ON fc.id = (SELECT id FROM fan_consents WHERE fan_profile_id = fp.id ORDER BY created_at DESC LIMIT 1)
WHERE fp.organization_id = ?
GROUP BY fp.id
ORDER BY fp.created_at DESC`, organizationID)
	if err != nil {
		return nil, fmt.Errorf("error listing fan profiles: %w", err)
	}
	defer rows.Close()

	profiles := []FanProfile{}
	for rows.Next() {
		var p FanProfile
		var createdAt sql.NullString
		var eventID, voteID int
		var badgeList sql.NullString
		if err := rows.Scan(&p.ID, &p.OrganizationID, &eventID, &voteID, &p.FanID, &p.DeviceID, &p.FirstName, &p.LastName, &p.Email, &p.AgeRange, &p.Location, &p.AttendanceFrequency, &p.DiscoveryChannel, &p.SponsorPreference, &p.PromoOptIn, &p.ContactChannel, &p.Interests, &createdAt, &p.Points, &p.ConsentClub, &p.ConsentSponsors, &p.ConsentAnalytics, &p.LastActivity, &p.LastActivityAt, &badgeList); err != nil {
			return nil, fmt.Errorf("error parsing fan profile: %w", err)
		}
		p.EventID = eventID
		p.VoteID = voteID
		if createdAt.Valid {
			p.CreatedAt = createdAt.String
		}
		if badgeList.Valid && badgeList.String != "" {
			p.Badges = strings.Split(badgeList.String, ",")
		}
		profiles = append(profiles, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error reading fan profiles: %w", err)
	}
	return profiles, nil
}

// RecordGamificationEvent stores a gamification entry for a fan profile.
func (db *appdbimpl) RecordGamificationEvent(event GamificationEvent) error {
	_, err := db.c.Exec(`INSERT INTO fan_gamification_events (organization_id, fan_profile_id, event_id, source, points, season_label) VALUES (?, ?, ?, ?, ?, ?)`,
		event.OrganizationID, event.FanProfileID, event.EventID, strings.TrimSpace(event.Source), event.Points, strings.TrimSpace(event.SeasonLabel))
	if err != nil {
		return fmt.Errorf("error recording gamification event: %w", err)
	}
	return nil
}

// ListGamificationEvents returns gamification logs for an organization.
func (db *appdbimpl) ListGamificationEvents(organizationID int) ([]GamificationEvent, error) {
	rows, err := db.c.Query(`SELECT id, organization_id, fan_profile_id, IFNULL(event_id, 0), IFNULL(source, ''), IFNULL(points, 0), IFNULL(season_label, ''), IFNULL(created_at, '')
FROM fan_gamification_events WHERE organization_id = ? ORDER BY created_at DESC`, organizationID)
	if err != nil {
		return nil, fmt.Errorf("error listing gamification events: %w", err)
	}
	defer rows.Close()

	events := []GamificationEvent{}
	for rows.Next() {
		var e GamificationEvent
		var eventID int
		if err := rows.Scan(&e.ID, &e.OrganizationID, &e.FanProfileID, &eventID, &e.Source, &e.Points, &e.SeasonLabel, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("error parsing gamification event: %w", err)
		}
		e.EventID = eventID
		events = append(events, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error reading gamification events: %w", err)
	}
	return events, nil
}

// HasGamificationEvent checks if a specific source has been logged for a profile and event.
func (db *appdbimpl) HasGamificationEvent(profileID int, source string, eventID int) (bool, error) {
	row := db.c.QueryRow(`SELECT 1 FROM fan_gamification_events WHERE fan_profile_id = ? AND source = ? AND IFNULL(event_id, 0) = ? LIMIT 1`, profileID, strings.TrimSpace(source), eventID)
	var marker int
	switch err := row.Scan(&marker); err {
	case nil:
		return true, nil
	case sql.ErrNoRows:
		return false, nil
	default:
		return false, err
	}
}

// GetFanProfileWithStats returns a single profile enriched with points and badges.
func (db *appdbimpl) GetFanProfileWithStats(organizationID int, fanID, deviceID, email string) (FanProfile, error) {
	var profile FanProfile
	fanID = strings.TrimSpace(fanID)
	deviceID = strings.TrimSpace(deviceID)
	email = strings.TrimSpace(email)

	row := db.c.QueryRow(`SELECT fp.id, fp.organization_id, IFNULL(fp.event_id, 0), IFNULL(fp.vote_id, 0), IFNULL(fp.fan_id, ''), IFNULL(fp.device_id, ''), IFNULL(fp.first_name, ''), IFNULL(fp.last_name, ''), IFNULL(fp.email, ''),
       IFNULL(fp.age_range, ''), IFNULL(fp.location, ''), IFNULL(fp.attendance_frequency, ''), IFNULL(fp.discovery_channel, ''), IFNULL(fp.sponsor_preference, ''),
       fp.promo_opt_in, IFNULL(fp.contact_channel, ''), IFNULL(fp.interests, ''), IFNULL(fp.created_at, ''),
       COALESCE((SELECT SUM(points) FROM fan_gamification_events WHERE fan_profile_id = fp.id), 0) AS total_points,
       IFNULL(fc.consent_club, 0), IFNULL(fc.consent_sponsors, 0), IFNULL(fc.consent_analytics, 0),
       (SELECT source FROM fan_gamification_events WHERE fan_profile_id = fp.id ORDER BY created_at DESC LIMIT 1) AS last_activity,
       (SELECT created_at FROM fan_gamification_events WHERE fan_profile_id = fp.id ORDER BY created_at DESC LIMIT 1) AS last_activity_at,
       (SELECT GROUP_CONCAT(name, ',') FROM marketing_badges mb WHERE mb.organization_id = fp.organization_id AND mb.threshold <= COALESCE((SELECT SUM(points) FROM fan_gamification_events WHERE fan_profile_id = fp.id), 0)) AS badges
FROM fan_profiles fp
LEFT JOIN fan_consents fc ON fc.id = (SELECT id FROM fan_consents WHERE fan_profile_id = fp.id ORDER BY created_at DESC LIMIT 1)
WHERE fp.organization_id = ? AND (
        (fp.fan_id IS NOT NULL AND fp.fan_id != '' AND fp.fan_id = ?) OR
        (fp.device_id IS NOT NULL AND fp.device_id != '' AND fp.device_id = ?) OR
        (fp.email IS NOT NULL AND fp.email != '' AND fp.email = ?)
)
ORDER BY fp.created_at DESC
LIMIT 1`, organizationID, fanID, deviceID, email)

	var createdAt sql.NullString
	var eventID, voteID int
	var badges sql.NullString
	if err := row.Scan(&profile.ID, &profile.OrganizationID, &eventID, &voteID, &profile.FanID, &profile.DeviceID, &profile.FirstName, &profile.LastName, &profile.Email, &profile.AgeRange, &profile.Location, &profile.AttendanceFrequency, &profile.DiscoveryChannel, &profile.SponsorPreference, &profile.PromoOptIn, &profile.ContactChannel, &profile.Interests, &createdAt, &profile.Points, &profile.ConsentClub, &profile.ConsentSponsors, &profile.ConsentAnalytics, &profile.LastActivity, &profile.LastActivityAt, &badges); err != nil {
		return profile, err
	}
	profile.EventID = eventID
	profile.VoteID = voteID
	if createdAt.Valid {
		profile.CreatedAt = createdAt.String
	}
	if badges.Valid && badges.String != "" {
		profile.Badges = strings.Split(badges.String, ",")
	}
	return profile, nil
}

// SavePrivacyPolicy stores a privacy policy version for a club.
func (db *appdbimpl) SavePrivacyPolicy(policy PrivacyPolicy) (PrivacyPolicy, error) {
	now := time.Now().Format(time.RFC3339)
	policy.CreatedAt = now
	res, err := db.c.Exec(`INSERT OR REPLACE INTO privacy_policies (id, organization_id, version, title, summary, link, created_at) VALUES ((SELECT id FROM privacy_policies WHERE organization_id = ? AND version = ?), ?, ?, ?, ?, ?, ?)`,
		policy.OrganizationID, strings.TrimSpace(policy.Version), policy.OrganizationID, strings.TrimSpace(policy.Version), strings.TrimSpace(policy.Title), strings.TrimSpace(policy.Summary), strings.TrimSpace(policy.Link), now)
	if err != nil {
		return policy, fmt.Errorf("error saving privacy policy: %w", err)
	}
	id, err := res.LastInsertId()
	if err == nil {
		policy.ID = int(id)
	}
	return policy, nil
}

// ListPrivacyPolicies returns all privacy policies for a club.
func (db *appdbimpl) ListPrivacyPolicies(organizationID int) ([]PrivacyPolicy, error) {
	rows, err := db.c.Query(`SELECT id, organization_id, version, title, IFNULL(summary, ''), IFNULL(link, ''), IFNULL(created_at, '') FROM privacy_policies WHERE organization_id = ? ORDER BY created_at DESC`, organizationID)
	if err != nil {
		return nil, fmt.Errorf("error listing privacy policies: %w", err)
	}
	defer rows.Close()

	policies := []PrivacyPolicy{}
	for rows.Next() {
		var p PrivacyPolicy
		if err := rows.Scan(&p.ID, &p.OrganizationID, &p.Version, &p.Title, &p.Summary, &p.Link, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("error parsing privacy policy: %w", err)
		}
		policies = append(policies, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error reading privacy policies: %w", err)
	}
	return policies, nil
}

// LogFanConsent saves a granular consent state for a profile.
func (db *appdbimpl) LogFanConsent(consent FanConsent) (FanConsent, error) {
	now := time.Now().Format(time.RFC3339)
	consent.CreatedAt = now
	res, err := db.c.Exec(`INSERT INTO fan_consents (organization_id, fan_profile_id, policy_version, consent_club, consent_sponsors, consent_analytics, ip, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		consent.OrganizationID, consent.FanProfileID, strings.TrimSpace(consent.PolicyVersion), consent.ConsentClub, consent.ConsentSponsors, consent.ConsentAnalytics, strings.TrimSpace(consent.IP), now)
	if err != nil {
		return consent, fmt.Errorf("error saving consent: %w", err)
	}
	id, err := res.LastInsertId()
	if err == nil {
		consent.ID = int(id)
	}
	return consent, nil
}

// GetConsentBreakdown aggregates consents per organization.
func (db *appdbimpl) GetConsentBreakdown(organizationID int) (ConsentBreakdown, error) {
	breakdown := ConsentBreakdown{}
	rows, err := db.c.Query(`SELECT consent_club, consent_sponsors, consent_analytics FROM fan_consents WHERE organization_id = ?`, organizationID)
	if err != nil {
		return breakdown, fmt.Errorf("error reading consent breakdown: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var club, sponsors, analytics bool
		if err := rows.Scan(&club, &sponsors, &analytics); err != nil {
			return breakdown, fmt.Errorf("error parsing consent row: %w", err)
		}
		breakdown.Total++
		if club {
			breakdown.ConsentClubYes++
		} else {
			breakdown.ConsentClubNo++
		}
		if sponsors {
			breakdown.ConsentSponsorsYes++
		} else {
			breakdown.ConsentSponsorsNo++
		}
		if analytics {
			breakdown.ConsentAnalyticsYes++
		} else {
			breakdown.ConsentAnalyticsNo++
		}
	}
	if err := rows.Err(); err != nil {
		return breakdown, fmt.Errorf("error processing consent breakdown: %w", err)
	}
	return breakdown, nil
}

// SaveMarketingBadge stores a badge definition.
func (db *appdbimpl) SaveMarketingBadge(badge MarketingBadge) (MarketingBadge, error) {
	now := time.Now().Format(time.RFC3339)
	badge.CreatedAt = now
	res, err := db.c.Exec(`INSERT INTO marketing_badges (organization_id, name, threshold, description, created_at) VALUES (?, ?, ?, ?, ?)`,
		badge.OrganizationID, strings.TrimSpace(badge.Name), badge.Threshold, strings.TrimSpace(badge.Description), now)
	if err != nil {
		return badge, fmt.Errorf("error saving marketing badge: %w", err)
	}
	id, err := res.LastInsertId()
	if err == nil {
		badge.ID = int(id)
	}
	return badge, nil
}

// ListMarketingBadges returns badge definitions for the organization.
func (db *appdbimpl) ListMarketingBadges(organizationID int) ([]MarketingBadge, error) {
	rows, err := db.c.Query(`SELECT id, organization_id, name, threshold, IFNULL(description, ''), IFNULL(created_at, '') FROM marketing_badges WHERE organization_id = ? ORDER BY threshold ASC`, organizationID)
	if err != nil {
		return nil, fmt.Errorf("error listing marketing badges: %w", err)
	}
	defer rows.Close()

	badges := []MarketingBadge{}
	for rows.Next() {
		var b MarketingBadge
		if err := rows.Scan(&b.ID, &b.OrganizationID, &b.Name, &b.Threshold, &b.Description, &b.CreatedAt); err != nil {
			return nil, fmt.Errorf("error parsing marketing badge: %w", err)
		}
		badges = append(badges, b)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error reading marketing badges: %w", err)
	}
	return badges, nil
}

// BuildGamificationSummary aggregates gamification events.
func BuildGamificationSummary(profiles []FanProfile, events []GamificationEvent) GamificationSummary {
	summary := GamificationSummary{Distribution: map[string]int{}, TopEvents: map[int]TopFanScore{}}
	profilePoints := make(map[int]int)
	for _, p := range profiles {
		profilePoints[p.ID] = p.Points
	}
	if len(events) > 0 {
		summary.TotalEvents = len(events)
	}
	for profileID, pts := range profilePoints {
		summary.TotalFans++
		summary.AveragePoints += float64(pts)
		switch {
		case pts >= 100:
			summary.Distribution[">=100"]++
		case pts >= 50:
			summary.Distribution["50-99"]++
		case pts >= 25:
			summary.Distribution["25-49"]++
		default:
			summary.Distribution["0-24"]++
		}
		if current, ok := summary.TopEvents[0]; !ok || pts > current.Points {
			summary.TopEvents[0] = TopFanScore{FanProfileID: profileID, Points: pts}
		}
	}
	if summary.TotalFans > 0 {
		summary.AveragePoints = summary.AveragePoints / float64(summary.TotalFans)
	}

	// derive per-event top fan
	perEvent := map[int]TopFanScore{}
	for _, ev := range events {
		current := perEvent[ev.EventID]
		current.EventID = ev.EventID
		if ev.Points > current.Points {
			current.Points = ev.Points
			current.FanProfileID = ev.FanProfileID
		}
		perEvent[ev.EventID] = current
	}
	if len(perEvent) > 0 {
		for eventID, score := range perEvent {
			summary.TopEvents[eventID] = score
		}
	}

	// top fans overall sorted by points
	type pair struct {
		id     int
		points int
	}
	top := []pair{}
	for id, pts := range profilePoints {
		top = append(top, pair{id: id, points: pts})
	}
	for i := 0; i < len(top); i++ {
		for j := i + 1; j < len(top); j++ {
			if top[j].points > top[i].points {
				top[i], top[j] = top[j], top[i]
			}
		}
	}
	for idx, p := range top {
		if idx >= 5 {
			break
		}
		summary.TopFans = append(summary.TopFans, TopFanScore{FanProfileID: p.id, Points: p.points})
	}

	return summary
}
