package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
	"github.com/go-chi/chi/v5"
)

type historyTimelineBucket struct {
	Start string `json:"start"`
	End   string `json:"end"`
	Label string `json:"label"`
	Votes int    `json:"votes"`
}

type eventHistoryPrize struct {
	ID               int    `json:"id"`
	Name             string `json:"name"`
	Position         int    `json:"position"`
	WinnerTicketCode string `json:"winner_ticket_code,omitempty"`
}

type eventFeedbackSummaryResponse struct {
	TotalResponses    int            `json:"total_responses"`
	Experience        map[string]int `json:"experience"`
	TeamSpirit        map[string]int `json:"team_spirit"`
	PerksInterest     map[string]int `json:"perks_interest"`
	MiniGamesInterest map[string]int `json:"mini_games_interest"`
	Suggestions       []string       `json:"suggestions"`
}

type eventHistoryEntry struct {
	ID                 int                           `json:"id"`
	Title              string                        `json:"title"`
	StartDateTime      string                        `json:"start_datetime"`
	Location           string                        `json:"location"`
	TotalVotes         int                           `json:"total_votes"`
	SponsorClicksTotal int                           `json:"sponsor_clicks_total"`
	MVP                *database.EventMVP            `json:"mvp,omitempty"`
	SponsorClicks      []database.SponsorClickStat   `json:"sponsor_clicks"`
	SponsorAnalytics   sponsorAnalyticsResponse      `json:"sponsor_analytics"`
	Timeline           []historyTimelineBucket       `json:"timeline"`
	HomeTeam           string                        `json:"home_team"`
	AwayTeam           string                        `json:"away_team"`
	Prizes             []eventHistoryPrize           `json:"prizes"`
	HasPrizeDraw       bool                          `json:"has_prize_draw"`
	FeedbackSummary    *eventFeedbackSummaryResponse `json:"feedback_summary,omitempty"`
}

type historyEntryWrapper struct {
	entry     eventHistoryEntry
	startTime time.Time
}

func (rt *_router) recordSponsorClick(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	eventID, err := strconv.Atoi(chi.URLParam(r, "eventId"))
	if err != nil || eventID <= 0 {
		ctx.Logger.WithField("event_id", chi.URLParam(r, "eventId")).Warn("invalid event id while tracking sponsor click")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	sponsorID, err := strconv.Atoi(chi.URLParam(r, "sponsorId"))
	if err != nil || sponsorID <= 0 {
		ctx.Logger.WithField("sponsor_id", chi.URLParam(r, "sponsorId")).Warn("invalid sponsor id while tracking sponsor click")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var payload struct {
		DeviceID string `json:"device_id"`
	}
	if r.Body != nil {
		defer r.Body.Close()
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&payload); err != nil && !errors.Is(err, io.EOF) {
			ctx.Logger.WithError(err).Warn("invalid sponsor click payload")
		}
	}

	deviceID := strings.TrimSpace(payload.DeviceID)
	if deviceID == "" {
		deviceID = rt.deviceIDFromRequest(r)
	}

	if err := rt.db.RecordSponsorClick(eventID, sponsorID, deviceID); err != nil {
		ctx.Logger.WithError(err).Warn("cannot record sponsor click")
	}

	w.WriteHeader(http.StatusNoContent)
}

func (rt *_router) buildEventHistoryEntry(ctx reqcontext.RequestContext, event database.Event) (*historyEntryWrapper, error) {
	startTime, normalizedStart := parseEventStart(event.StartDateTime)

	totalVotes, err := rt.db.GetEventVoteCount(event.ID)
	if err != nil {
		ctx.Logger.WithError(err).WithField("event_id", event.ID).Warn("cannot count votes for history")
		return nil, err
	}

	mvp, err := rt.db.GetEventMVP(event.ID)
	var mvpPtr *database.EventMVP
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			ctx.Logger.WithError(err).WithField("event_id", event.ID).Warn("cannot compute MVP for history")
		}
	} else {
		mvpPtr = &mvp
	}

	sponsorStats, err := rt.db.GetSponsorClickStats(event.ID)
	sponsorTotal := 0
	if err != nil {
		ctx.Logger.WithError(err).WithField("event_id", event.ID).Warn("cannot load sponsor stats for history")
		sponsorStats = []database.SponsorClickStat{}
	}
	for _, stat := range sponsorStats {
		sponsorTotal += stat.Clicks
	}

	sponsorSummary, err := rt.db.GetSponsorAnalytics(event.ID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		ctx.Logger.WithError(err).WithField("event_id", event.ID).Warn("cannot load sponsor analytics for history")
		sponsorSummary = database.SponsorAnalytics{}
	}
	sponsorAnalytics := buildSponsorAnalyticsResponse(sponsorSummary)

	voteTimestamps, err := rt.db.ListEventVoteTimestamps(event.ID)
	if err != nil {
		ctx.Logger.WithError(err).WithField("event_id", event.ID).Warn("cannot load vote timestamps for history")
		voteTimestamps = nil
	}

	prizeList, err := rt.db.ListEventPrizes(event.ID)
	if err != nil {
		ctx.Logger.WithError(err).WithField("event_id", event.ID).Warn("cannot load prizes for history")
		prizeList = nil
	}

	prizes := make([]eventHistoryPrize, 0, len(prizeList))
	hasPrizeDraw := false
	for _, prize := range prizeList {
		winnerCode := ""
		if prize.Winner != nil {
			winnerCode = strings.TrimSpace(prize.Winner.TicketCode)
			if winnerCode != "" {
				hasPrizeDraw = true
			}
		}

		prizes = append(prizes, eventHistoryPrize{
			ID:               prize.ID,
			Name:             strings.TrimSpace(prize.Name),
			Position:         prize.Position,
			WinnerTicketCode: winnerCode,
		})
	}

	sort.SliceStable(prizes, func(i, j int) bool {
		if prizes[i].Position == prizes[j].Position {
			return prizes[i].ID < prizes[j].ID
		}
		return prizes[i].Position < prizes[j].Position
	})

	if startTime.IsZero() && len(voteTimestamps) > 0 {
		startTime = voteTimestamps[0]
		normalizedStart = startTime.UTC().Format(time.RFC3339)
	}

	timeline := buildVoteTimeline(startTime, voteTimestamps)

	var feedbackSummaryPtr *eventFeedbackSummaryResponse
	if feedbackSummary, err := rt.db.GetEventFeedbackSummary(event.ID); err != nil {
		ctx.Logger.WithError(err).WithField("event_id", event.ID).Warn("cannot load feedback summary for history")
	} else {
		summaryResponse := buildEventFeedbackSummaryResponse(feedbackSummary)
		feedbackSummaryPtr = &summaryResponse
	}

	entry := eventHistoryEntry{
		ID:                 event.ID,
		Title:              buildEventTitle(event),
		StartDateTime:      normalizedStart,
		Location:           strings.TrimSpace(event.Location),
		TotalVotes:         totalVotes,
		SponsorClicksTotal: sponsorTotal,
		MVP:                mvpPtr,
		SponsorClicks:      sponsorStats,
		SponsorAnalytics:   sponsorAnalytics,
		Timeline:           timeline,
		HomeTeam:           strings.TrimSpace(event.Team1Name),
		AwayTeam:           strings.TrimSpace(event.Team2Name),
		Prizes:             prizes,
		HasPrizeDraw:       hasPrizeDraw,
		FeedbackSummary:    feedbackSummaryPtr,
	}

	return &historyEntryWrapper{entry: entry, startTime: startTime}, nil
}

func (rt *_router) getEventHistory(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	events, err := rt.db.ListEvents()
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot list events for history")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	wrappers := make([]historyEntryWrapper, 0, len(events))

	for _, event := range events {
		if !event.IsConcluded {
			continue
		}
		wrapper, err := rt.buildEventHistoryEntry(ctx, event)
		if err != nil {
			continue
		}
		wrappers = append(wrappers, *wrapper)
	}

	sort.SliceStable(wrappers, func(i, j int) bool {
		return wrappers[i].startTime.After(wrappers[j].startTime)
	})

	history := make([]eventHistoryEntry, 0, len(wrappers))
	for _, wrapper := range wrappers {
		history = append(history, wrapper.entry)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(history); err != nil {
		ctx.Logger.WithError(err).Error("cannot encode event history response")
	}
}

func (rt *_router) downloadEventHistoryReport(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	eventID, err := strconv.Atoi(chi.URLParam(r, "eventId"))
	if err != nil || eventID <= 0 {
		ctx.Logger.WithField("event_id", chi.URLParam(r, "eventId")).Warn("invalid event id for history report")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	events, err := rt.db.ListEvents()
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot list events for history report")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var target *database.Event
	for i := range events {
		if events[i].ID == eventID {
			target = &events[i]
			break
		}
	}

	if target == nil || !target.IsConcluded {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	wrapper, err := rt.buildEventHistoryEntry(ctx, *target)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	pdfBytes, err := buildEventHistoryPDF(wrapper.entry)
	if err != nil {
		ctx.Logger.WithError(err).WithField("event_id", eventID).Error("cannot generate history report")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	filename := buildHistoryReportFilename(wrapper.entry)
	if filename == "" {
		filename = fmt.Sprintf("evento-%d-report.pdf", wrapper.entry.ID)
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.Header().Set("Content-Length", strconv.Itoa(len(pdfBytes)))

	if _, err := w.Write(pdfBytes); err != nil {
		ctx.Logger.WithError(err).WithField("event_id", eventID).Warn("cannot write history report response")
	}
}

func (rt *_router) purgeEvent(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	if !strings.EqualFold(ctx.AdminRole, "superadmin") {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	eventID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil || eventID <= 0 {
		ctx.Logger.WithField("event_id", chi.URLParam(r, "id")).Warn("invalid event id while purging")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var payload struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		ctx.Logger.WithError(err).Warn("invalid payload while purging event")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(payload.Password) == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	admin, err := rt.db.GetAdminByID(ctx.AdminID)
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot retrieve admin while purging event")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if !adminPasswordMatches(admin.PasswordHash, payload.Password) {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	if err := rt.db.PurgeEventData(eventID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		ctx.Logger.WithError(err).Error("cannot purge event")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func parseEventStart(value string) (time.Time, string) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return time.Time{}, ""
	}

	candidates := []string{trimmed}
	if !strings.Contains(trimmed, "T") {
		candidates = append(candidates, strings.Replace(trimmed, " ", "T", 1))
	}

	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02T15:04",
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
	}

	for _, candidate := range candidates {
		for _, layout := range layouts {
			if ts, err := time.ParseInLocation(layout, candidate, time.UTC); err == nil {
				return ts, ts.UTC().Format(time.RFC3339)
			}
		}
	}

	return time.Time{}, trimmed
}

func buildEventTitle(event database.Event) string {
	home := strings.TrimSpace(event.Team1Name)
	away := strings.TrimSpace(event.Team2Name)
	switch {
	case home != "" && away != "":
		return fmt.Sprintf("%s - %s", home, away)
	case home != "":
		return home
	case away != "":
		return away
	default:
		return fmt.Sprintf("Evento #%d", event.ID)
	}
}

func buildVoteTimeline(start time.Time, votes []time.Time) []historyTimelineBucket {
	if start.IsZero() {
		return []historyTimelineBucket{}
	}

	bucketDuration := 15 * time.Minute
	totalWindow := 3 * time.Hour
	bucketCount := int(totalWindow / bucketDuration)
	if bucketCount <= 0 {
		bucketCount = 1
	}

	windowStart := start.Add(-1 * time.Hour)

	counts := make([]int, bucketCount)
	for _, ts := range votes {
		if ts.IsZero() {
			continue
		}

		diff := ts.Sub(windowStart)
		index := 0
		if diff > 0 {
			index = int(diff / bucketDuration)
			if index >= bucketCount {
				index = bucketCount - 1
			}
		}
		counts[index]++
	}

	timeline := make([]historyTimelineBucket, 0, bucketCount)
	for i := 0; i < bucketCount; i++ {
		bucketStart := windowStart.Add(time.Duration(i) * bucketDuration)
		bucketEnd := bucketStart.Add(bucketDuration)
		timeline = append(timeline, historyTimelineBucket{
			Start: bucketStart.UTC().Format(time.RFC3339),
			End:   bucketEnd.UTC().Format(time.RFC3339),
			Label: fmt.Sprintf("%s-%s", bucketStart.Format("15:04"), bucketEnd.Format("15:04")),
			Votes: counts[i],
		})
	}

	return timeline
}

func buildEventFeedbackSummaryResponse(summary database.EventFeedbackSummary) eventFeedbackSummaryResponse {
	cloneMap := func(source map[string]int) map[string]int {
		if source == nil {
			return map[string]int{}
		}
		cloned := make(map[string]int, len(source))
		for key, value := range source {
			cloned[key] = value
		}
		return cloned
	}

	suggestions := make([]string, 0, len(summary.Suggestions))
	for _, suggestion := range summary.Suggestions {
		trimmed := strings.TrimSpace(suggestion)
		if trimmed != "" {
			suggestions = append(suggestions, trimmed)
		}
	}

	return eventFeedbackSummaryResponse{
		TotalResponses:    summary.TotalResponses,
		Experience:        cloneMap(summary.ExperienceCounts),
		TeamSpirit:        cloneMap(summary.TeamSpiritCounts),
		PerksInterest:     cloneMap(summary.PerksInterestCounts),
		MiniGamesInterest: cloneMap(summary.MiniGamesInterestCounts),
		Suggestions:       suggestions,
	}
}

const (
	pdfPageWidth    = 595.0
	pdfPageHeight   = 842.0
	pdfLeftMargin   = 56.0
	pdfRightMargin  = 56.0
	pdfTopMargin    = 56.0
	pdfBottomMargin = 56.0
)

type pdfLine struct {
	Text    string
	Font    string
	Size    float64
	Indent  float64
	Spacing float64
}

type pdfLinePlacement struct {
	Text string
	Font string
	Size float64
	X    float64
	Y    float64
}

type pdfObject struct {
	id   int
	data []byte
}

type feedbackQuestionDefinition struct {
	ID      string
	Title   string
	Answers map[string]string
}

var feedbackQuestionDefinitions = []feedbackQuestionDefinition{
	{
		ID:    "experience",
		Title: "Com’è stata la tua esperienza di voto oggi?",
		Answers: map[string]string{
			"very_easy": "Facilissima",
			"easy":      "Abbastanza semplice",
			"complex":   "Un po’ macchinosa",
			"hard":      "Difficile",
		},
	},
	{
		ID:    "team_spirit",
		Title: "Ti sei sentito parte della squadra mentre sceglievi l’MVP del pubblico?",
		Answers: map[string]string{
			"very_much":  "Moltissimo",
			"quite":      "Abbastanza",
			"a_bit":      "Poco",
			"not_at_all": "Per niente",
		},
	},
	{
		ID:    "perks_interest",
		Title: "Quanto ti interessano i contenuti esclusivi dedicati ai tifosi?",
		Answers: map[string]string{
			"very_much":  "Molto",
			"some":       "Abbastanza",
			"little":     "Poco",
			"not_at_all": "Per nulla",
		},
	},
	{
		ID:    "mini_games_interest",
		Title: "Vorresti altri mini-giochi e attivazioni durante le partite?",
		Answers: map[string]string{
			"yes":   "Sì, assolutamente",
			"maybe": "Forse, a seconda della partita",
			"no":    "Non mi interessano",
		},
	},
}

const feedbackSuggestionTitle = "Suggerimenti lasciati dai tifosi"

var italianMonthNames = []string{
	"gennaio",
	"febbraio",
	"marzo",
	"aprile",
	"maggio",
	"giugno",
	"luglio",
	"agosto",
	"settembre",
	"ottobre",
	"novembre",
	"dicembre",
}

func buildEventHistoryPDF(entry eventHistoryEntry) ([]byte, error) {
	lines := assembleHistoryReportLines(entry)
	pages := layoutPDFLines(lines)
	pageCount := len(pages)
	if pageCount == 0 {
		pages = append(pages, []pdfLinePlacement{})
		pageCount = 1
	}

	pageObjStart := 3
	contentObjStart := pageObjStart + pageCount
	fontRegularID := contentObjStart + pageCount
	fontBoldID := fontRegularID + 1
	lastObjectID := fontBoldID

	pageIDs := make([]int, pageCount)
	contentIDs := make([]int, pageCount)
	for i := 0; i < pageCount; i++ {
		pageIDs[i] = pageObjStart + i
		contentIDs[i] = contentObjStart + i
	}

	objects := make([]pdfObject, 0, 2+pageCount*2+2)
	objects = append(objects,
		pdfObject{id: 1, data: []byte(fmt.Sprintf("<< /Type /Catalog /Pages %d 0 R >>", 2))},
		pdfObject{id: 2, data: buildPagesObjectData(pageIDs)},
	)

	for i, placements := range pages {
		pageObjID := pageIDs[i]
		contentObjID := contentIDs[i]
		contentData := renderPageContent(placements)

		objects = append(objects,
			pdfObject{id: pageObjID, data: buildPageObjectData(2, contentObjID, fontRegularID, fontBoldID)},
			pdfObject{id: contentObjID, data: buildContentObjectData(contentData)},
		)
	}

	objects = append(objects,
		pdfObject{id: fontRegularID, data: []byte("<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>")},
		pdfObject{id: fontBoldID, data: []byte("<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica-Bold >>")},
	)

	sort.Slice(objects, func(i, j int) bool {
		return objects[i].id < objects[j].id
	})

	offsets := make(map[int]int, lastObjectID)
	var buf bytes.Buffer
	buf.WriteString("%PDF-1.4\n%âãÏÓ\n")

	for _, obj := range objects {
		offsets[obj.id] = buf.Len()
		fmt.Fprintf(&buf, "%d 0 obj\n%s\nendobj\n", obj.id, obj.data)
	}

	startXref := buf.Len()
	fmt.Fprintf(&buf, "xref\n0 %d\n", lastObjectID+1)
	buf.WriteString("0000000000 65535 f \n")
	for id := 1; id <= lastObjectID; id++ {
		offset := offsets[id]
		fmt.Fprintf(&buf, "%010d 00000 n \n", offset)
	}

	fmt.Fprintf(&buf, "trailer << /Size %d /Root 1 0 R >>\n", lastObjectID+1)
	buf.WriteString("startxref\n")
	fmt.Fprintf(&buf, "%d\n", startXref)
	buf.WriteString("%%EOF\n")

	return buf.Bytes(), nil
}

func buildPagesObjectData(pageIDs []int) []byte {
	var builder strings.Builder
	builder.WriteString("<< /Type /Pages /Kids [")
	for _, id := range pageIDs {
		builder.WriteString(fmt.Sprintf(" %d 0 R", id))
	}
	builder.WriteString(" ] /Count ")
	builder.WriteString(strconv.Itoa(len(pageIDs)))
	builder.WriteString(" >>")
	return []byte(builder.String())
}

func buildPageObjectData(parentID, contentID, fontRegularID, fontBoldID int) []byte {
	return []byte(fmt.Sprintf("<< /Type /Page /Parent %d 0 R /MediaBox [0 0 %.0f %.0f] /Resources << /Font << /F1 %d 0 R /F2 %d 0 R >> >> /Contents %d 0 R >>",
		parentID, pdfPageWidth, pdfPageHeight, fontRegularID, fontBoldID, contentID))
}

func buildContentObjectData(content []byte) []byte {
	return []byte(fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(content), content))
}

func renderPageContent(lines []pdfLinePlacement) []byte {
	var buf bytes.Buffer
	for _, line := range lines {
		fontName := "F1"
		if line.Font == "bold" {
			fontName = "F2"
		}
		buf.WriteString("BT ")
		fmt.Fprintf(&buf, "/%s %.2f Tf 1 0 0 1 %.2f %.2f Tm (%s) Tj ET\n", fontName, line.Size, line.X, line.Y, escapePDFText(line.Text))
	}
	return buf.Bytes()
}

func assembleHistoryReportLines(entry eventHistoryEntry) []pdfLine {
	lines := make([]pdfLine, 0, 160)
	addParagraphLine(&lines, entry.Title, "bold", 18, 0, 10)
	addParagraphLine(&lines, fmt.Sprintf("Report generato il %s", formatItalianDateTime(time.Now())), "regular", 11, 0, 8)

	eventDate := formatHistoryDateForReport(entry.StartDateTime)
	addParagraphLine(&lines, fmt.Sprintf("Data evento: %s", eventDate), "regular", 11, 0, 4)

	location := strings.TrimSpace(entry.Location)
	if location != "" {
		addParagraphLine(&lines, fmt.Sprintf("Location: %s", location), "regular", 11, 0, 4)
	}

	teamNames := filterNonEmpty([]string{strings.TrimSpace(entry.HomeTeam), strings.TrimSpace(entry.AwayTeam)})
	if len(teamNames) > 0 {
		addParagraphLine(&lines, fmt.Sprintf("Squadre: %s", strings.Join(teamNames, " vs ")), "regular", 11, 0, 6)
	}

	addSectionTitle(&lines, "Numeri principali")

	summaryBullets := []string{
		fmt.Sprintf("Voti totali: %s", formatItalianNumber(entry.TotalVotes)),
		fmt.Sprintf("Click totali sugli sponsor: %s", formatItalianNumber(entry.SponsorClicksTotal)),
		fmt.Sprintf("Utenti totali nella sezione sponsor: %s", formatItalianNumber(entry.SponsorAnalytics.TotalUsers)),
		fmt.Sprintf("Utenti che hanno visto la sezione sponsor: %s", formatItalianNumber(entry.SponsorAnalytics.SeenUsers)),
	}

	if entry.MVP != nil {
		mvpName := strings.TrimSpace(strings.Join(filterNonEmpty([]string{entry.MVP.FirstName, entry.MVP.LastName}), " "))
		if mvpName == "" {
			if entry.MVP.PlayerID > 0 {
				mvpName = fmt.Sprintf("Giocatore #%d", entry.MVP.PlayerID)
			} else {
				mvpName = "Giocatore"
			}
		}
		summaryBullets = append(summaryBullets, fmt.Sprintf("MVP del pubblico: %s (%s voti)", mvpName, formatItalianNumber(entry.MVP.Votes)))
	} else {
		summaryBullets = append(summaryBullets, "MVP del pubblico: non assegnato")
	}

	if entry.FeedbackSummary != nil {
		summaryBullets = append(summaryBullets, fmt.Sprintf("Feedback raccolti: %s", formatItalianNumber(entry.FeedbackSummary.TotalResponses)))
	} else {
		summaryBullets = append(summaryBullets, "Feedback raccolti: nessun questionario compilato")
	}

	for _, bullet := range summaryBullets {
		addBulletLine(&lines, bullet, 11, 4)
	}

	addSectionTitle(&lines, "MVP del pubblico")
	if entry.MVP != nil {
		mvpName := strings.TrimSpace(strings.Join(filterNonEmpty([]string{entry.MVP.FirstName, entry.MVP.LastName}), " "))
		if mvpName == "" {
			if entry.MVP.PlayerID > 0 {
				mvpName = fmt.Sprintf("Giocatore #%d", entry.MVP.PlayerID)
			} else {
				mvpName = "Giocatore"
			}
		}
		addParagraphLine(&lines, fmt.Sprintf("%s ha raccolto %s voti.", mvpName, formatItalianNumber(entry.MVP.Votes)), "regular", 11, 0, 6)
	} else {
		addParagraphLine(&lines, "Nessun MVP è stato assegnato per questo evento.", "regular", 11, 0, 6)
	}

	addSectionTitle(&lines, "Andamento voti")
	if len(entry.Timeline) == 0 {
		addParagraphLine(&lines, "Non sono stati registrati voti durante l’evento.", "regular", 11, 0, 6)
	} else {
		for _, bucket := range entry.Timeline {
			label := bucket.Label
			if label == "" {
				label = formatTimelineBucket(bucket)
			}
			votesLabel := formatItalianNumber(bucket.Votes)
			addBulletLine(&lines, fmt.Sprintf("%s: %s voti nel periodo", label, votesLabel), 10.5, 3)
		}
	}

	addSectionTitle(&lines, "Interazioni sponsor")
	sponsorHighlights := []string{
		fmt.Sprintf("Utenti totali: %s", formatItalianNumber(entry.SponsorAnalytics.TotalUsers)),
		fmt.Sprintf("Utenti che hanno visto la sezione: %s", formatItalianNumber(entry.SponsorAnalytics.SeenUsers)),
		fmt.Sprintf("Utenti che hanno interagito con i contenuti: %s", formatItalianNumber(entry.SponsorAnalytics.WatchedUsers)),
	}

	if entry.SponsorAnalytics.AverageWatchTimeMs > 0 {
		sponsorHighlights = append(sponsorHighlights, fmt.Sprintf("Tempo medio di permanenza: %s", formatWatchDuration(entry.SponsorAnalytics.AverageWatchTimeMs)))
	}
	if entry.SponsorAnalytics.TotalWatchTimeMs > 0 {
		sponsorHighlights = append(sponsorHighlights, fmt.Sprintf("Tempo totale di visione: %s", formatWatchDuration(float64(entry.SponsorAnalytics.TotalWatchTimeMs))))
	}
	sponsorHighlights = append(sponsorHighlights, fmt.Sprintf("Click totali: %s", formatItalianNumber(entry.SponsorAnalytics.TotalClicks)))
	sponsorHighlights = append(sponsorHighlights, fmt.Sprintf("Utenti unici che hanno cliccato: %s", formatItalianNumber(entry.SponsorAnalytics.UniqueClickers)))
	if entry.SponsorAnalytics.TotalUsers > 0 {
		sponsorHighlights = append(sponsorHighlights, fmt.Sprintf("Tasso di visualizzazione: %s", formatPercentage(entry.SponsorAnalytics.SeenRate)))
		sponsorHighlights = append(sponsorHighlights, fmt.Sprintf("Tasso di click: %s", formatPercentage(entry.SponsorAnalytics.ClickRate)))
	}
	if entry.SponsorAnalytics.TopSponsor != nil && strings.TrimSpace(entry.SponsorAnalytics.TopSponsor.Name) != "" {
		sponsorHighlights = append(sponsorHighlights, fmt.Sprintf("Sponsor più coinvolgente: %s (%s visualizzazioni)", strings.TrimSpace(entry.SponsorAnalytics.TopSponsor.Name), formatItalianNumber(entry.SponsorAnalytics.TopSponsor.Views)))
	}

	for _, highlight := range sponsorHighlights {
		addBulletLine(&lines, highlight, 10.5, 3)
	}

	if len(entry.SponsorClicks) > 0 {
		addParagraphLine(&lines, "Dettaglio click per sponsor:", "bold", 11, 0, 3)
		for _, sponsor := range entry.SponsorClicks {
			name := strings.TrimSpace(sponsor.Name)
			if name == "" {
				name = "Sponsor"
			}
			addBulletLine(&lines, fmt.Sprintf("%s – %s click", name, formatItalianNumber(sponsor.Clicks)), 10, 3)
		}
	} else {
		addParagraphLine(&lines, "Nessun click registrato sugli sponsor durante l’evento.", "regular", 11, 0, 4)
	}

	if len(entry.SponsorAnalytics.Timeline) > 0 {
		addParagraphLine(&lines, "Andamento delle interazioni:", "bold", 11, 0, 3)
		for _, point := range entry.SponsorAnalytics.Timeline {
			label := formatSponsorTimelineLabel(point.Timestamp)
			if label == "" {
				label = "Intervallo"
			}
			highlight := fmt.Sprintf("%s: %s viste • %s viste complete • %s click",
				label,
				formatItalianNumber(point.Seen),
				formatItalianNumber(point.Watched),
				formatItalianNumber(point.Clicks),
			)
			addBulletLine(&lines, highlight, 10, 3)
		}
	}

	addSectionTitle(&lines, "Premi e lotteria")
	if len(entry.Prizes) == 0 {
		addParagraphLine(&lines, "Non sono stati registrati premi per questo evento.", "regular", 11, 0, 4)
	} else {
		for _, prize := range entry.Prizes {
			positionLabel := "Premio"
			if prize.Position > 0 {
				positionLabel = fmt.Sprintf("%dº premio", prize.Position)
			}
			winner := ""
			if strings.TrimSpace(prize.WinnerTicketCode) != "" {
				winner = fmt.Sprintf(" – Vincitore biglietto %s", strings.TrimSpace(prize.WinnerTicketCode))
			}
			addBulletLine(&lines, fmt.Sprintf("%s: %s%s", positionLabel, prize.Name, winner), 10, 3)
		}
		if !entry.HasPrizeDraw {
			addParagraphLine(&lines, "La lotteria non è stata ancora estratta.", "regular", 10, 0, 4)
		}
	}

	addSectionTitle(&lines, "Feedback dei tifosi")
	if entry.FeedbackSummary == nil || entry.FeedbackSummary.TotalResponses == 0 {
		addParagraphLine(&lines, "Non sono stati raccolti feedback dai partecipanti.", "regular", 11, 0, 4)
	} else {
		addParagraphLine(&lines, fmt.Sprintf("Risposte totali: %s", formatItalianNumber(entry.FeedbackSummary.TotalResponses)), "regular", 11, 0, 4)

		for _, question := range feedbackQuestionDefinitions {
			addParagraphLine(&lines, question.Title, "bold", 12, 0, 3)
			counts := feedbackCounts(entry.FeedbackSummary, question.ID)
			for key, label := range question.Answers {
				count := counts[key]
				parts := []string{label, fmt.Sprintf("%s risposte", formatItalianNumber(count))}
				if entry.FeedbackSummary.TotalResponses > 0 {
					percentage := 0.0
					if entry.FeedbackSummary.TotalResponses > 0 {
						percentage = float64(count) / float64(entry.FeedbackSummary.TotalResponses) * 100
					}
					parts = append(parts, formatPercentage(percentage))
				}
				addBulletLine(&lines, strings.Join(parts, " • "), 10, 2)
			}
		}

		addParagraphLine(&lines, feedbackSuggestionTitle, "bold", 12, 0, 3)
		if len(entry.FeedbackSummary.Suggestions) == 0 {
			addParagraphLine(&lines, "Nessun suggerimento inserito dai tifosi.", "regular", 10, 0, 4)
		} else {
			for _, suggestion := range entry.FeedbackSummary.Suggestions {
				addBulletLine(&lines, suggestion, 10, 2)
			}
		}
	}

	addParagraphLine(&lines, "Grazie per aver coinvolto i tifosi con WMVP Voting System.", "regular", 10, 0, 6)

	return lines
}

func layoutPDFLines(lines []pdfLine) [][]pdfLinePlacement {
	pages := make([][]pdfLinePlacement, 0, 1)
	cursorY := pdfPageHeight - pdfTopMargin
	current := make([]pdfLinePlacement, 0, len(lines))

	for _, line := range lines {
		lineHeight := line.Size * 1.2
		if cursorY-lineHeight < pdfBottomMargin {
			pages = append(pages, current)
			current = make([]pdfLinePlacement, 0)
			cursorY = pdfPageHeight - pdfTopMargin
		}

		cursorY -= lineHeight

		placement := pdfLinePlacement{
			Text: line.Text,
			Font: line.Font,
			Size: line.Size,
			X:    pdfLeftMargin + line.Indent,
			Y:    cursorY,
		}
		current = append(current, placement)

		cursorY -= line.Spacing
	}

	if len(current) > 0 {
		pages = append(pages, current)
	}

	return pages
}

func addParagraphLine(lines *[]pdfLine, text, font string, size, indent, spacing float64) {
	wrapped := wrapTextSegments(text, size, indent)
	if len(wrapped) == 0 {
		wrapped = []string{""}
	}
	for i, segment := range wrapped {
		gap := 2.0
		if i == len(wrapped)-1 {
			gap = spacing
		}
		*lines = append(*lines, pdfLine{
			Text:    segment,
			Font:    font,
			Size:    size,
			Indent:  indent,
			Spacing: gap,
		})
	}
}

func addSectionTitle(lines *[]pdfLine, title string) {
	addParagraphLine(lines, title, "bold", 14, 0, 6)
}

func addBulletLine(lines *[]pdfLine, text string, size, spacing float64) {
	wrapped := wrapTextSegments(strings.TrimSpace(text), size, 12)
	if len(wrapped) == 0 {
		wrapped = []string{""}
	}
	for i, segment := range wrapped {
		prefix := "• "
		if i > 0 {
			prefix = "  "
		}
		gap := 2.0
		if i == len(wrapped)-1 {
			gap = spacing
		}
		*lines = append(*lines, pdfLine{
			Text:    prefix + segment,
			Font:    "regular",
			Size:    size,
			Indent:  0,
			Spacing: gap,
		})
	}
}

func wrapTextSegments(text string, size, indent float64) []string {
	usableWidth := pdfPageWidth - pdfLeftMargin - pdfRightMargin - indent
	if usableWidth <= 0 {
		return []string{text}
	}

	// Estimate characters per line based on font size.
	maxChars := int(usableWidth / (size * 0.55))
	if maxChars < 16 {
		maxChars = 16
	}

	segments := make([]string, 0)
	paragraphs := strings.Split(text, "\n")
	for _, paragraph := range paragraphs {
		words := strings.Fields(paragraph)
		if len(words) == 0 {
			segments = append(segments, "")
			continue
		}
		var builder strings.Builder
		for _, word := range words {
			if builder.Len() == 0 {
				builder.WriteString(word)
				continue
			}
			if builder.Len()+1+len(word) <= maxChars {
				builder.WriteByte(' ')
				builder.WriteString(word)
			} else {
				segments = append(segments, builder.String())
				builder.Reset()
				builder.WriteString(word)
			}
		}
		if builder.Len() > 0 {
			segments = append(segments, builder.String())
		}
	}
	return segments
}

func escapePDFText(value string) string {
	replaced := strings.ReplaceAll(value, "\\", "\\\\")
	replaced = strings.ReplaceAll(replaced, "(", "\\(")
	replaced = strings.ReplaceAll(replaced, ")", "\\)")
	return replaced
}

func formatHistoryDateForReport(value string) string {
	if strings.TrimSpace(value) == "" {
		return "Data non disponibile"
	}

	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return formatItalianDateTime(parsed)
	}

	if parsed, _ := parseEventStart(value); !parsed.IsZero() {
		return formatItalianDateTime(parsed)
	}

	return strings.TrimSpace(value)
}

func formatItalianDateTime(ts time.Time) string {
	localized := ts.In(time.Local)
	monthIndex := int(localized.Month()) - 1
	monthName := ""
	if monthIndex >= 0 && monthIndex < len(italianMonthNames) {
		monthName = italianMonthNames[monthIndex]
	}
	return fmt.Sprintf("%d %s %d, %02d:%02d", localized.Day(), monthName, localized.Year(), localized.Hour(), localized.Minute())
}

func formatItalianShortDateTime(ts time.Time) string {
	localized := ts.In(time.Local)
	return localized.Format("02/01/2006 15:04")
}

func formatItalianNumber(value int) string {
	negative := value < 0
	if negative {
		value = -value
	}
	digits := strconv.Itoa(value)
	var builder strings.Builder
	if negative {
		builder.WriteByte('-')
	}
	for i, r := range digits {
		if i > 0 && (len(digits)-i)%3 == 0 {
			builder.WriteByte('.')
		}
		builder.WriteRune(r)
	}
	return builder.String()
}

func formatPercentage(value float64) string {
	if value <= 0 {
		return "0%"
	}
	rounded := math.Round(value*10) / 10
	if math.Abs(rounded-math.Round(rounded)) < 0.05 {
		return fmt.Sprintf("%.0f%%", math.Round(rounded))
	}
	return fmt.Sprintf("%.1f%%", rounded)
}

func formatWatchDuration(ms float64) string {
	if ms <= 0 {
		return "0s"
	}
	duration := time.Duration(math.Round(ms)) * time.Millisecond
	hours := duration / time.Hour
	minutes := (duration % time.Hour) / time.Minute
	seconds := (duration % time.Minute) / time.Second

	parts := make([]string, 0, 3)
	if hours > 0 {
		parts = append(parts, fmt.Sprintf("%dh", hours))
	}
	if minutes > 0 {
		parts = append(parts, fmt.Sprintf("%dm", minutes))
	}
	if seconds > 0 && hours == 0 {
		parts = append(parts, fmt.Sprintf("%ds", seconds))
	}
	if len(parts) == 0 {
		parts = append(parts, fmt.Sprintf("%ds", seconds))
	}
	return strings.Join(parts, " ")
}

func formatTimelineBucket(bucket historyTimelineBucket) string {
	start := parseTimeString(bucket.Start)
	end := parseTimeString(bucket.End)

	switch {
	case !start.IsZero() && !end.IsZero():
		if start.Year() == end.Year() && start.YearDay() == end.YearDay() {
			return fmt.Sprintf("%s-%s (%s)", start.In(time.Local).Format("15:04"), end.In(time.Local).Format("15:04"), formatItalianShortDateTime(start))
		}
		return fmt.Sprintf("%s - %s", formatItalianShortDateTime(start), formatItalianShortDateTime(end))
	case !start.IsZero():
		return formatItalianShortDateTime(start)
	case !end.IsZero():
		return formatItalianShortDateTime(end)
	default:
		if strings.TrimSpace(bucket.Label) != "" {
			return strings.TrimSpace(bucket.Label)
		}
	}
	return "Intervallo"
}

func formatSponsorTimelineLabel(value string) string {
	parsed := parseTimeString(value)
	if parsed.IsZero() {
		return strings.TrimSpace(value)
	}
	return formatItalianShortDateTime(parsed)
}

func parseTimeString(value string) time.Time {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return time.Time{}
	}
	if ts, err := time.Parse(time.RFC3339, trimmed); err == nil {
		return ts
	}
	if ts, _ := parseEventStart(trimmed); !ts.IsZero() {
		return ts
	}
	return time.Time{}
}

func feedbackCounts(summary *eventFeedbackSummaryResponse, questionID string) map[string]int {
	if summary == nil {
		return map[string]int{}
	}
	switch questionID {
	case "experience":
		return summary.Experience
	case "team_spirit":
		return summary.TeamSpirit
	case "perks_interest":
		return summary.PerksInterest
	case "mini_games_interest":
		return summary.MiniGamesInterest
	default:
		return map[string]int{}
	}
}

func filterNonEmpty(values []string) []string {
	filtered := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			filtered = append(filtered, trimmed)
		}
	}
	return filtered
}

func buildHistoryReportFilename(entry eventHistoryEntry) string {
	sanitized := sanitizeFilenameComponent(entry.Title)
	if sanitized == "" {
		if entry.ID > 0 {
			sanitized = fmt.Sprintf("evento-%d", entry.ID)
		} else {
			sanitized = "evento-storico"
		}
	}

	datePart := ""
	if ts, err := time.Parse(time.RFC3339, entry.StartDateTime); err == nil {
		datePart = ts.In(time.Local).Format("20060102")
	}

	parts := make([]string, 0, 3)
	if datePart != "" {
		parts = append(parts, datePart)
	}
	parts = append(parts, sanitized, "report")

	return strings.Join(parts, "_") + ".pdf"
}

func sanitizeFilenameComponent(value string) string {
	lower := strings.ToLower(strings.TrimSpace(value))
	var builder strings.Builder
	previousHyphen := false
	for _, r := range lower {
		switch {
		case r >= 'a' && r <= 'z':
			builder.WriteRune(r)
			previousHyphen = false
		case r >= '0' && r <= '9':
			builder.WriteRune(r)
			previousHyphen = false
		case r == ' ' || r == '-' || r == '_':
			if !previousHyphen && builder.Len() > 0 {
				builder.WriteByte('-')
				previousHyphen = true
			}
		default:
			if unicode.IsLetter(r) || unicode.IsDigit(r) {
				// keep Unicode letters/digits if ASCII equivalent is unavailable
				builder.WriteRune(r)
				previousHyphen = false
			}
		}
	}
	sanitized := strings.Trim(builder.String(), "-")
	return sanitized
}
