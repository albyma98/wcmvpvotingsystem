/*
Package database is the middleware between the app database and the code. All data (de)serialization (save/load) from a
persistent database are handled here. Database specific logic should never escape this package.

To use this package you need to apply migrations to the database if needed/wanted, connect to it (using the database
data source name from config), and then initialize an instance of AppDatabase from the DB connection.

For example, this code adds a parameter in `webapi` executable for the database data source name (add it to the
main.WebAPIConfiguration structure):

	DB struct {
		Filename string `conf:""`
	}

This is an example on how to migrate the DB and connect to it:

	// Start Database
	logger.Println("initializing database support")
	db, err := sql.Open("sqlite3", "./foo.db")
	if err != nil {
		logger.WithError(err).Error("error opening SQLite DB")
		return fmt.Errorf("opening SQLite: %w", err)
	}
	defer func() {
		logger.Debug("database stopping")
		_ = db.Close()
	}()

Then you can initialize the AppDatabase and pass it to the api package.
*/
package database

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// AppDatabase is the high level interface for the DB
type Team struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	Championship string `json:"championship"`
}

type Player struct {
	ID             int    `json:"id"`
	FirstName      string `json:"first_name"`
	LastName       string `json:"last_name"`
	Role           string `json:"role"`
	JerseyNumber   int    `json:"jersey_number"`
	ImageURL       string `json:"image_url"`
	IsCalledUp     bool   `json:"is_called_up"`
	TeamID         int    `json:"team_id"`
	OrganizationID int    `json:"organization_id"`
}

type Event struct {
	ID                        int                        `json:"id"`
	OrganizationID            int                        `json:"organization_id"`
	OrganizationName          string                     `json:"organization_name,omitempty"`
	OrganizationLogoURL       string                     `json:"organization_logo_url,omitempty"`
	Team1ID                   int                        `json:"team1_id"`
	Team2ID                   int                        `json:"team2_id"`
	StartDateTime             string                     `json:"start_datetime"`
	Location                  string                     `json:"location"`
	IsActive                  bool                       `json:"is_active"`
	VotesClosed               bool                       `json:"votes_closed"`
	IsConcluded               bool                       `json:"is_concluded"`
	ShowReactionTest          bool                       `json:"show_reaction_test"`
	ShowSelfie                bool                       `json:"show_selfie"`
	ShowVoteTrend             bool                       `json:"show_vote_trend"`
	ShowFeedbackSurvey        bool                       `json:"show_feedback_survey"`
	ShowPreVoteSponsors       bool                       `json:"show_pre_vote_sponsors"`
	ShowPreVoteBottomSponsors bool                       `json:"show_pre_vote_bottom_sponsors"`
	ShowVoteCounter           bool                       `json:"show_vote_counter"`
	Team1Name                 string                     `json:"team1_name,omitempty"`
	Team2Name                 string                     `json:"team2_name,omitempty"`
	Prizes                    []EventPrize               `json:"prizes,omitempty"`
	FeedbackSurvey            *EventFeedbackSurveyConfig `json:"feedback_survey,omitempty"`
}

type EventFeedbackSurveyConfig struct {
	Questions        []EventFeedbackQuestionConfig `json:"questions"`
	SuggestionPrompt string                        `json:"suggestion_prompt,omitempty"`
}

type EventFeedbackQuestionConfig struct {
	ID      string                      `json:"id"`
	Title   string                      `json:"title"`
	Answers []EventFeedbackAnswerConfig `json:"answers"`
}

type EventFeedbackAnswerConfig struct {
	Value string `json:"value"`
	Label string `json:"label"`
	Icon  string `json:"icon,omitempty"`
}

type Organization struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	Slug         string `json:"slug"`
	City         string `json:"city,omitempty"`
	LogoURL      string `json:"logo_url,omitempty"`
	IsActive     bool   `json:"is_active"`
	RosterSchema int    `json:"roster_schema,omitempty"`
	TeamID       int    `json:"team_id,omitempty"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

type OrganizationStats struct {
	OrganizationID int    `json:"organization_id"`
	TotalVotes     int    `json:"total_votes"`
	LastMatchVotes int    `json:"last_match_votes"`
	LastMatchDate  string `json:"last_match_date,omitempty"`
	TotalMatches   int    `json:"total_matches"`
}

type EventEngagementStats struct {
	EventID                int     `json:"event_id"`
	TotalDurationSeconds   int64   `json:"total_duration_seconds"`
	AverageDurationSeconds float64 `json:"average_duration_seconds"`
	TotalUsers             int     `json:"total_users"`
	VoteTrendOpens         int     `json:"vote_trend_opens"`
	SelfieOpens            int     `json:"selfie_opens"`
	SelfieAbandons         int     `json:"selfie_abandons"`
	ReactionOpens          int     `json:"reaction_opens"`
	ReactionAbandons       int     `json:"reaction_abandons"`
	ExperienceOpens        int     `json:"experience_opens"`
	ExperienceAbandons     int     `json:"experience_abandons"`
	PhotoEditOpens         int     `json:"photo_edit_opens"`
	VoteEditOpens          int     `json:"vote_edit_opens"`
	VoteEditAbandons       int     `json:"vote_edit_abandons"`
	VoteEditCompletions    int     `json:"vote_edit_completions"`
}

type OrganizationEngagementStat struct {
	OrganizationID          int     `json:"organization_id"`
	Name                    string  `json:"name"`
	Slug                    string  `json:"slug"`
	TotalDurationSeconds    int64   `json:"total_duration_seconds"`
	AverageDurationPerMatch float64 `json:"average_duration_per_match"`
	AverageDurationPerUser  float64 `json:"average_duration_per_user"`
}

type MasterEngagementSummary struct {
	TotalDurationSeconds    int64                        `json:"total_duration_seconds"`
	AverageDurationPerMatch float64                      `json:"average_duration_per_match"`
	AverageDurationPerUser  float64                      `json:"average_duration_per_user"`
	Organizations           []OrganizationEngagementStat `json:"organizations"`
}

type MasterDashboardSummary struct {
	TotalOrganizations int `json:"total_organizations"`
	TotalVotes         int `json:"total_votes"`
	VotesLast7Days     int `json:"votes_last_7_days"`
	TotalEvents        int `json:"total_events"`
}

type OrganizationLeaderboardEntry struct {
	OrganizationID   int     `json:"organization_id"`
	Name             string  `json:"name"`
	Slug             string  `json:"slug"`
	City             string  `json:"city"`
	LogoURL          string  `json:"logo_url"`
	TotalVotes       int     `json:"total_votes"`
	VotesLast7Days   int     `json:"votes_last_7_days"`
	TotalEvents      int     `json:"total_events"`
	GrowthPercentage float64 `json:"growth_percentage"`
}

type VoteTrendPoint struct {
	Date  string `json:"date"`
	Votes int    `json:"votes"`
}

type OrganizationVoteTrend struct {
	OrganizationID int              `json:"organization_id"`
	Name           string           `json:"name"`
	Slug           string           `json:"slug"`
	Data           []VoteTrendPoint `json:"data"`
}

type VoteTrendAnalytics struct {
	Global          []VoteTrendPoint        `json:"global"`
	PerOrganization []OrganizationVoteTrend `json:"per_organization"`
}

type TopEventEntry struct {
	EventID          int    `json:"event_id"`
	OrganizationID   int    `json:"organization_id"`
	OrganizationName string `json:"organization_name"`
	OrganizationSlug string `json:"organization_slug"`
	Label            string `json:"label"`
	StartDate        string `json:"start_date"`
	TotalVotes       int    `json:"total_votes"`
}

type TopEventsAnalytics struct {
	AllTime   []TopEventEntry `json:"all_time"`
	Last7Days []TopEventEntry `json:"last_7_days"`
}

type SponsorOrganizationStat struct {
	OrganizationID int     `json:"organization_id"`
	Name           string  `json:"name"`
	Slug           string  `json:"slug"`
	Impressions    int     `json:"impressions"`
	Clicks         int     `json:"clicks"`
	CTR            float64 `json:"ctr"`
}

type SponsorMasterStats struct {
	TotalImpressions int                       `json:"total_impressions"`
	TotalClicks      int                       `json:"total_clicks"`
	AverageCTR       float64                   `json:"average_ctr"`
	Organizations    []SponsorOrganizationStat `json:"organizations"`
}

type MonthlyMetrics struct {
	Month       string `json:"month"`
	Votes       int    `json:"votes"`
	Events      int    `json:"events"`
	UniqueUsers int    `json:"unique_users"`
}

type MetricDelta struct {
	Absolute int     `json:"absolute"`
	Percent  float64 `json:"percent"`
}

type MonthlyComparison struct {
	Current           MonthlyMetrics `json:"current"`
	Previous          MonthlyMetrics `json:"previous"`
	VotesChange       MetricDelta    `json:"votes_change"`
	EventsChange      MetricDelta    `json:"events_change"`
	UniqueUsersChange MetricDelta    `json:"unique_users_change"`
}

type MasterAnalytics struct {
	OrganizationLeaderboard []OrganizationLeaderboardEntry `json:"organization_leaderboard"`
	VoteTrends              VoteTrendAnalytics             `json:"vote_trends"`
	TopEvents               TopEventsAnalytics             `json:"top_events"`
	SponsorStats            SponsorMasterStats             `json:"sponsor_stats"`
	MonthlySummary          MonthlyComparison              `json:"monthly_summary"`
	Engagement              MasterEngagementSummary        `json:"engagement"`
}

var slugSanitizer = regexp.MustCompile(`[^a-z0-9]+`)

func DefaultEventFeedbackSurveyConfig() EventFeedbackSurveyConfig {
	return EventFeedbackSurveyConfig{
		Questions: []EventFeedbackQuestionConfig{
			{
				ID:    "experience",
				Title: "Com’è stata la tua esperienza di voto oggi?",
				Answers: []EventFeedbackAnswerConfig{
					{Value: "very_easy", Label: "Facilissima", Icon: "🤩"},
					{Value: "easy", Label: "Abbastanza semplice", Icon: "🙂"},
					{Value: "complex", Label: "Un po’ macchinosa", Icon: "😐"},
					{Value: "hard", Label: "Difficile", Icon: "😣"},
				},
			},
			{
				ID:    "team_spirit",
				Title: "Ti sei sentito parte della squadra mentre sceglievi l’MVP del pubblico?",
				Answers: []EventFeedbackAnswerConfig{
					{Value: "high", Label: "Sì, tantissimo!", Icon: "🔥"},
					{Value: "medium", Label: "In parte", Icon: "🙂"},
					{Value: "low", Label: "Non proprio", Icon: "🙄"},
				},
			},
			{
				ID:    "perks_interest",
				Title: "Immagina che la tua partecipazione ti permetta di vivere esperienze speciali o vantaggi come vero tifoso… ti piacerebbe?",
				Answers: []EventFeedbackAnswerConfig{
					{Value: "yes", Label: "Sì, assolutamente", Icon: "💙"},
					{Value: "maybe", Label: "Forse", Icon: "🙂"},
					{Value: "no", Label: "No", Icon: "🙄"},
				},
			},
			{
				ID:    "mini_games_interest",
				Title: "Ti piacerebbe divertirti ancora di più con mini-giochi o sfide tra un set e l’altro per mettere alla prova i tuoi riflessi?",
				Answers: []EventFeedbackAnswerConfig{
					{Value: "super_excited", Label: "Sì, carichissimo!", Icon: "🔥"},
					{Value: "maybe", Label: "Forse più avanti", Icon: "🙂"},
					{Value: "no", Label: "No grazie", Icon: "🙄"},
				},
			},
		},
		SuggestionPrompt: "Se potessi migliorare qualcosa, cosa ti piacerebbe aggiungere o cambiare?",
	}
}

func NormalizeEventFeedbackSurveyConfig(cfg *EventFeedbackSurveyConfig) EventFeedbackSurveyConfig {
	defaults := DefaultEventFeedbackSurveyConfig()
	if cfg == nil {
		return defaults
	}

	sanitized := EventFeedbackSurveyConfig{
		Questions:        make([]EventFeedbackQuestionConfig, len(defaults.Questions)),
		SuggestionPrompt: defaults.SuggestionPrompt,
	}

	questionOverrides := make(map[string]EventFeedbackQuestionConfig, len(cfg.Questions))
	for _, question := range cfg.Questions {
		if question.ID == "" {
			continue
		}
		questionOverrides[strings.TrimSpace(question.ID)] = question
	}

	for idx, base := range defaults.Questions {
		sanitizedQuestion := EventFeedbackQuestionConfig{
			ID:      base.ID,
			Title:   base.Title,
			Answers: make([]EventFeedbackAnswerConfig, len(base.Answers)),
		}

		override, ok := questionOverrides[base.ID]
		if ok {
			if trimmed := strings.TrimSpace(override.Title); trimmed != "" {
				sanitizedQuestion.Title = trimmed
			}
		}

		answerOverrides := make(map[string]EventFeedbackAnswerConfig)
		if ok {
			for _, answer := range override.Answers {
				if answer.Value == "" {
					continue
				}
				answerOverrides[strings.TrimSpace(answer.Value)] = answer
			}
		}

		for answerIdx, baseAnswer := range base.Answers {
			sanitizedAnswer := EventFeedbackAnswerConfig{
				Value: baseAnswer.Value,
				Label: baseAnswer.Label,
				Icon:  baseAnswer.Icon,
			}
			if overrideAnswer, found := answerOverrides[baseAnswer.Value]; found {
				if trimmed := strings.TrimSpace(overrideAnswer.Label); trimmed != "" {
					sanitizedAnswer.Label = trimmed
				}
				if trimmedIcon := strings.TrimSpace(overrideAnswer.Icon); trimmedIcon != "" {
					sanitizedAnswer.Icon = trimmedIcon
				}
			}
			sanitizedQuestion.Answers[answerIdx] = sanitizedAnswer
		}

		sanitized.Questions[idx] = sanitizedQuestion
	}

	if trimmed := strings.TrimSpace(cfg.SuggestionPrompt); trimmed != "" {
		sanitized.SuggestionPrompt = trimmed
	}

	return sanitized
}

func encodeEventFeedbackSurveyConfig(cfg EventFeedbackSurveyConfig) string {
	data, err := json.Marshal(cfg)
	if err != nil {
		return ""
	}
	return string(data)
}

func decodeEventFeedbackSurveyConfig(raw sql.NullString) EventFeedbackSurveyConfig {
	if raw.Valid {
		trimmed := strings.TrimSpace(raw.String)
		if trimmed != "" {
			var cfg EventFeedbackSurveyConfig
			if err := json.Unmarshal([]byte(trimmed), &cfg); err == nil {
				return NormalizeEventFeedbackSurveyConfig(&cfg)
			}
		}
	}
	return DefaultEventFeedbackSurveyConfig()
}

type EventFeedback struct {
	ID                int    `json:"id"`
	EventID           int    `json:"event_id"`
	Experience        string `json:"experience"`
	TeamSpirit        string `json:"team_spirit"`
	PerksInterest     string `json:"perks_interest"`
	MiniGamesInterest string `json:"mini_games_interest"`
	Suggestion        string `json:"suggestion"`
	CreatedAt         string `json:"created_at"`
}

type EventFeedbackSummary struct {
	TotalResponses          int
	ExperienceCounts        map[string]int
	TeamSpiritCounts        map[string]int
	PerksInterestCounts     map[string]int
	MiniGamesInterestCounts map[string]int
	Suggestions             []string
}

type EventPrize struct {
	ID       int               `json:"id"`
	EventID  int               `json:"event_id"`
	Name     string            `json:"name"`
	Position int               `json:"position"`
	Winner   *EventPrizeWinner `json:"winner,omitempty"`
}

type EventPrizeWinner struct {
	VoteID          int    `json:"vote_id"`
	TicketCode      string `json:"ticket_code"`
	PlayerID        int    `json:"player_id"`
	PlayerFirstName string `json:"player_first_name"`
	PlayerLastName  string `json:"player_last_name"`
	AssignedAt      string `json:"assigned_at"`
}

type Vote struct {
	ID              int    `json:"id"`
	EventID         int    `json:"event_id"`
	PlayerID        int    `json:"player_id"`
	TicketCode      string `json:"ticket_code"`
	TicketSignature string `json:"ticket_signature"`
	DeviceID        string `json:"device_id"`
	CreatedAt       string `json:"created_at"`
}

type EventVoteResult struct {
	PlayerID   int    `json:"player_id"`
	Votes      int    `json:"votes"`
	LastVoteAt string `json:"last_vote_at"`
}

type EventVoteLeaderboardEntry struct {
	PlayerID   int    `json:"player_id"`
	FirstName  string `json:"first_name"`
	LastName   string `json:"last_name"`
	ImageURL   string `json:"image_url"`
	Votes      int    `json:"votes"`
	LastVoteAt string `json:"last_vote_at"`
}

type EventTicket struct {
	VoteID          int    `json:"vote_id"`
	TicketCode      string `json:"ticket_code"`
	TicketSignature string `json:"ticket_signature"`
	PlayerID        int    `json:"player_id"`
	PlayerFirstName string `json:"player_first_name"`
	PlayerLastName  string `json:"player_last_name"`
	CreatedAt       string `json:"created_at"`
}

type TicketValidationPrize struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Position int    `json:"position"`
}

type TicketValidationResult struct {
	VoteID          int                    `json:"vote_id"`
	EventID         int                    `json:"event_id"`
	PlayerID        int                    `json:"player_id"`
	TicketCode      string                 `json:"ticket_code"`
	TicketSignature string                 `json:"ticket_signature"`
	PlayerFirstName string                 `json:"player_first_name"`
	PlayerLastName  string                 `json:"player_last_name"`
	CreatedAt       string                 `json:"created_at"`
	AssignedPrize   *TicketValidationPrize `json:"assigned_prize,omitempty"`
}

type Admin struct {
	ID             int    `json:"id"`
	Username       string `json:"username"`
	PasswordHash   string `json:"password_hash"`
	Role           string `json:"role"`
	OrganizationID int    `json:"organization_id"`
	CreatedAt      string `json:"created_at"`
}

type Sponsor struct {
	ID             int    `json:"id"`
	Position       int    `json:"position"`
	Name           string `json:"name"`
	ReportName     string `json:"report_name"`
	LogoData       string `json:"logo_data"`
	LinkURL        string `json:"link_url"`
	IsActive       bool   `json:"is_active"`
	OrganizationID int    `json:"organization_id"`
}

type SponsorClickStat struct {
	SponsorID  int    `json:"sponsor_id"`
	Name       string `json:"name"`
	ReportName string `json:"report_name"`
	LinkURL    string `json:"link_url"`
	Clicks     int    `json:"clicks"`
}

type SponsorViewStat struct {
	SponsorID  int    `json:"sponsor_id"`
	Name       string `json:"name"`
	ReportName string `json:"report_name"`
	Views      int    `json:"views"`
}

type SponsorTimelinePoint struct {
	Timestamp string `json:"timestamp"`
	Seen      int    `json:"seen"`
	Watched   int    `json:"watched"`
	Clicks    int    `json:"clicks"`
}

type SponsorAnalytics struct {
	TotalSessions    int                    `json:"total_sessions"`
	SeenSessions     int                    `json:"seen_sessions"`
	WatchedSessions  int                    `json:"watched_sessions"`
	TotalWatchTimeMs int64                  `json:"total_watch_time_ms"`
	AverageWatchTime float64                `json:"average_watch_time_ms"`
	TotalClicks      int                    `json:"total_clicks"`
	UniqueClickers   int                    `json:"unique_clickers"`
	TopSponsor       *SponsorViewStat       `json:"top_sponsor,omitempty"`
	Timeline         []SponsorTimelinePoint `json:"timeline"`
}

// Coupon represents a promotional offer connected to one or more matches and sponsors.
type Coupon struct {
	ID               int    `json:"id"`
	Title            string `json:"title"`
	ShortDesc        string `json:"short_desc"`
	SponsorID        int    `json:"sponsor_id"`
	MerchantID       int    `json:"merchant_id"`
	OrganizationID   int    `json:"organization_id"`
	MatchIDs         []int  `json:"match_ids"`
	StartDate        string `json:"start_date"`
	EndDate          string `json:"end_date"`
	MaxUses          int    `json:"max_uses"`
	Status           string `json:"status"`
	ImageURL         string `json:"image_url"`
	Highlight        bool   `json:"highlight"`
	Segmentation     string `json:"segmentation"`
	TotalViews       int    `json:"total_views"`
	TotalClaims      int    `json:"total_claims"`
	TotalRedemptions int    `json:"total_redemptions"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
}

// UserCoupon stores a generated code for a coupon assigned to a user or device.
type UserCoupon struct {
	ID              int     `json:"id"`
	CouponID        int     `json:"coupon_id"`
	UserID          *int    `json:"user_id,omitempty"`
	MatchID         int     `json:"match_id"`
	Code            string  `json:"code"`
	ClaimedAt       string  `json:"claimed_at"`
	UsedAt          *string `json:"used_at,omitempty"`
	UsedBySponsorID *int    `json:"used_by_sponsor_id,omitempty"`
	CreatedAt       string  `json:"created_at"`
	Coupon          *Coupon `json:"coupon,omitempty"`
}

type EventMVP struct {
	PlayerID   int    `json:"player_id"`
	FirstName  string `json:"first_name"`
	LastName   string `json:"last_name"`
	Votes      int    `json:"votes"`
	LastVoteAt string `json:"last_vote_at"`
}

type Selfie struct {
	ID                 int    `json:"id"`
	EventID            int    `json:"event_id"`
	DeviceID           string `json:"device_id"`
	Caption            string `json:"caption"`
	ImagePath          string `json:"image_path"`
	ImageURL           string `json:"image_url"`
	ContentType        string `json:"content_type"`
	Approved           bool   `json:"approved"`
	ShowOnScreen       bool   `json:"show_on_screen"`
	AcceptedImageTerms bool   `json:"accepted_image_terms"`
	CreatedAt          string `json:"created_at"`
}

type ReactionTestAttempt struct {
	ID             int       `json:"id"`
	EventID        int       `json:"event_id"`
	DeviceID       string    `json:"device_id"`
	ReactionTimeMs int       `json:"reaction_time_ms"`
	IsValid        bool      `json:"is_valid"`
	CreatedAt      time.Time `json:"created_at"`
}

type ReactionTestStats struct {
	Attempts int     `json:"attempts"`
	Average  float64 `json:"average_ms"`
}

type EventQuizConfig struct {
	EventID             int    `json:"event_id"`
	Enabled             bool   `json:"enabled"`
	QuestionsPerSession int    `json:"questions_per_session"`
	SecondsPerQuestion  int    `json:"seconds_per_question"`
	BaseReward          int    `json:"base_reward"`
	CompletionBonus     int    `json:"completion_bonus"`
	StreakBonus         int    `json:"streak_bonus"`
	ActiveFrom          string `json:"active_from,omitempty"`
	ActiveTo            string `json:"active_to,omitempty"`
}

type EventQuizQuestion struct {
	ID           int      `json:"id"`
	QuizID       int      `json:"quiz_id"`
	QuestionText string   `json:"question_text"`
	Answers      []string `json:"answers"`
	CorrectIndex int      `json:"correct_index"`
	OrderIndex   int      `json:"order_index"`
}

type EventStory struct {
	ID           int    `json:"id"`
	EventID      int    `json:"event_id"`
	PlayerName   string `json:"player_name"`
	ThumbnailURL string `json:"thumbnail_url"`
	VideoURL     string `json:"video_url"`
	Title        string `json:"title,omitempty"`
	IsActive     bool   `json:"is_active"`
	OrderIndex   int    `json:"order_index"`
}

type ContactSubmission struct {
	ID               int    `json:"id"`
	EventID          int    `json:"event_id"`
	DeviceID         string `json:"device_id"`
	ContactValue     string `json:"contact_value"`
	ContactType      string `json:"contact_type"`
	MarketingConsent bool   `json:"marketing_consent"`
	IsVerified       bool   `json:"is_verified"`
	BonusCode        string `json:"bonus_code,omitempty"`
	BonusSignature   string `json:"bonus_signature,omitempty"`
	CreatedAt        string `json:"created_at"`
}

type FanProfile struct {
	ID             int    `json:"id"`
	OrganizationID int    `json:"organization_id"`
	Nickname       string `json:"nickname"`
	Gender         string `json:"gender"`
	Phone          string `json:"phone"`
	AcceptedTerms  bool   `json:"accepted_terms"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}

type FanSession struct {
	Token    string `json:"token"`
	FanID    int    `json:"fan_id"`
	DeviceID string `json:"device_id"`
}

type FanRegisterInput struct {
	OrganizationID int
	EventID        int
	DeviceID       string
	SessionToken   string
	Nickname       string
	Gender         string
	Phone          string
	AcceptedTerms  bool
	GuestCoins     int
	EnterLottery   bool
}

type FanProfileSummary struct {
	Profile FanProfile `json:"profile"`
	Wallet  int        `json:"wallet"`
}

type FanRewardRedemption struct {
	ID        int    `json:"id"`
	EventID   int    `json:"event_id"`
	FanID     int    `json:"fan_id"`
	RewardKey string `json:"reward_key"`
	CostCoins int    `json:"cost_coins"`
	CreatedAt string `json:"created_at"`
}

type FanLeaderboardEntry struct {
	FanID     int    `json:"fan_id"`
	Nickname  string `json:"nickname"`
	Coins     int    `json:"coins"`
	Rank      int    `json:"rank"`
	CreatedAt string `json:"created_at"`
}

type ShopProduct struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	PriceCents  int    `json:"price_cents"`
	ImageURL    string `json:"image_url"`
	CreatedAt   string `json:"created_at"`
}

type ShopOrder struct {
	ID            int             `json:"id"`
	CustomerName  string          `json:"customer_name"`
	CustomerEmail string          `json:"customer_email"`
	CustomerNotes string          `json:"customer_notes"`
	TotalCents    int             `json:"total_cents"`
	CreatedAt     string          `json:"created_at"`
	Items         []ShopOrderItem `json:"items,omitempty"`
}

type ShopOrderItem struct {
	ID              int    `json:"id"`
	OrderID         int    `json:"order_id"`
	ProductID       int    `json:"product_id"`
	ProductName     string `json:"product_name"`
	Quantity        int    `json:"quantity"`
	UnitPriceCents  int    `json:"unit_price_cents"`
	ProductImageURL string `json:"product_image_url,omitempty"`
}

type QRRedirect struct {
	ID         int    `json:"id"`
	SourcePath string `json:"source_path"`
	TargetPath string `json:"target_path"`
	Hits       int    `json:"hits"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}

// AppDatabase is the high level interface for the DB
type AppDatabase interface {
	GetName() (string, error)
	SetName(name string) error
	AddVote(eventID, playerID int, code, signature, deviceID string) error
	CreateTeam(name, championship string) (int, error)
	ListTeams() ([]Team, error)
	UpdateTeam(id int, name, championship string) error
	DeleteTeam(id int) error
	CreatePlayer(p Player) (int, error)
	GetPlayerByID(id int) (Player, error)
	ListPlayers() ([]Player, error)
	ListPlayersByOrganization(organizationID int) ([]Player, error)
	UpdatePlayer(p Player) error
	DeletePlayer(id int) error
	UpdateOrganizationRosterSchema(organizationID int, rosterSchema int) error
	GetOrganizationRosterSchema(organizationID int) (int, error)
	CreateEvent(e Event) (int, error)
	ListEvents() ([]Event, error)
	ListEventsByOrganization(organizationID int) ([]Event, error)
	UpdateEvent(e Event) error
	DeleteEvent(id int) error
	SetActiveEvent(eventID int, organizationID int) error
	ClearActiveEvent(organizationID int) error
	CloseEventVoting(eventID int) error
	ConcludeEvent(eventID int) error
	GetActiveEvent(organizationID int) (Event, error)
	GetEventTeamIDs(eventID int) (int, int, error)
	GetEventOrganizationID(eventID int) (int, error)
	ListVotes() ([]Vote, error)
	GetVoteEventID(voteID int) (int, error)
	ListVotesByOrganization(organizationID int) ([]Vote, error)
	ListEventTickets(eventID int) ([]EventTicket, error)
	ValidateTicket(eventID int, code string) (TicketValidationResult, error)
	RedeemTicket(eventID int, code, signature string) (bool, error)
	ListEventPrizes(eventID int) ([]EventPrize, error)
	AssignPrizeWinner(eventID, prizeID, voteID int) (EventPrize, error)
	ClearPrizeWinner(eventID, prizeID int) error
	GetEventResults(eventID int) ([]EventVoteResult, error)
	GetEventVoteLeaderboard(eventID, limit int) ([]EventVoteLeaderboardEntry, error)
	GetEventVoteCount(eventID int) (int, error)
	ListEventVoteTimestamps(eventID int) ([]time.Time, error)
	GetEventMVP(eventID int) (EventMVP, error)
	DeleteVote(id int) error
	HasDeviceVoted(eventID int, deviceID string) (bool, error)
	GetDeviceVote(eventID int, deviceID string) (Vote, error)
	SaveSelfie(eventID int, deviceID, caption, imagePath, contentType string, acceptedImageTerms bool) (Selfie, error)
	UpdateSelfieURL(id int, imageURL string) error
	GetSelfieForDevice(eventID int, deviceID string) (Selfie, error)
	GetSelfieByID(id int) (Selfie, error)
	ListEventSelfies(eventID int) ([]Selfie, error)
	ListApprovedSelfies(eventID int) ([]Selfie, error)
	UpdateSelfieStatus(id int, approved bool, showOnScreen bool) error
	DeleteSelfie(id int) error
	RecordReactionTestAttempt(eventID int, deviceID string, reactionMs int) (ReactionTestAttempt, error)
	GetLatestReactionTestAttempt(eventID int, deviceID string) (ReactionTestAttempt, error)
	GetReactionTestStats(eventID int) (ReactionTestStats, error)
	GetEventQuizConfig(eventID int) (EventQuizConfig, error)
	UpsertEventQuizConfig(config EventQuizConfig) (EventQuizConfig, error)
	ListEventQuizQuestions(eventID int) ([]EventQuizQuestion, error)
	CreateEventQuizQuestion(eventID int, question EventQuizQuestion) (EventQuizQuestion, error)
	UpdateEventQuizQuestion(eventID int, questionID int, question EventQuizQuestion) (EventQuizQuestion, error)
	DeleteEventQuizQuestion(eventID int, questionID int) error
	GetEventQuizQuestion(eventID int, questionID int) (EventQuizQuestion, error)
	ListEventStories(eventID int, includeInactive bool) ([]EventStory, error)
	CreateEventStory(eventID int, story EventStory) (EventStory, error)
	UpdateEventStory(eventID int, storyID int, story EventStory) (EventStory, error)
	DeleteEventStory(eventID int, storyID int) error
	CreateAdmin(a Admin) (int, error)
	ListAdmins(organizationID int) ([]Admin, error)
	UpdateAdmin(a Admin) error
	DeleteAdmin(id int) error
	GetAdminByUsername(username string, organizationID int) (Admin, error)
	GetAdminByID(id int) (Admin, error)
	ListPartners(organizationID int) ([]Admin, error)
	CreateOrganization(org Organization) (Organization, error)
	UpdateOrganization(org Organization) (Organization, error)
	ListOrganizations() ([]Organization, error)
	GetOrganization(id int) (Organization, error)
	GetOrganizationBySlug(slug string) (Organization, error)
	GetOrganizationStats(id int) (OrganizationStats, error)
	GetMasterDashboardSummary() (MasterDashboardSummary, error)
	GetMasterAnalytics() (MasterAnalytics, error)
	CreateSponsor(s Sponsor) (int, error)
	UpdateSponsor(s Sponsor) error
	DeleteSponsor(id int, organizationID int) error
	ListSponsors(organizationID int) ([]Sponsor, error)
	ListActiveSponsors(organizationID int) ([]Sponsor, error)
	RecordSponsorSession(eventID int, deviceID string) error
	RecordSponsorExposure(eventID int, sponsorIDs []int, deviceID, exposureType string, durationMs int) error
	RecordSponsorClick(eventID, sponsorID int, deviceID string) error
	GetSponsorAnalytics(eventID int) (SponsorAnalytics, error)
	GetSponsorClickStats(eventID int) ([]SponsorClickStat, error)
	RecordEngagementSession(eventID int, deviceID string, durationSeconds int) error
	RecordPostVoteAction(eventID int, deviceID, action string) error
	GetEventEngagement(eventID int) (EventEngagementStats, error)
	GetMasterEngagement() (MasterEngagementSummary, error)
	RecordContactSubmission(contact ContactSubmission) (ContactSubmission, error)
	GetContactSubmission(eventID int, deviceID string) (ContactSubmission, error)
	ListContactBonuses(eventID int, deviceID string) ([]ContactSubmission, error)
	RecordContactEvent(eventID int, deviceID, name string) error
	RegisterFan(input FanRegisterInput) (FanProfileSummary, error)
	GetFanByPhoneE164(phone string) (FanProfile, error)
	CreateFanWithPhoneE164(phone string) (FanProfile, error)
	MarkFanPhoneVerified(phone string, verifiedAt time.Time) error
	UpsertFanSession(token string, fanID int, deviceID string) error
	GetFanBySessionToken(token string) (FanProfileSummary, error)
	GetFanByDevice(eventID int, organizationID int, deviceID string) (FanProfileSummary, error)
	GetGuestCoins(eventID int, organizationID int, deviceID string) (int, error)
	UpsertGuestCoins(eventID int, organizationID int, deviceID string, coins int) error
	GetFanLeaderboard(eventID int, organizationID int, limit int) ([]FanLeaderboardEntry, error)
	GetFanRank(eventID int, organizationID int, fanID int) (FanLeaderboardEntry, error)
	ListFanRewardRedemptions(eventID int, fanID int) ([]FanRewardRedemption, error)
	GetFanLotteryTicket(eventID int, fanID int) (EventTicket, error)
	RecordFanLotteryEntry(eventID int, fanID int, source string) error
	RecordFanRewardRedemption(eventID int, fanID int, rewardKey string, costCoins int) error
	PurgeEventData(eventID int) error
	RecordEventFeedback(feedback EventFeedback) error
	GetEventFeedbackSummary(eventID int) (EventFeedbackSummary, error)
	ListShopProducts() ([]ShopProduct, error)
	GetShopProduct(id int) (ShopProduct, error)
	CreateShopProduct(product ShopProduct) (ShopProduct, error)
	ListShopOrders() ([]ShopOrder, error)
	CreateShopOrder(order ShopOrder, items []ShopOrderItem) (ShopOrder, error)
	UpsertQRRedirect(sourcePath, targetPath string) (QRRedirect, error)
	ListQRRedirects() ([]QRRedirect, error)
	GetQRRedirectBySource(sourcePath string) (QRRedirect, error)
	IncrementQRRedirectHit(id int) error
	DeleteQRRedirect(id int) error
	// Coupons
	CreateCoupon(coupon Coupon) (Coupon, error)
	UpdateCoupon(coupon Coupon) (Coupon, error)
	DeleteCoupon(id int, organizationID int) error
	ListCoupons(organizationID int) ([]Coupon, error)
	GetCouponByID(id int) (Coupon, error)
	RecordCouponView(id int) error
	ClaimCoupon(couponID int, userID *int, matchID int) (UserCoupon, error)
	RedeemCoupon(code string, sponsorID int) (UserCoupon, error)
	ListUserCoupons(userID *int, sponsorID int) ([]UserCoupon, error)
	Ping() error
}

type appdbimpl struct {
	c *sql.DB
}

const maxSponsorSlots = 4
const couponCodeAttempts = 50

var (
	ErrMaxSponsors             = errors.New("maximum number of sponsors reached")
	ErrInvalidSponsorPos       = errors.New("invalid sponsor position")
	ErrInvalidSponsorData      = errors.New("invalid sponsor data")
	ErrPrizeAlreadyAssigned    = errors.New("prize already has a winner")
	ErrPrizeWinnerConflict     = errors.New("winner already assigned to another prize")
	ErrPrizeVoteMismatch       = errors.New("selected ticket is not valid for this event")
	ErrPrizeLockedByWinner     = errors.New("cannot remove a prize that already has a winner")
	ErrTicketSignatureMismatch = errors.New("ticket signature mismatch")
	ErrEventAlreadyConcluded   = errors.New("event already concluded")
	ErrInvalidOrganizationData = errors.New("invalid organization data")
	ErrInvalidContactData      = errors.New("invalid contact data")
	ErrCouponUnavailable       = errors.New("coupon not available")
	ErrCouponWrongSponsor      = errors.New("coupon sponsor mismatch")
	ErrCouponExpired           = errors.New("coupon expired")
	ErrCouponAlreadyUsed       = errors.New("coupon already used")
	ErrCouponMaxReached        = errors.New("coupon max uses reached")
)

var allowedPostVoteActions = map[string]struct{}{
	"vote_trend_open":    {},
	"selfie_open":        {},
	"selfie_abandon":     {},
	"reaction_open":      {},
	"reaction_abandon":   {},
	"experience_open":    {},
	"experience_abandon": {},
	"photo_edit_open":    {},
	"vote_edit_open":     {},
	"vote_edit_abandon":  {},
	"vote_edit_complete": {},
}

type rowScanner interface {
	Scan(dest ...interface{}) error
}

// New returns a new instance of AppDatabase based on the SQLite connection `db`.
// `db` is required - an error will be returned if `db` is `nil`.
func New(db *sql.DB) (AppDatabase, error) {
	if db == nil {
		return nil, errors.New("database is required when building a AppDatabase")
	}

	// Check if table exists. If not, the database is empty, and we need to create the structure
	var tableName string
	err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='example_table';`).Scan(&tableName)
	if errors.Is(err, sql.ErrNoRows) {
		sqlStmt := `CREATE TABLE example_table (id INTEGER NOT NULL PRIMARY KEY, name TEXT);`
		_, err = db.Exec(sqlStmt)
		if err != nil {
			return nil, fmt.Errorf("error creating database structure: %w", err)
		}
	}
	// Create teams table if not exists
	err = db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='teams';`).Scan(&tableName)
	if errors.Is(err, sql.ErrNoRows) {
		sqlStmt := `CREATE TABLE teams (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL, championship TEXT NOT NULL DEFAULT '');`
		_, err = db.Exec(sqlStmt)
		if err != nil {
			return nil, fmt.Errorf("error creating teams table: %w", err)
		}
	} else {
		if _, err = db.Exec(`ALTER TABLE teams ADD COLUMN championship TEXT NOT NULL DEFAULT ''`); err != nil {
			if !strings.Contains(err.Error(), "duplicate column name") {
				return nil, fmt.Errorf("error ensuring teams championship column: %w", err)
			}
		}
	}

	// Create players table if not exists
	err = db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='players';`).Scan(&tableName)
	if errors.Is(err, sql.ErrNoRows) {
		sqlStmt := `CREATE TABLE players (id INTEGER PRIMARY KEY AUTOINCREMENT, first_name TEXT NOT NULL, last_name TEXT NOT NULL, role TEXT NOT NULL, jersey_number INTEGER, image_url TEXT, is_called_up INTEGER NOT NULL DEFAULT 1, team_id INTEGER NOT NULL, organization_id INTEGER NOT NULL DEFAULT 0, FOREIGN KEY (team_id) REFERENCES teams(id), FOREIGN KEY (organization_id) REFERENCES organizations(id));`
		_, err = db.Exec(sqlStmt)
		if err != nil {
			return nil, fmt.Errorf("error creating players table: %w", err)
		}
	} else {
		// attempt schema update if column missing
		_, _ = db.Exec(`ALTER TABLE players ADD COLUMN image_url TEXT`)
		_, _ = db.Exec(`ALTER TABLE players ADD COLUMN is_called_up INTEGER NOT NULL DEFAULT 1`)
		_, _ = db.Exec(`ALTER TABLE players ADD COLUMN organization_id INTEGER NOT NULL DEFAULT 0`)
	}

	if _, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_players_org ON players(organization_id)`); err != nil {
		return nil, fmt.Errorf("error ensuring players organization index: %w", err)
	}

	// Create events table if not exists
	err = db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='events';`).Scan(&tableName)
	if errors.Is(err, sql.ErrNoRows) {
		sqlStmt := `CREATE TABLE events (id INTEGER PRIMARY KEY AUTOINCREMENT, organization_id INTEGER NOT NULL DEFAULT 0, team1_id INTEGER NOT NULL, team2_id INTEGER NOT NULL, start_datetime TEXT NOT NULL, location TEXT, FOREIGN KEY (organization_id) REFERENCES organizations(id), FOREIGN KEY (team1_id) REFERENCES teams(id), FOREIGN KEY (team2_id) REFERENCES teams(id));`
		_, err = db.Exec(sqlStmt)
		if err != nil {
			return nil, fmt.Errorf("error creating events table: %w", err)
		}

	} else {
		if _, err = db.Exec(`ALTER TABLE events ADD COLUMN organization_id INTEGER NOT NULL DEFAULT 0`); err != nil {
			if !strings.Contains(err.Error(), "duplicate column name") {
				return nil, fmt.Errorf("error ensuring events organization column: %w", err)
			}
		}

	}

	err = db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='event_prizes';`).Scan(&tableName)
	if errors.Is(err, sql.ErrNoRows) {
		sqlStmt := `CREATE TABLE event_prizes (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        event_id INTEGER NOT NULL,
        name TEXT NOT NULL,
        position INTEGER NOT NULL DEFAULT 1,
        winner_vote_id INTEGER,
        winner_assigned_at TEXT,
        FOREIGN KEY (event_id) REFERENCES events(id) ON DELETE CASCADE,
        FOREIGN KEY (winner_vote_id) REFERENCES votes(id)
);`
		_, err = db.Exec(sqlStmt)
		if err != nil {
			return nil, fmt.Errorf("error creating event_prizes table: %w", err)
		}
	}

	if _, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_event_prizes_event ON event_prizes(event_id)`); err != nil {
		return nil, fmt.Errorf("error ensuring event_prizes event index: %w", err)
	}

	if _, err = db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_event_prizes_event_position ON event_prizes(event_id, position)`); err != nil {
		return nil, fmt.Errorf("error ensuring event_prizes position index: %w", err)
	}

	if _, err = db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_event_prizes_winner_vote ON event_prizes(winner_vote_id)`); err != nil {
		return nil, fmt.Errorf("error ensuring event_prizes winner index: %w", err)
	}

	if _, err = db.Exec(`ALTER TABLE events ADD COLUMN is_active INTEGER NOT NULL DEFAULT 0`); err != nil {
		if !strings.Contains(err.Error(), "duplicate column name") {
			return nil, fmt.Errorf("error ensuring events active column: %w", err)
		}
	}

	if _, err = db.Exec(`ALTER TABLE events ADD COLUMN votes_closed INTEGER NOT NULL DEFAULT 0`); err != nil {
		if !strings.Contains(err.Error(), "duplicate column name") {
			return nil, fmt.Errorf("error ensuring events votes closed column: %w", err)
		}
	}

	if _, err = db.Exec(`ALTER TABLE events ADD COLUMN is_concluded INTEGER NOT NULL DEFAULT 0`); err != nil {
		if !strings.Contains(err.Error(), "duplicate column name") {
			return nil, fmt.Errorf("error ensuring events concluded column: %w", err)
		}
	}

	if _, err = db.Exec(`ALTER TABLE events ADD COLUMN show_reaction_test INTEGER NOT NULL DEFAULT 1`); err != nil {
		if !strings.Contains(err.Error(), "duplicate column name") {
			return nil, fmt.Errorf("error ensuring events reaction test column: %w", err)
		}
	}

	if _, err = db.Exec(`ALTER TABLE events ADD COLUMN show_selfie INTEGER NOT NULL DEFAULT 1`); err != nil {
		if !strings.Contains(err.Error(), "duplicate column name") {
			return nil, fmt.Errorf("error ensuring events selfie column: %w", err)
		}
	}

	if _, err = db.Exec(`ALTER TABLE events ADD COLUMN show_vote_trend INTEGER NOT NULL DEFAULT 1`); err != nil {
		if !strings.Contains(err.Error(), "duplicate column name") {
			return nil, fmt.Errorf("error ensuring events vote trend column: %w", err)
		}
	}

	if _, err = db.Exec(`ALTER TABLE events ADD COLUMN show_feedback_survey INTEGER NOT NULL DEFAULT 1`); err != nil {
		if !strings.Contains(err.Error(), "duplicate column name") {
			return nil, fmt.Errorf("error ensuring events feedback survey column: %w", err)
		}
	}

	if _, err = db.Exec(`ALTER TABLE events ADD COLUMN show_pre_vote_sponsors INTEGER NOT NULL DEFAULT 1`); err != nil {
		if !strings.Contains(err.Error(), "duplicate column name") {
			return nil, fmt.Errorf("error ensuring events pre vote sponsors column: %w", err)
		}
	}

	if _, err = db.Exec(`ALTER TABLE events ADD COLUMN show_pre_vote_bottom_sponsors INTEGER NOT NULL DEFAULT 1`); err != nil {
		if !strings.Contains(err.Error(), "duplicate column name") {
			return nil, fmt.Errorf("error ensuring events pre vote bottom sponsors column: %w", err)
		}
	}

	if _, err = db.Exec(`ALTER TABLE events ADD COLUMN show_vote_counter INTEGER NOT NULL DEFAULT 1`); err != nil {
		if !strings.Contains(err.Error(), "duplicate column name") {
			return nil, fmt.Errorf("error ensuring events vote counter column: %w", err)
		}
	}

	if _, err = db.Exec(`ALTER TABLE events ADD COLUMN feedback_survey_config TEXT`); err != nil {
		if !strings.Contains(err.Error(), "duplicate column name") {
			return nil, fmt.Errorf("error ensuring events feedback survey config column: %w", err)
		}
	}

	if _, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_events_org ON events(organization_id)`); err != nil {
		return nil, fmt.Errorf("error ensuring events organization index: %w", err)
	}

	// Create votes table if not exists
	err = db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='votes';`).Scan(&tableName)
	if errors.Is(err, sql.ErrNoRows) {
		sqlStmt := `CREATE TABLE votes (id INTEGER PRIMARY KEY AUTOINCREMENT, event_id INTEGER NOT NULL, player_id INTEGER NOT NULL, ticket_code TEXT NOT NULL, ticket_signature TEXT NOT NULL, device_id TEXT NOT NULL, created_at TEXT DEFAULT CURRENT_TIMESTAMP, FOREIGN KEY (event_id) REFERENCES events(id), FOREIGN KEY (player_id) REFERENCES players(id));`
		_, err = db.Exec(sqlStmt)
		if err != nil {
			return nil, fmt.Errorf("error creating votes table: %w", err)
		}
		_, err = db.Exec(`CREATE UNIQUE INDEX unique_vote_per_event_device ON votes (event_id, device_id);`)
		if err != nil {
			return nil, fmt.Errorf("error creating votes index: %w", err)
		}
	}

	// Create organizations table if not exists
	err = db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='organizations';`).Scan(&tableName)
	if errors.Is(err, sql.ErrNoRows) {
		sqlStmt := `CREATE TABLE organizations (
id INTEGER PRIMARY KEY AUTOINCREMENT,
name TEXT NOT NULL,
slug TEXT NOT NULL DEFAULT '',
city TEXT,
logo_url TEXT,
is_active INTEGER NOT NULL DEFAULT 1,
roster_schema INTEGER NOT NULL DEFAULT 13,
team_id INTEGER,
created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
FOREIGN KEY (team_id) REFERENCES teams(id)
);`
		_, err = db.Exec(sqlStmt)
		if err != nil {
			return nil, fmt.Errorf("error creating organizations table: %w", err)
		}
	}

	if _, err = db.Exec(`ALTER TABLE organizations ADD COLUMN slug TEXT NOT NULL DEFAULT ''`); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column name") {
			return nil, fmt.Errorf("error ensuring organizations slug column: %w", err)
		}
	}
	if _, err = db.Exec(`ALTER TABLE organizations ADD COLUMN roster_schema INTEGER NOT NULL DEFAULT 13`); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column name") {
			return nil, fmt.Errorf("error ensuring organizations roster schema column: %w", err)
		}
	}
	if _, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_organizations_team ON organizations(team_id)`); err != nil {
		return nil, fmt.Errorf("error ensuring organizations team index: %w", err)
	}
	if _, err = db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_organizations_slug ON organizations(slug) WHERE slug <> ''`); err != nil {
		return nil, fmt.Errorf("error ensuring organizations slug index: %w", err)
	}

	if _, err = db.Exec(`CREATE TRIGGER IF NOT EXISTS trg_organizations_updated_at AFTER UPDATE ON organizations BEGIN UPDATE organizations SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id; END;`); err != nil {
		return nil, fmt.Errorf("error ensuring organizations update trigger: %w", err)
	}

	if err = ensureAdminsTable(db); err != nil {
		return nil, err
	}
	_, err = db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS unique_vote_per_event_device ON votes (event_id, device_id);`)
	if err != nil {
		return nil, fmt.Errorf("error ensuring votes device index: %w", err)
	}
	_, err = db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS unique_vote_code_per_event ON votes (event_id, ticket_code);`)
	if err != nil {
		return nil, fmt.Errorf("error ensuring votes code index: %w", err)
	}
	if _, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_votes_event ON votes (event_id);`); err != nil {
		return nil, fmt.Errorf("error ensuring votes event index: %w", err)
	}

	err = db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='selfies';`).Scan(&tableName)
	if errors.Is(err, sql.ErrNoRows) {
		sqlStmt := `CREATE TABLE selfies (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        event_id INTEGER NOT NULL,
        device_id TEXT NOT NULL,
        caption TEXT,
        image_path TEXT NOT NULL,
        image_url TEXT,
        content_type TEXT,
        approved INTEGER NOT NULL DEFAULT 0,
        show_on_screen INTEGER NOT NULL DEFAULT 0,
        accepted_image_terms INTEGER NOT NULL DEFAULT 0,
        created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
        FOREIGN KEY (event_id) REFERENCES events(id) ON DELETE CASCADE,
        UNIQUE(event_id, device_id)
);`
		_, err = db.Exec(sqlStmt)
		if err != nil {
			return nil, fmt.Errorf("error creating selfies table: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("error verifying selfies table: %w", err)
	}

	if _, err = db.Exec(`ALTER TABLE selfies ADD COLUMN accepted_image_terms INTEGER NOT NULL DEFAULT 0`); err != nil {
		if !strings.Contains(err.Error(), "duplicate column name") {
			return nil, fmt.Errorf("error ensuring selfies accepted terms column: %w", err)
		}
	}

	if _, err = db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_selfies_event_device ON selfies(event_id, device_id);`); err != nil {
		return nil, fmt.Errorf("error ensuring selfies device index: %w", err)
	}
	if _, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_selfies_event_created ON selfies(event_id, created_at);`); err != nil {
		return nil, fmt.Errorf("error ensuring selfies created index: %w", err)
	}
	if _, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_selfies_approved ON selfies(event_id, approved);`); err != nil {
		return nil, fmt.Errorf("error ensuring selfies approval index: %w", err)
	}

	err = db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='reaction_tests';`).Scan(&tableName)
	if errors.Is(err, sql.ErrNoRows) {
		sqlStmt := `CREATE TABLE reaction_tests (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        event_id INTEGER NOT NULL,
        device_id TEXT NOT NULL,
        reaction_time_ms INTEGER NOT NULL,
        is_valid INTEGER NOT NULL DEFAULT 1,
        created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
        FOREIGN KEY (event_id) REFERENCES events(id) ON DELETE CASCADE
);`
		_, err = db.Exec(sqlStmt)
		if err != nil {
			return nil, fmt.Errorf("error creating reaction_tests table: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("error verifying reaction_tests table: %w", err)
	}

	if _, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_reaction_tests_event ON reaction_tests(event_id)`); err != nil {
		return nil, fmt.Errorf("error ensuring reaction_tests event index: %w", err)
	}

	if _, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_reaction_tests_device ON reaction_tests(event_id, device_id)`); err != nil {
		return nil, fmt.Errorf("error ensuring reaction_tests device index: %w", err)
	}

	err = db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='event_quiz';`).Scan(&tableName)
	if errors.Is(err, sql.ErrNoRows) {
		sqlStmt := `CREATE TABLE event_quiz (
        event_id INTEGER PRIMARY KEY,
        enabled INTEGER NOT NULL DEFAULT 0,
        questions_per_session INTEGER NOT NULL DEFAULT 5,
        seconds_per_question INTEGER NOT NULL DEFAULT 8,
        base_reward INTEGER NOT NULL DEFAULT 3,
        completion_bonus INTEGER NOT NULL DEFAULT 5,
        streak_bonus INTEGER NOT NULL DEFAULT 1,
        active_from TEXT,
        active_to TEXT,
        FOREIGN KEY (event_id) REFERENCES events(id) ON DELETE CASCADE
);`
		_, err = db.Exec(sqlStmt)
		if err != nil {
			return nil, fmt.Errorf("error creating event_quiz table: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("error verifying event_quiz table: %w", err)
	}

	err = db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='event_quiz_questions';`).Scan(&tableName)
	if errors.Is(err, sql.ErrNoRows) {
		sqlStmt := `CREATE TABLE event_quiz_questions (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        quiz_id INTEGER NOT NULL,
        question_text TEXT NOT NULL,
        answers_json TEXT NOT NULL,
        correct_index INTEGER NOT NULL,
        order_index INTEGER NOT NULL DEFAULT 0,
        FOREIGN KEY (quiz_id) REFERENCES event_quiz(event_id) ON DELETE CASCADE
);`
		_, err = db.Exec(sqlStmt)
		if err != nil {
			return nil, fmt.Errorf("error creating event_quiz_questions table: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("error verifying event_quiz_questions table: %w", err)
	}

	if _, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_event_quiz_questions_quiz ON event_quiz_questions(quiz_id, order_index)`); err != nil {
		return nil, fmt.Errorf("error ensuring event_quiz_questions quiz index: %w", err)
	}

	err = db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='event_stories';`).Scan(&tableName)
	if errors.Is(err, sql.ErrNoRows) {
		sqlStmt := `CREATE TABLE event_stories (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        event_id INTEGER NOT NULL,
        player_name TEXT NOT NULL,
        thumbnail_url TEXT NOT NULL,
        video_url TEXT NOT NULL,
        title TEXT,
        is_active INTEGER NOT NULL DEFAULT 1,
        order_index INTEGER NOT NULL DEFAULT 0,
        created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
        updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
        FOREIGN KEY (event_id) REFERENCES events(id) ON DELETE CASCADE
);`
		_, err = db.Exec(sqlStmt)
		if err != nil {
			return nil, fmt.Errorf("error creating event_stories table: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("error verifying event_stories table: %w", err)
	}

	if _, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_event_stories_event ON event_stories(event_id, order_index, id)`); err != nil {
		return nil, fmt.Errorf("error ensuring event_stories event index: %w", err)
	}

	err = db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='page_engagements';`).Scan(&tableName)
	if errors.Is(err, sql.ErrNoRows) {
		sqlStmt := `CREATE TABLE page_engagements (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        event_id INTEGER NOT NULL,
        device_id TEXT NOT NULL,
        duration_seconds INTEGER NOT NULL,
        created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
        FOREIGN KEY (event_id) REFERENCES events(id) ON DELETE CASCADE
);`
		_, err = db.Exec(sqlStmt)
		if err != nil {
			return nil, fmt.Errorf("error creating page_engagements table: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("error verifying page_engagements table: %w", err)
	}

	if _, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_page_engagements_event ON page_engagements(event_id)`); err != nil {
		return nil, fmt.Errorf("error ensuring page_engagements event index: %w", err)
	}
	if _, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_page_engagements_device ON page_engagements(event_id, device_id)`); err != nil {
		return nil, fmt.Errorf("error ensuring page_engagements device index: %w", err)
	}

	err = db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='post_vote_actions';`).Scan(&tableName)
	if errors.Is(err, sql.ErrNoRows) {
		sqlStmt := `CREATE TABLE post_vote_actions (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        event_id INTEGER NOT NULL,
        device_id TEXT NOT NULL,
        action TEXT NOT NULL,
        created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
        FOREIGN KEY (event_id) REFERENCES events(id) ON DELETE CASCADE
);`
		_, err = db.Exec(sqlStmt)
		if err != nil {
			return nil, fmt.Errorf("error creating post_vote_actions table: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("error verifying post_vote_actions table: %w", err)
	}

	if _, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_post_vote_actions_event ON post_vote_actions(event_id)`); err != nil {
		return nil, fmt.Errorf("error ensuring post_vote_actions event index: %w", err)
	}
	if _, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_post_vote_actions_action ON post_vote_actions(event_id, action)`); err != nil {
		return nil, fmt.Errorf("error ensuring post_vote_actions action index: %w", err)
	}

	err = db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='event_feedback';`).Scan(&tableName)
	if errors.Is(err, sql.ErrNoRows) {
		sqlStmt := `CREATE TABLE event_feedback (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        event_id INTEGER NOT NULL,
        experience TEXT NOT NULL,
        team_spirit TEXT NOT NULL,
        perks_interest TEXT NOT NULL,
        mini_games_interest TEXT NOT NULL,
        suggestion TEXT,
        created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
        FOREIGN KEY (event_id) REFERENCES events(id) ON DELETE CASCADE
);`
		_, err = db.Exec(sqlStmt)
		if err != nil {
			return nil, fmt.Errorf("error creating event_feedback table: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("error verifying event_feedback table: %w", err)
	}

	if _, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_event_feedback_event ON event_feedback(event_id)`); err != nil {
		return nil, fmt.Errorf("error ensuring event_feedback event index: %w", err)
	}

	err = db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='contacts';`).Scan(&tableName)
	if errors.Is(err, sql.ErrNoRows) {
		sqlStmt := `CREATE TABLE contacts (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        event_id INTEGER NOT NULL,
        device_id TEXT NOT NULL,
        contact_value TEXT NOT NULL,
        contact_type TEXT NOT NULL,
        marketing_consent INTEGER NOT NULL DEFAULT 0,
        is_verified INTEGER NOT NULL DEFAULT 0,
        bonus_code TEXT,
        bonus_signature TEXT,
        created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
        UNIQUE(event_id, device_id),
        FOREIGN KEY (event_id) REFERENCES events(id) ON DELETE CASCADE
);`
		_, err = db.Exec(sqlStmt)
		if err != nil {
			return nil, fmt.Errorf("error creating contacts table: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("error verifying contacts table: %w", err)
	}

	if _, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_contacts_event ON contacts(event_id)`); err != nil {
		return nil, fmt.Errorf("error ensuring contacts event index: %w", err)
	}

	err = db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='contact_events';`).Scan(&tableName)
	if errors.Is(err, sql.ErrNoRows) {
		sqlStmt := `CREATE TABLE contact_events (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        event_id INTEGER NOT NULL,
        device_id TEXT NOT NULL,
        event_name TEXT NOT NULL,
        created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
        FOREIGN KEY (event_id) REFERENCES events(id) ON DELETE CASCADE
);`
		_, err = db.Exec(sqlStmt)
		if err != nil {
			return nil, fmt.Errorf("error creating contact_events table: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("error verifying contact_events table: %w", err)
	}

	if _, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_contact_events_event ON contact_events(event_id)`); err != nil {
		return nil, fmt.Errorf("error ensuring contact_events event index: %w", err)
	}

	err = db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='tickets';`).Scan(&tableName)
	if errors.Is(err, sql.ErrNoRows) {
		sqlStmt := `CREATE TABLE tickets (
        event_id INTEGER NOT NULL,
        code TEXT NOT NULL,
        signature TEXT NOT NULL,
        redeemed_at TEXT,
        PRIMARY KEY (event_id, code)
);`
		_, err = db.Exec(sqlStmt)
		if err != nil {
			return nil, fmt.Errorf("error creating tickets table: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("error verifying tickets table: %w", err)
	}

	if _, err = db.Exec(`ALTER TABLE tickets ADD COLUMN redeemed_at TEXT`); err != nil {
		if !strings.Contains(err.Error(), "duplicate column name") {
			return nil, fmt.Errorf("error ensuring tickets redeemed_at column: %w", err)
		}
	}

	if _, err = db.Exec(`ALTER TABLE tickets ADD COLUMN signature TEXT`); err != nil {
		if !strings.Contains(err.Error(), "duplicate column name") {
			return nil, fmt.Errorf("error ensuring tickets signature column: %w", err)
		}
	}

	err = db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='sponsors';`).Scan(&tableName)
	if errors.Is(err, sql.ErrNoRows) {
		sqlStmt := `CREATE TABLE sponsors (id INTEGER PRIMARY KEY AUTOINCREMENT, organization_id INTEGER NOT NULL DEFAULT 0, position INTEGER NOT NULL, name TEXT NOT NULL, report_name TEXT, logo_data TEXT NOT NULL, link_url TEXT, is_active INTEGER NOT NULL DEFAULT 1, CHECK(position BETWEEN 1 AND ` + fmt.Sprint(maxSponsorSlots) + `), FOREIGN KEY (organization_id) REFERENCES organizations(id), UNIQUE(organization_id, position));`
		_, err = db.Exec(sqlStmt)
		if err != nil {
			return nil, fmt.Errorf("error creating sponsors table: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("error verifying sponsors table: %w", err)
	} else {
		_, _ = db.Exec(`ALTER TABLE sponsors ADD COLUMN organization_id INTEGER NOT NULL DEFAULT 0`)
		if _, err = db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_sponsors_org_position ON sponsors(organization_id, position)`); err != nil {
			return nil, fmt.Errorf("error ensuring sponsors organization position index: %w", err)
		}
	}

	if _, err = db.Exec(`ALTER TABLE sponsors ADD COLUMN report_name TEXT`); err != nil {
		if !strings.Contains(err.Error(), "duplicate column name") {
			return nil, fmt.Errorf("error ensuring sponsors report_name column: %w", err)
		}
	}

	if _, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_sponsors_org ON sponsors(organization_id)`); err != nil {
		return nil, fmt.Errorf("error ensuring sponsors organization index: %w", err)
	}

	err = db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='sponsor_clicks';`).Scan(&tableName)
	if errors.Is(err, sql.ErrNoRows) {
		sqlStmt := `CREATE TABLE sponsor_clicks (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        event_id INTEGER NOT NULL,
        sponsor_id INTEGER NOT NULL,
        device_id TEXT,
        clicked_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
        FOREIGN KEY (event_id) REFERENCES events(id) ON DELETE CASCADE,
        FOREIGN KEY (sponsor_id) REFERENCES sponsors(id) ON DELETE CASCADE
);`
		_, err = db.Exec(sqlStmt)
		if err != nil {
			return nil, fmt.Errorf("error creating sponsor_clicks table: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("error verifying sponsor_clicks table: %w", err)
	}

	if _, err = db.Exec(`ALTER TABLE sponsor_clicks ADD COLUMN device_id TEXT`); err != nil {
		if !strings.Contains(err.Error(), "duplicate column name") {
			return nil, fmt.Errorf("error ensuring sponsor_clicks device_id column: %w", err)
		}
	}

	if _, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_sponsor_clicks_event ON sponsor_clicks(event_id)`); err != nil {
		return nil, fmt.Errorf("error ensuring sponsor_clicks event index: %w", err)
	}
	if _, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_sponsor_clicks_sponsor ON sponsor_clicks(sponsor_id)`); err != nil {
		return nil, fmt.Errorf("error ensuring sponsor_clicks sponsor index: %w", err)
	}

	err = db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='sponsor_sessions';`).Scan(&tableName)
	if errors.Is(err, sql.ErrNoRows) {
		sqlStmt := `CREATE TABLE sponsor_sessions (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        event_id INTEGER NOT NULL,
        device_id TEXT NOT NULL,
        first_seen TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
        last_seen TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
        UNIQUE(event_id, device_id),
        FOREIGN KEY (event_id) REFERENCES events(id) ON DELETE CASCADE
);`
		if _, err = db.Exec(sqlStmt); err != nil {
			return nil, fmt.Errorf("error creating sponsor_sessions table: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("error verifying sponsor_sessions table: %w", err)
	}

	if _, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_sponsor_sessions_event ON sponsor_sessions(event_id)`); err != nil {
		return nil, fmt.Errorf("error ensuring sponsor_sessions event index: %w", err)
	}

	err = db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='sponsor_exposures';`).Scan(&tableName)
	if errors.Is(err, sql.ErrNoRows) {
		sqlStmt := `CREATE TABLE sponsor_exposures (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        event_id INTEGER NOT NULL,
        sponsor_id INTEGER NOT NULL,
        device_id TEXT NOT NULL,
        exposure_type TEXT NOT NULL,
        duration_ms INTEGER,
        created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
        FOREIGN KEY (event_id) REFERENCES events(id) ON DELETE CASCADE,
        FOREIGN KEY (sponsor_id) REFERENCES sponsors(id) ON DELETE CASCADE
);`
		if _, err = db.Exec(sqlStmt); err != nil {
			return nil, fmt.Errorf("error creating sponsor_exposures table: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("error verifying sponsor_exposures table: %w", err)
	}

	if _, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_sponsor_exposures_event ON sponsor_exposures(event_id)`); err != nil {
		return nil, fmt.Errorf("error ensuring sponsor_exposures event index: %w", err)
	}
	if _, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_sponsor_exposures_sponsor ON sponsor_exposures(sponsor_id)`); err != nil {
		return nil, fmt.Errorf("error ensuring sponsor_exposures sponsor index: %w", err)
	}
	if _, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_sponsor_exposures_device_event ON sponsor_exposures(event_id, device_id)`); err != nil {
		return nil, fmt.Errorf("error ensuring sponsor_exposures device index: %w", err)
	}

	// Coupons tables
	err = db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='coupons';`).Scan(&tableName)
	if errors.Is(err, sql.ErrNoRows) {
		sqlStmt := `CREATE TABLE coupons (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        title TEXT NOT NULL,
        short_desc TEXT,
        sponsor_id INTEGER NOT NULL,
        merchant_id INTEGER NOT NULL DEFAULT 0,
        match_ids TEXT,
        start_date TEXT,
        end_date TEXT,
        max_uses INTEGER DEFAULT 0,
        status TEXT NOT NULL DEFAULT 'draft',
        image_url TEXT,
        highlight INTEGER NOT NULL DEFAULT 0,
        segmentation TEXT NOT NULL DEFAULT 'all',
        total_views INTEGER NOT NULL DEFAULT 0,
        total_claims INTEGER NOT NULL DEFAULT 0,
        total_redemptions INTEGER NOT NULL DEFAULT 0,
        created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
        updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
        organization_id INTEGER NOT NULL DEFAULT 0,
        FOREIGN KEY (sponsor_id) REFERENCES sponsors(id) ON DELETE CASCADE
);`
		if _, err = db.Exec(sqlStmt); err != nil {
			return nil, fmt.Errorf("error creating coupons table: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("error verifying coupons table: %w", err)
	}

	// Ensure merchant_id column exists for coupons
	hasMerchantColumn := false
	couponCols, err := db.Query(`PRAGMA table_info(coupons)`)
	if err != nil {
		return nil, fmt.Errorf("error inspecting coupons table: %w", err)
	}
	for couponCols.Next() {
		var (
			cid      int
			name     string
			colType  string
			notNull  int
			defaultV sql.NullString
			primary  int
		)
		if err := couponCols.Scan(&cid, &name, &colType, &notNull, &defaultV, &primary); err != nil {
			couponCols.Close()
			return nil, fmt.Errorf("error parsing coupons table info: %w", err)
		}
		if name == "merchant_id" {
			hasMerchantColumn = true
		}
	}
	couponCols.Close()
	if !hasMerchantColumn {
		if _, err = db.Exec(`ALTER TABLE coupons ADD COLUMN merchant_id INTEGER NOT NULL DEFAULT 0`); err != nil {
			return nil, fmt.Errorf("error adding merchant_id to coupons: %w", err)
		}
	}

	if _, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_coupons_sponsor ON coupons(sponsor_id)`); err != nil {
		return nil, fmt.Errorf("error ensuring coupons sponsor index: %w", err)
	}
	if _, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_coupons_org ON coupons(organization_id)`); err != nil {
		return nil, fmt.Errorf("error ensuring coupons organization index: %w", err)
	}

	err = db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='user_coupons';`).Scan(&tableName)
	if errors.Is(err, sql.ErrNoRows) {
		sqlStmt := `CREATE TABLE user_coupons (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        coupon_id INTEGER NOT NULL,
        user_id INTEGER,
        match_id INTEGER NOT NULL,
        code TEXT NOT NULL UNIQUE,
        claimed_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
        used_at TEXT,
        used_by_sponsor_id INTEGER,
        created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
        FOREIGN KEY (coupon_id) REFERENCES coupons(id) ON DELETE CASCADE
);`
		if _, err = db.Exec(sqlStmt); err != nil {
			return nil, fmt.Errorf("error creating user_coupons table: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("error verifying user_coupons table: %w", err)
	}

	if _, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_user_coupons_coupon ON user_coupons(coupon_id)`); err != nil {
		return nil, fmt.Errorf("error ensuring user_coupons coupon index: %w", err)
	}
	if _, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_user_coupons_sponsor ON user_coupons(used_by_sponsor_id)`); err != nil {
		return nil, fmt.Errorf("error ensuring user_coupons sponsor index: %w", err)
	}

	err = db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='shop_products';`).Scan(&tableName)
	if errors.Is(err, sql.ErrNoRows) {
		sqlStmt := `CREATE TABLE shop_products (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        name TEXT NOT NULL,
        description TEXT NOT NULL,
        price_cents INTEGER NOT NULL,
        image_url TEXT NOT NULL,
        created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);`
		if _, err = db.Exec(sqlStmt); err != nil {
			return nil, fmt.Errorf("error creating shop_products table: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("error verifying shop_products table: %w", err)
	}

	if _, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_shop_products_name ON shop_products(name)`); err != nil {
		return nil, fmt.Errorf("error ensuring shop_products name index: %w", err)
	}

	err = db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='shop_orders';`).Scan(&tableName)
	if errors.Is(err, sql.ErrNoRows) {
		sqlStmt := `CREATE TABLE shop_orders (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        customer_name TEXT NOT NULL,
        customer_email TEXT NOT NULL,
        customer_notes TEXT,
        total_cents INTEGER NOT NULL,
        created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);`
		if _, err = db.Exec(sqlStmt); err != nil {
			return nil, fmt.Errorf("error creating shop_orders table: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("error verifying shop_orders table: %w", err)
	}

	err = db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='shop_order_items';`).Scan(&tableName)
	if errors.Is(err, sql.ErrNoRows) {
		sqlStmt := `CREATE TABLE shop_order_items (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        order_id INTEGER NOT NULL,
        product_id INTEGER NOT NULL,
        product_name TEXT NOT NULL,
        product_image_url TEXT,
        quantity INTEGER NOT NULL,
        unit_price_cents INTEGER NOT NULL,
        FOREIGN KEY (order_id) REFERENCES shop_orders(id) ON DELETE CASCADE,
        FOREIGN KEY (product_id) REFERENCES shop_products(id)
);`
		if _, err = db.Exec(sqlStmt); err != nil {
			return nil, fmt.Errorf("error creating shop_order_items table: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("error verifying shop_order_items table: %w", err)
	}

	if _, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_shop_order_items_order ON shop_order_items(order_id)`); err != nil {
		return nil, fmt.Errorf("error ensuring shop_order_items order index: %w", err)
	}
	if _, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_shop_order_items_product ON shop_order_items(product_id)`); err != nil {
		return nil, fmt.Errorf("error ensuring shop_order_items product index: %w", err)
	}

	var shopProductCount int
	if err = db.QueryRow(`SELECT COUNT(*) FROM shop_products`).Scan(&shopProductCount); err != nil {
		return nil, fmt.Errorf("error counting shop_products: %w", err)
	}

	if shopProductCount == 0 {
		sampleProducts := []ShopProduct{
			{
				Name:        "Wearing Cash Street Hoodie",
				Description: "Felpa premium in cotone organico con logo ricamato Wearing Cash, ideale per le serate fresche in città.",
				PriceCents:  7900,
				ImageURL:    "https://dummyimage.com/600x600/111827/ffffff&text=Wearing+Cash+Hoodie",
			},
			{
				Name:        "Wearing Cash Signature Tee",
				Description: "T-shirt leggera unisex con stampa frontale Wearing Cash Signature, perfetta per ogni outfit quotidiano.",
				PriceCents:  3200,
				ImageURL:    "https://dummyimage.com/600x600/1f2937/ffffff&text=Wearing+Cash+Tee",
			},
			{
				Name:        "Wearing Cash Essential Cap",
				Description: "Cappellino regolabile in tessuto tecnico con visiera curva e ricamo tono su tono del brand.",
				PriceCents:  2800,
				ImageURL:    "https://dummyimage.com/600x600/0f172a/ffffff&text=Wearing+Cash+Cap",
			},
			{
				Name:        "Wearing Cash Jetset Bomber",
				Description: "Bomber leggero con fodera satinata e dettagli metal per uno stile urbano deciso e contemporaneo.",
				PriceCents:  12900,
				ImageURL:    "https://dummyimage.com/600x600/0b1120/ffffff&text=Wearing+Cash+Bomber",
			},
			{
				Name:        "Wearing Cash Iconic Sneakers",
				Description: "Sneakers in pelle con inserti reflective e suola memory foam per un comfort premium 24/7.",
				PriceCents:  14900,
				ImageURL:    "https://dummyimage.com/600x600/111111/ffffff&text=Wearing+Cash+Sneakers",
			},
		}

		for _, product := range sampleProducts {
			if _, err = db.Exec(`INSERT INTO shop_products (name, description, price_cents, image_url) VALUES (?, ?, ?, ?)`, product.Name, product.Description, product.PriceCents, product.ImageURL); err != nil {
				return nil, fmt.Errorf("error seeding shop_products: %w", err)
			}
		}
	}

	if err = ensureFanProfileTables(db); err != nil {
		return nil, err
	}

	return &appdbimpl{
		c: db,
	}, nil
}

func ensureAdminsTable(db *sql.DB) error {
	rows, err := db.Query(`PRAGMA table_info(admins)`)
	if err != nil {
		return fmt.Errorf("error inspecting admins table: %w", err)
	}
	defer rows.Close()

	hasAdmins := false
	hasOrgColumn := false
	for rows.Next() {
		hasAdmins = true
		var (
			cid      int
			name     string
			colType  string
			notNull  int
			defaultV sql.NullString
			primary  int
		)
		if err := rows.Scan(&cid, &name, &colType, &notNull, &defaultV, &primary); err != nil {
			return fmt.Errorf("error parsing admins table info: %w", err)
		}
		if name == "organization_id" {
			hasOrgColumn = true
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("error reading admins table info: %w", err)
	}

	if !hasAdmins {
		if err := createAdminsTable(db); err != nil {
			return err
		}
	} else if !hasOrgColumn {
		if err := recreateAdminsTable(db); err != nil {
			return err
		}
	}

	if _, err := db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_admins_org_username ON admins(organization_id, username);`); err != nil {
		return fmt.Errorf("error ensuring admins organization index: %w", err)
	}

	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS qr_redirects (
id INTEGER PRIMARY KEY AUTOINCREMENT,
source_path TEXT NOT NULL,
target_path TEXT NOT NULL,
hits INTEGER NOT NULL DEFAULT 0,
created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);`); err != nil {
		return fmt.Errorf("error ensuring qr_redirects table: %w", err)
	}
	if _, err := db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_qr_redirects_source ON qr_redirects(source_path);`); err != nil {
		return fmt.Errorf("error ensuring qr_redirects source index: %w", err)
	}

	return nil
}

func createAdminsTable(db *sql.DB) error {
	if _, err := db.Exec(`CREATE TABLE admins (
id INTEGER PRIMARY KEY AUTOINCREMENT,
username TEXT NOT NULL,
password_hash TEXT NOT NULL,
role TEXT NOT NULL DEFAULT 'staff',
organization_id INTEGER,
created_at TEXT DEFAULT CURRENT_TIMESTAMP,
FOREIGN KEY (organization_id) REFERENCES organizations(id)
);`); err != nil {
		return fmt.Errorf("error creating admins table: %w", err)
	}
	return nil
}

func recreateAdminsTable(db *sql.DB) error {
	if _, err := db.Exec(`DROP TABLE IF EXISTS admins_old;`); err != nil {
		return fmt.Errorf("error preparing legacy admins table: %w", err)
	}

	if _, err := db.Exec(`ALTER TABLE admins RENAME TO admins_old;`); err != nil {
		return fmt.Errorf("error renaming admins table: %w", err)
	}

	if err := createAdminsTable(db); err != nil {
		return err
	}

	if _, err := db.Exec(`INSERT INTO admins (id, username, password_hash, role, organization_id, created_at) SELECT id, username, password_hash, role, NULL, created_at FROM admins_old;`); err != nil {
		return fmt.Errorf("error copying data into admins table: %w", err)
	}

	if _, err := db.Exec(`DROP TABLE admins_old;`); err != nil {
		return fmt.Errorf("error dropping legacy admins table: %w", err)
	}

	return nil
}

func hashPasswordSHA256(password string) string {
	sum := sha256.Sum256([]byte(password))
	return hex.EncodeToString(sum[:])
}

func createDefaultStaffAdmin(tx *sql.Tx, organizationID int) error {
	if organizationID <= 0 {
		return fmt.Errorf("invalid organization id for default admin")
	}

	_, err := tx.Exec(`INSERT INTO admins (username, password_hash, role, organization_id) VALUES (?, ?, 'staff', ?)`, "staff", hashPasswordSHA256("staff"), organizationID)
	return err
}

func (db *appdbimpl) Ping() error {
	return db.c.Ping()
}

// AddVote stores a vote in the database. If the device already voted for the event,
// the latest choice replaces the previous one (including the ticket data).
func (db *appdbimpl) AddVote(eventID, playerID int, code, signature, deviceID string) error {
	_, err := db.c.Exec(`INSERT INTO votes (event_id, player_id, ticket_code, ticket_signature, device_id)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT(event_id, device_id) DO UPDATE SET
player_id = excluded.player_id,
ticket_code = excluded.ticket_code,
ticket_signature = excluded.ticket_signature,
created_at = CURRENT_TIMESTAMP`, eventID, playerID, code, signature, deviceID)
	return err
}

// GetEventVoteCount returns the total number of votes for a specific event
func (db *appdbimpl) GetEventVoteCount(eventID int) (int, error) {
	var count int
	err := db.c.QueryRow(`SELECT COUNT(1) FROM votes WHERE event_id = ?`, eventID).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (db *appdbimpl) GetEventVoteLeaderboard(eventID, limit int) ([]EventVoteLeaderboardEntry, error) {
	if limit <= 0 {
		limit = 5
	}
	if limit > 50 {
		limit = 50
	}

	rows, err := db.c.Query(`
SELECT v.player_id,
       IFNULL(p.first_name, ''),
       IFNULL(p.last_name, ''),
       IFNULL(p.image_url, ''),
       COUNT(v.id) AS votes,
       IFNULL(MAX(v.created_at), '') AS last_vote_at
FROM votes v
INNER JOIN players p ON p.id = v.player_id
WHERE v.event_id = ?
GROUP BY v.player_id, p.first_name, p.last_name, p.image_url
ORDER BY votes DESC, last_vote_at DESC, v.player_id ASC
LIMIT ?
`, eventID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries := make([]EventVoteLeaderboardEntry, 0)
	for rows.Next() {
		var entry EventVoteLeaderboardEntry
		var lastVoteRaw string
		if err := rows.Scan(&entry.PlayerID, &entry.FirstName, &entry.LastName, &entry.ImageURL, &entry.Votes, &lastVoteRaw); err != nil {
			return nil, err
		}

		if ts, parseErr := parseSQLiteTimestamp(lastVoteRaw); parseErr == nil {
			entry.LastVoteAt = ts.UTC().Format(time.RFC3339)
		} else {
			entry.LastVoteAt = strings.TrimSpace(lastVoteRaw)
		}

		entry.FirstName = strings.TrimSpace(entry.FirstName)
		entry.LastName = strings.TrimSpace(entry.LastName)
		entry.ImageURL = strings.TrimSpace(entry.ImageURL)
		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return entries, nil
}

func (db *appdbimpl) ListEventVoteTimestamps(eventID int) ([]time.Time, error) {
	rows, err := db.c.Query(`SELECT created_at FROM votes WHERE event_id = ? ORDER BY created_at ASC`, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var timestamps []time.Time
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		ts, parseErr := parseSQLiteTimestamp(raw)
		if parseErr != nil {
			continue
		}
		timestamps = append(timestamps, ts)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return timestamps, nil
}

func (db *appdbimpl) GetEventMVP(eventID int) (EventMVP, error) {
	row := db.c.QueryRow(`
SELECT v.player_id,
       IFNULL(p.first_name, ''),
        IFNULL(p.last_name, ''),
        COUNT(v.id) AS votes,
        IFNULL(MAX(v.created_at), '')
FROM votes v
INNER JOIN players p ON p.id = v.player_id
WHERE v.event_id = ?
GROUP BY v.player_id, p.first_name, p.last_name
ORDER BY votes DESC, MAX(v.created_at) DESC, v.player_id ASC
LIMIT 1
`, eventID)

	var mvp EventMVP
	var lastVoteRaw string
	if err := row.Scan(&mvp.PlayerID, &mvp.FirstName, &mvp.LastName, &mvp.Votes, &lastVoteRaw); err != nil {
		return EventMVP{}, err
	}

	if ts, err := parseSQLiteTimestamp(lastVoteRaw); err == nil && !ts.IsZero() {
		mvp.LastVoteAt = ts.UTC().Format(time.RFC3339)
	} else {
		mvp.LastVoteAt = strings.TrimSpace(lastVoteRaw)
	}

	return mvp, nil
}

func (db *appdbimpl) HasDeviceVoted(eventID int, deviceID string) (bool, error) {
	if eventID <= 0 || strings.TrimSpace(deviceID) == "" {
		return false, nil
	}

	var exists int
	err := db.c.QueryRow(`SELECT 1 FROM votes WHERE event_id = ? AND device_id = ? LIMIT 1`, eventID, deviceID).Scan(&exists)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return exists == 1, nil
}

// GetDeviceVote returns the vote cast by a device for a specific event, if interface{}.
func (db *appdbimpl) GetDeviceVote(eventID int, deviceID string) (Vote, error) {
	if eventID <= 0 || strings.TrimSpace(deviceID) == "" {
		return Vote{}, sql.ErrNoRows
	}

	var vote Vote
	var createdRaw string
	err := db.c.QueryRow(`SELECT id, event_id, player_id, ticket_code, ticket_signature, device_id, created_at FROM votes WHERE event_id = ? AND device_id = ? LIMIT 1`, eventID, deviceID).Scan(&vote.ID, &vote.EventID, &vote.PlayerID, &vote.TicketCode, &vote.TicketSignature, &vote.DeviceID, &createdRaw)
	if err != nil {
		return Vote{}, err
	}

	if ts, parseErr := parseSQLiteTimestamp(createdRaw); parseErr == nil && !ts.IsZero() {
		vote.CreatedAt = ts.UTC().Format(time.RFC3339)
	} else {
		vote.CreatedAt = strings.TrimSpace(createdRaw)
	}

	return vote, nil
}

func scanSelfieRow(scanner rowScanner) (Selfie, error) {
	var s Selfie
	var approved, showOnScreen, acceptedImageTerms int
	var createdRaw string
	if err := scanner.Scan(&s.ID, &s.EventID, &s.DeviceID, &s.Caption, &s.ImagePath, &s.ImageURL, &s.ContentType, &approved, &showOnScreen, &acceptedImageTerms, &createdRaw); err != nil {
		return Selfie{}, err
	}
	s.Approved = approved == 1
	s.ShowOnScreen = showOnScreen == 1
	s.AcceptedImageTerms = acceptedImageTerms == 1
	if ts, err := parseSQLiteTimestamp(createdRaw); err == nil && !ts.IsZero() {
		s.CreatedAt = ts.UTC().Format(time.RFC3339)
	} else {
		s.CreatedAt = strings.TrimSpace(createdRaw)
	}
	return s, nil
}

func (db *appdbimpl) SaveSelfie(eventID int, deviceID, caption, imagePath, contentType string, acceptedImageTerms bool) (Selfie, error) {
	deviceID = strings.TrimSpace(deviceID)
	if eventID <= 0 || deviceID == "" || strings.TrimSpace(imagePath) == "" {
		return Selfie{}, fmt.Errorf("invalid selfie payload")
	}

	if len([]rune(caption)) > 80 {
		caption = string([]rune(caption)[:80])
	}

	tx, err := db.c.Begin()
	if err != nil {
		return Selfie{}, err
	}
	defer tx.Rollback()

	result, err := tx.Exec(`
INSERT INTO selfies (event_id, device_id, caption, image_path, image_url, content_type, approved, show_on_screen, accepted_image_terms, created_at)
VALUES (?, ?, ?, ?, '', ?, 0, 0, ?, CURRENT_TIMESTAMP)
ON CONFLICT(event_id, device_id) DO UPDATE SET
caption=excluded.caption,
image_path=excluded.image_path,
image_url=excluded.image_url,
content_type=excluded.content_type,
approved=0,
show_on_screen=0,
accepted_image_terms=excluded.accepted_image_terms,
created_at=CURRENT_TIMESTAMP
`, eventID, deviceID, strings.TrimSpace(caption), strings.TrimSpace(imagePath), strings.TrimSpace(contentType), boolToInt(acceptedImageTerms))
	if err != nil {
		return Selfie{}, err
	}

	var selfieID int
	if id, err := result.LastInsertId(); err == nil && id > 0 {
		selfieID = int(id)
	}
	if selfieID == 0 {
		if err := tx.QueryRow(`SELECT id FROM selfies WHERE event_id = ? AND device_id = ?`, eventID, deviceID).Scan(&selfieID); err != nil {
			return Selfie{}, err
		}
	}

	if err := tx.Commit(); err != nil {
		return Selfie{}, err
	}

	return db.GetSelfieByID(selfieID)
}

func (db *appdbimpl) UpdateSelfieURL(id int, imageURL string) error {
	_, err := db.c.Exec(`UPDATE selfies SET image_url = ? WHERE id = ?`, strings.TrimSpace(imageURL), id)
	return err
}

func (db *appdbimpl) GetSelfieForDevice(eventID int, deviceID string) (Selfie, error) {
	row := db.c.QueryRow(`SELECT id, event_id, device_id, caption, image_path, image_url, content_type, approved, show_on_screen, accepted_image_terms, created_at FROM selfies WHERE event_id = ? AND device_id = ?`, eventID, deviceID)
	return scanSelfieRow(row)
}

func (db *appdbimpl) GetSelfieByID(id int) (Selfie, error) {
	row := db.c.QueryRow(`SELECT id, event_id, device_id, caption, image_path, image_url, content_type, approved, show_on_screen, accepted_image_terms, created_at FROM selfies WHERE id = ?`, id)
	return scanSelfieRow(row)
}

func (db *appdbimpl) ListEventSelfies(eventID int) ([]Selfie, error) {
	rows, err := db.c.Query(`SELECT id, event_id, device_id, caption, image_path, image_url, content_type, approved, show_on_screen, accepted_image_terms, created_at FROM selfies WHERE event_id = ? ORDER BY created_at DESC, id DESC`, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var selfies []Selfie
	for rows.Next() {
		selfie, err := scanSelfieRow(rows)
		if err != nil {
			return nil, err
		}
		selfies = append(selfies, selfie)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return selfies, nil
}

func (db *appdbimpl) ListApprovedSelfies(eventID int) ([]Selfie, error) {
	rows, err := db.c.Query(`SELECT id, event_id, device_id, caption, image_path, image_url, content_type, approved, show_on_screen, accepted_image_terms, created_at FROM selfies WHERE event_id = ? AND approved = 1 ORDER BY created_at DESC, id DESC`, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var selfies []Selfie
	for rows.Next() {
		selfie, err := scanSelfieRow(rows)
		if err != nil {
			return nil, err
		}
		selfies = append(selfies, selfie)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return selfies, nil
}

func (db *appdbimpl) UpdateSelfieStatus(id int, approved bool, showOnScreen bool) error {
	if !approved {
		showOnScreen = false
	}
	_, err := db.c.Exec(`UPDATE selfies SET approved = ?, show_on_screen = ? WHERE id = ?`, boolToInt(approved), boolToInt(showOnScreen), id)
	return err
}

func (db *appdbimpl) DeleteSelfie(id int) error {
	_, err := db.c.Exec(`DELETE FROM selfies WHERE id = ?`, id)
	return err
}

func scanReactionTestAttempt(row rowScanner) (ReactionTestAttempt, error) {
	var attempt ReactionTestAttempt
	var createdAtRaw string
	var isValid int
	if err := row.Scan(&attempt.ID, &attempt.EventID, &attempt.DeviceID, &attempt.ReactionTimeMs, &isValid, &createdAtRaw); err != nil {
		return ReactionTestAttempt{}, err
	}
	attempt.IsValid = isValid != 0
	if createdAtRaw != "" {
		if ts, err := parseSQLiteTimestamp(createdAtRaw); err == nil {
			attempt.CreatedAt = ts
		}
	}
	return attempt, nil
}

func (db *appdbimpl) RecordReactionTestAttempt(eventID int, deviceID string, reactionMs int) (ReactionTestAttempt, error) {
	res, err := db.c.Exec(`INSERT INTO reaction_tests (event_id, device_id, reaction_time_ms, is_valid) VALUES (?, ?, ?, 1)`, eventID, deviceID, reactionMs)
	if err != nil {
		return ReactionTestAttempt{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return ReactionTestAttempt{}, err
	}
	row := db.c.QueryRow(`SELECT id, event_id, device_id, reaction_time_ms, is_valid, created_at FROM reaction_tests WHERE id = ?`, id)
	return scanReactionTestAttempt(row)
}

func (db *appdbimpl) GetLatestReactionTestAttempt(eventID int, deviceID string) (ReactionTestAttempt, error) {
	row := db.c.QueryRow(`SELECT id, event_id, device_id, reaction_time_ms, is_valid, created_at FROM reaction_tests WHERE event_id = ? AND device_id = ? ORDER BY created_at DESC, id DESC LIMIT 1`, eventID, deviceID)
	return scanReactionTestAttempt(row)
}

func (db *appdbimpl) GetReactionTestStats(eventID int) (ReactionTestStats, error) {
	var stats ReactionTestStats
	var avg sql.NullFloat64
	err := db.c.QueryRow(`SELECT COUNT(1) AS attempts, AVG(reaction_time_ms) FROM reaction_tests WHERE event_id = ? AND is_valid = 1`, eventID).Scan(&stats.Attempts, &avg)
	if err != nil {
		return ReactionTestStats{}, err
	}
	if avg.Valid {
		stats.Average = avg.Float64
	}
	return stats, nil
}

func (db *appdbimpl) GetEventQuizConfig(eventID int) (EventQuizConfig, error) {
	var cfg EventQuizConfig
	var enabled int
	var from sql.NullString
	var to sql.NullString
	err := db.c.QueryRow(`SELECT event_id, enabled, questions_per_session, seconds_per_question, base_reward, completion_bonus, streak_bonus, active_from, active_to FROM event_quiz WHERE event_id = ?`, eventID).Scan(&cfg.EventID, &enabled, &cfg.QuestionsPerSession, &cfg.SecondsPerQuestion, &cfg.BaseReward, &cfg.CompletionBonus, &cfg.StreakBonus, &from, &to)
	if err != nil {
		return EventQuizConfig{}, err
	}
	cfg.Enabled = enabled != 0
	if from.Valid {
		cfg.ActiveFrom = from.String
	}
	if to.Valid {
		cfg.ActiveTo = to.String
	}
	return cfg, nil
}

func (db *appdbimpl) UpsertEventQuizConfig(config EventQuizConfig) (EventQuizConfig, error) {
	_, err := db.c.Exec(`INSERT INTO event_quiz (event_id, enabled, questions_per_session, seconds_per_question, base_reward, completion_bonus, streak_bonus, active_from, active_to)
VALUES (?, ?, ?, ?, ?, ?, ?, NULLIF(?, ''), NULLIF(?, ''))
ON CONFLICT(event_id) DO UPDATE SET enabled=excluded.enabled, questions_per_session=excluded.questions_per_session, seconds_per_question=excluded.seconds_per_question, base_reward=excluded.base_reward, completion_bonus=excluded.completion_bonus, streak_bonus=excluded.streak_bonus, active_from=excluded.active_from, active_to=excluded.active_to`, config.EventID, boolToInt(config.Enabled), config.QuestionsPerSession, config.SecondsPerQuestion, config.BaseReward, config.CompletionBonus, config.StreakBonus, strings.TrimSpace(config.ActiveFrom), strings.TrimSpace(config.ActiveTo))
	if err != nil {
		return EventQuizConfig{}, err
	}
	return db.GetEventQuizConfig(config.EventID)
}

func (db *appdbimpl) ListEventQuizQuestions(eventID int) ([]EventQuizQuestion, error) {
	rows, err := db.c.Query(`SELECT id, quiz_id, question_text, answers_json, correct_index, order_index FROM event_quiz_questions WHERE quiz_id = ? ORDER BY order_index ASC, id ASC`, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]EventQuizQuestion, 0)
	for rows.Next() {
		var q EventQuizQuestion
		var answersJSON string
		if err := rows.Scan(&q.ID, &q.QuizID, &q.QuestionText, &answersJSON, &q.CorrectIndex, &q.OrderIndex); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(answersJSON), &q.Answers)
		out = append(out, q)
	}
	return out, rows.Err()
}

func (db *appdbimpl) CreateEventQuizQuestion(eventID int, question EventQuizQuestion) (EventQuizQuestion, error) {
	b, _ := json.Marshal(question.Answers)
	res, err := db.c.Exec(`INSERT INTO event_quiz_questions (quiz_id, question_text, answers_json, correct_index, order_index) VALUES (?, ?, ?, ?, ?)`, eventID, strings.TrimSpace(question.QuestionText), string(b), question.CorrectIndex, question.OrderIndex)
	if err != nil {
		return EventQuizQuestion{}, err
	}
	id, _ := res.LastInsertId()
	return db.GetEventQuizQuestion(eventID, int(id))
}

func (db *appdbimpl) UpdateEventQuizQuestion(eventID int, questionID int, question EventQuizQuestion) (EventQuizQuestion, error) {
	b, _ := json.Marshal(question.Answers)
	_, err := db.c.Exec(`UPDATE event_quiz_questions SET question_text=?, answers_json=?, correct_index=?, order_index=? WHERE id=? AND quiz_id=?`, strings.TrimSpace(question.QuestionText), string(b), question.CorrectIndex, question.OrderIndex, questionID, eventID)
	if err != nil {
		return EventQuizQuestion{}, err
	}
	return db.GetEventQuizQuestion(eventID, questionID)
}

func (db *appdbimpl) DeleteEventQuizQuestion(eventID int, questionID int) error {
	_, err := db.c.Exec(`DELETE FROM event_quiz_questions WHERE id = ? AND quiz_id = ?`, questionID, eventID)
	return err
}

func (db *appdbimpl) GetEventQuizQuestion(eventID int, questionID int) (EventQuizQuestion, error) {
	var q EventQuizQuestion
	var answersJSON string
	err := db.c.QueryRow(`SELECT id, quiz_id, question_text, answers_json, correct_index, order_index FROM event_quiz_questions WHERE id = ? AND quiz_id = ?`, questionID, eventID).Scan(&q.ID, &q.QuizID, &q.QuestionText, &answersJSON, &q.CorrectIndex, &q.OrderIndex)
	if err != nil {
		return EventQuizQuestion{}, err
	}
	_ = json.Unmarshal([]byte(answersJSON), &q.Answers)
	return q, nil
}

// Team operations
func (db *appdbimpl) CreateTeam(name, championship string) (int, error) {
	res, err := db.c.Exec(`INSERT INTO teams (name, championship) VALUES (?, ?)`, name, championship)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	return int(id), nil
}

func (db *appdbimpl) ListTeams() ([]Team, error) {
	rows, err := db.c.Query(`SELECT id, name, championship FROM teams`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ts []Team
	for rows.Next() {
		var t Team
		if err := rows.Scan(&t.ID, &t.Name, &t.Championship); err != nil {
			return nil, err
		}
		ts = append(ts, t)
	}
	return ts, nil
}

func (db *appdbimpl) UpdateTeam(id int, name, championship string) error {
	_, err := db.c.Exec(`UPDATE teams SET name=?, championship=? WHERE id=?`, name, championship, id)
	return err
}

func (db *appdbimpl) DeleteTeam(id int) error {
	_, err := db.c.Exec(`DELETE FROM teams WHERE id=?`, id)
	return err
}

// Player operations
func (db *appdbimpl) CreatePlayer(p Player) (int, error) {
	res, err := db.c.Exec(`INSERT INTO players (first_name, last_name, role, jersey_number, image_url, is_called_up, team_id, organization_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, p.FirstName, p.LastName, p.Role, p.JerseyNumber, p.ImageURL, boolToInt(p.IsCalledUp), p.TeamID, p.OrganizationID)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	return int(id), nil
}

func (db *appdbimpl) GetPlayerByID(id int) (Player, error) {
	var p Player
	row := db.c.QueryRow(`SELECT id, first_name, last_name, role, jersey_number, image_url, is_called_up, team_id, organization_id FROM players WHERE id = ?`, id)
	var isCalledUp int
	if err := row.Scan(&p.ID, &p.FirstName, &p.LastName, &p.Role, &p.JerseyNumber, &p.ImageURL, &isCalledUp, &p.TeamID, &p.OrganizationID); err != nil {
		return Player{}, err
	}
	p.IsCalledUp = isCalledUp != 0
	return p, nil
}

func (db *appdbimpl) ListPlayers() ([]Player, error) {
	rows, err := db.c.Query(`SELECT id, first_name, last_name, role, jersey_number, image_url, is_called_up, team_id, organization_id FROM players`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ps []Player
	for rows.Next() {
		var p Player
		var isCalledUp int
		if err := rows.Scan(&p.ID, &p.FirstName, &p.LastName, &p.Role, &p.JerseyNumber, &p.ImageURL, &isCalledUp, &p.TeamID, &p.OrganizationID); err != nil {
			return nil, err
		}
		p.IsCalledUp = isCalledUp != 0
		ps = append(ps, p)
	}
	return ps, nil
}

func (db *appdbimpl) ListPlayersByOrganization(organizationID int) ([]Player, error) {
	rows, err := db.c.Query(`SELECT id, first_name, last_name, role, jersey_number, image_url, is_called_up, team_id, organization_id FROM players WHERE organization_id = ?`, organizationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ps []Player
	for rows.Next() {
		var p Player
		var isCalledUp int
		if err := rows.Scan(&p.ID, &p.FirstName, &p.LastName, &p.Role, &p.JerseyNumber, &p.ImageURL, &isCalledUp, &p.TeamID, &p.OrganizationID); err != nil {
			return nil, err
		}
		p.IsCalledUp = isCalledUp != 0
		ps = append(ps, p)
	}
	return ps, nil
}

func (db *appdbimpl) UpdatePlayer(p Player) error {
	_, err := db.c.Exec(`UPDATE players SET first_name=?, last_name=?, role=?, jersey_number=?, image_url=?, is_called_up=?, team_id=?, organization_id=? WHERE id=?`, p.FirstName, p.LastName, p.Role, p.JerseyNumber, p.ImageURL, boolToInt(p.IsCalledUp), p.TeamID, p.OrganizationID, p.ID)
	return err
}

func (db *appdbimpl) DeletePlayer(id int) error {
	_, err := db.c.Exec(`DELETE FROM players WHERE id=?`, id)
	return err
}

// Event operations
func (db *appdbimpl) ListEventStories(eventID int, includeInactive bool) ([]EventStory, error) {
	query := `SELECT id, event_id, player_name, thumbnail_url, video_url, IFNULL(title, ''), is_active, order_index
		FROM event_stories WHERE event_id = ?`
	if !includeInactive {
		query += ` AND is_active = 1`
	}
	query += ` ORDER BY order_index ASC, id ASC`

	rows, err := db.c.Query(query, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stories := make([]EventStory, 0)
	for rows.Next() {
		var story EventStory
		var active int
		if err := rows.Scan(&story.ID, &story.EventID, &story.PlayerName, &story.ThumbnailURL, &story.VideoURL, &story.Title, &active, &story.OrderIndex); err != nil {
			return nil, err
		}
		story.IsActive = active == 1
		stories = append(stories, story)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return stories, nil
}

func (db *appdbimpl) CreateEventStory(eventID int, story EventStory) (EventStory, error) {
	res, err := db.c.Exec(`INSERT INTO event_stories (event_id, player_name, thumbnail_url, video_url, title, is_active, order_index)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		eventID,
		strings.TrimSpace(story.PlayerName),
		strings.TrimSpace(story.ThumbnailURL),
		strings.TrimSpace(story.VideoURL),
		strings.TrimSpace(story.Title),
		boolToInt(story.IsActive),
		story.OrderIndex,
	)
	if err != nil {
		return EventStory{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return EventStory{}, err
	}
	items, err := db.ListEventStories(eventID, true)
	if err != nil {
		return EventStory{}, err
	}
	for _, item := range items {
		if item.ID == int(id) {
			return item, nil
		}
	}
	return EventStory{}, sql.ErrNoRows
}

func (db *appdbimpl) UpdateEventStory(eventID int, storyID int, story EventStory) (EventStory, error) {
	res, err := db.c.Exec(`UPDATE event_stories
		SET player_name = ?, thumbnail_url = ?, video_url = ?, title = ?, is_active = ?, order_index = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND event_id = ?`,
		strings.TrimSpace(story.PlayerName),
		strings.TrimSpace(story.ThumbnailURL),
		strings.TrimSpace(story.VideoURL),
		strings.TrimSpace(story.Title),
		boolToInt(story.IsActive),
		story.OrderIndex,
		storyID,
		eventID,
	)
	if err != nil {
		return EventStory{}, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return EventStory{}, err
	}
	if affected == 0 {
		return EventStory{}, sql.ErrNoRows
	}
	items, err := db.ListEventStories(eventID, true)
	if err != nil {
		return EventStory{}, err
	}
	for _, item := range items {
		if item.ID == storyID {
			return item, nil
		}
	}
	return EventStory{}, sql.ErrNoRows
}

func (db *appdbimpl) DeleteEventStory(eventID int, storyID int) error {
	res, err := db.c.Exec(`DELETE FROM event_stories WHERE id = ? AND event_id = ?`, storyID, eventID)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func sanitizePrizeInputs(prizes []EventPrize) []EventPrize {
	cleaned := make([]EventPrize, 0, len(prizes))
	for _, prize := range prizes {
		name := strings.TrimSpace(prize.Name)
		if name == "" {
			continue
		}
		cleaned = append(cleaned, EventPrize{
			ID:       prize.ID,
			EventID:  prize.EventID,
			Name:     name,
			Position: prize.Position,
		})
	}
	if len(cleaned) == 0 {
		return cleaned
	}
	sort.SliceStable(cleaned, func(i, j int) bool {
		if cleaned[i].Position == cleaned[j].Position {
			return i < j
		}
		if cleaned[i].Position <= 0 {
			return false
		}
		if cleaned[j].Position <= 0 {
			return true
		}
		return cleaned[i].Position < cleaned[j].Position
	})
	for idx := range cleaned {
		cleaned[idx].Position = idx + 1
	}
	return cleaned
}

func (db *appdbimpl) syncEventPrizesTx(tx *sql.Tx, eventID int, prizes []EventPrize) error {
	cleaned := sanitizePrizeInputs(prizes)

	if _, err := tx.Exec(`UPDATE event_prizes SET position = position + 1000 WHERE event_id = ?`, eventID); err != nil {
		return err
	}

	rows, err := tx.Query(`SELECT id, IFNULL(winner_vote_id, 0) FROM event_prizes WHERE event_id = ?`, eventID)
	if err != nil {
		return err
	}
	defer rows.Close()

	type existingPrize struct {
		id        int
		hasWinner bool
	}

	existing := make(map[int]existingPrize)
	for rows.Next() {
		var id int
		var winnerVoteID int
		if err := rows.Scan(&id, &winnerVoteID); err != nil {
			return err
		}
		existing[id] = existingPrize{id: id, hasWinner: winnerVoteID > 0}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	processed := make(map[int]struct{})

	for _, prize := range cleaned {
		if prize.ID > 0 {
			if _, ok := existing[prize.ID]; ok {
				if _, err := tx.Exec(`UPDATE event_prizes SET name = ?, position = ? WHERE id = ?`, prize.Name, prize.Position, prize.ID); err != nil {
					return err
				}
				processed[prize.ID] = struct{}{}
				continue
			}
		}
		if _, err := tx.Exec(`INSERT INTO event_prizes (event_id, name, position) VALUES (?, ?, ?)`, eventID, prize.Name, prize.Position); err != nil {
			return err
		}
	}

	for id, info := range existing {
		if _, ok := processed[id]; ok {
			continue
		}
		if info.hasWinner {
			return ErrPrizeLockedByWinner
		}
		if _, err := tx.Exec(`DELETE FROM event_prizes WHERE id = ?`, id); err != nil {
			return err
		}
	}

	return nil
}

func (db *appdbimpl) CreateEvent(e Event) (int, error) {
	tx, err := db.c.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	survey := NormalizeEventFeedbackSurveyConfig(e.FeedbackSurvey)
	e.FeedbackSurvey = &survey
	surveyJSON := encodeEventFeedbackSurveyConfig(survey)

	res, err := tx.Exec(`INSERT INTO events (organization_id, team1_id, team2_id, start_datetime, location, show_reaction_test, show_selfie, show_vote_trend, show_feedback_survey, show_pre_vote_sponsors, show_pre_vote_bottom_sponsors, show_vote_counter, feedback_survey_config) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, e.OrganizationID, e.Team1ID, e.Team2ID, e.StartDateTime, e.Location, boolToInt(e.ShowReactionTest), boolToInt(e.ShowSelfie), boolToInt(e.ShowVoteTrend), boolToInt(e.ShowFeedbackSurvey), boolToInt(e.ShowPreVoteSponsors), boolToInt(e.ShowPreVoteBottomSponsors), boolToInt(e.ShowVoteCounter), surveyJSON)
	if err != nil {
		return 0, err
	}
	id64, _ := res.LastInsertId()
	eventID := int(id64)

	if err := db.syncEventPrizesTx(tx, eventID, e.Prizes); err != nil {
		return 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return eventID, nil
}

func (db *appdbimpl) ListEvents() ([]Event, error) {
	rows, err := db.c.Query(`
SELECT e.id,
       e.organization_id,
       e.team1_id,
       e.team2_id,
       e.start_datetime,
       e.location,
       e.is_active,
       e.votes_closed,
       e.is_concluded,
       e.show_reaction_test,
       e.show_selfie,
       e.show_vote_trend,
       e.show_feedback_survey,
       e.show_pre_vote_sponsors,
       e.show_pre_vote_bottom_sponsors,
       e.show_vote_counter,
       e.feedback_survey_config,
       IFNULL(t1.name, ''),
       IFNULL(t2.name, '')
FROM events e
LEFT JOIN teams t1 ON t1.id = e.team1_id
LEFT JOIN teams t2 ON t2.id = e.team2_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var es []Event
	for rows.Next() {
		var e Event
		var isActive int
		var votesClosed int
		var isConcluded int
		var showReaction int
		var showSelfie int
		var showVoteTrend int
		var showFeedback int
		var showPreVoteSponsors int
		var showPreVoteBottomSponsors int
		var showVoteCounter int
		var surveyConfig sql.NullString
		if err := rows.Scan(&e.ID, &e.OrganizationID, &e.Team1ID, &e.Team2ID, &e.StartDateTime, &e.Location, &isActive, &votesClosed, &isConcluded, &showReaction, &showSelfie, &showVoteTrend, &showFeedback, &showPreVoteSponsors, &showPreVoteBottomSponsors, &showVoteCounter, &surveyConfig, &e.Team1Name, &e.Team2Name); err != nil {
			return nil, err
		}
		e.IsActive = isActive == 1
		e.VotesClosed = votesClosed == 1
		e.IsConcluded = isConcluded == 1
		e.ShowReactionTest = showReaction == 1
		e.ShowSelfie = showSelfie == 1
		e.ShowVoteTrend = showVoteTrend == 1
		e.ShowFeedbackSurvey = showFeedback == 1
		e.ShowPreVoteSponsors = showPreVoteSponsors == 1
		e.ShowPreVoteBottomSponsors = showPreVoteBottomSponsors == 1
		e.ShowVoteCounter = showVoteCounter == 1
		cfg := decodeEventFeedbackSurveyConfig(surveyConfig)
		e.FeedbackSurvey = &cfg
		es = append(es, e)
	}
	for i := range es {
		prizes, err := db.ListEventPrizes(es[i].ID)
		if err != nil {
			return nil, err
		}
		es[i].Prizes = prizes
	}

	return es, nil
}

func (db *appdbimpl) ListEventsByOrganization(organizationID int) ([]Event, error) {
	rows, err := db.c.Query(`
SELECT e.id,
       e.organization_id,
       e.team1_id,
       e.team2_id,
       e.start_datetime,
       e.location,
       e.is_active,
       e.votes_closed,
       e.is_concluded,
       e.show_reaction_test,
       e.show_selfie,
       e.show_vote_trend,
       e.show_feedback_survey,
       e.show_pre_vote_sponsors,
       e.show_pre_vote_bottom_sponsors,
       e.show_vote_counter,
       e.feedback_survey_config,
       IFNULL(t1.name, ''),
       IFNULL(t2.name, '')
FROM events e
LEFT JOIN teams t1 ON t1.id = e.team1_id
LEFT JOIN teams t2 ON t2.id = e.team2_id
WHERE e.organization_id = ?
ORDER BY e.start_datetime DESC`, organizationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var es []Event
	for rows.Next() {
		var e Event
		var isActive int
		var votesClosed int
		var isConcluded int
		var showReaction int
		var showSelfie int
		var showVoteTrend int
		var showFeedback int
		var showPreVoteSponsors int
		var showPreVoteBottomSponsors int
		var showVoteCounter int
		var surveyConfig sql.NullString
		if err := rows.Scan(&e.ID, &e.OrganizationID, &e.Team1ID, &e.Team2ID, &e.StartDateTime, &e.Location, &isActive, &votesClosed, &isConcluded, &showReaction, &showSelfie, &showVoteTrend, &showFeedback, &showPreVoteSponsors, &showPreVoteBottomSponsors, &showVoteCounter, &surveyConfig, &e.Team1Name, &e.Team2Name); err != nil {
			return nil, err
		}
		e.IsActive = isActive == 1
		e.VotesClosed = votesClosed == 1
		e.IsConcluded = isConcluded == 1
		e.ShowReactionTest = showReaction == 1
		e.ShowSelfie = showSelfie == 1
		e.ShowVoteTrend = showVoteTrend == 1
		e.ShowFeedbackSurvey = showFeedback == 1
		e.ShowPreVoteSponsors = showPreVoteSponsors == 1
		e.ShowPreVoteBottomSponsors = showPreVoteBottomSponsors == 1
		e.ShowVoteCounter = showVoteCounter == 1
		cfg := decodeEventFeedbackSurveyConfig(surveyConfig)
		e.FeedbackSurvey = &cfg
		es = append(es, e)
	}
	for i := range es {
		prizes, err := db.ListEventPrizes(es[i].ID)
		if err != nil {
			return nil, err
		}
		es[i].Prizes = prizes
	}

	return es, nil
}

func (db *appdbimpl) UpdateEvent(e Event) error {
	tx, err := db.c.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	survey := NormalizeEventFeedbackSurveyConfig(e.FeedbackSurvey)
	e.FeedbackSurvey = &survey
	surveyJSON := encodeEventFeedbackSurveyConfig(survey)

	if _, err := tx.Exec(`UPDATE events SET team1_id=?, team2_id=?, start_datetime=?, location=?, show_reaction_test=?, show_selfie=?, show_vote_trend=?, show_feedback_survey=?, show_pre_vote_sponsors=?, show_pre_vote_bottom_sponsors=?, show_vote_counter=?, feedback_survey_config=? WHERE id=?`, e.Team1ID, e.Team2ID, e.StartDateTime, e.Location, boolToInt(e.ShowReactionTest), boolToInt(e.ShowSelfie), boolToInt(e.ShowVoteTrend), boolToInt(e.ShowFeedbackSurvey), boolToInt(e.ShowPreVoteSponsors), boolToInt(e.ShowPreVoteBottomSponsors), boolToInt(e.ShowVoteCounter), surveyJSON, e.ID); err != nil {
		return err
	}

	if err := db.syncEventPrizesTx(tx, e.ID, e.Prizes); err != nil {
		return err
	}

	return tx.Commit()
}

func (db *appdbimpl) GetEventTeamIDs(eventID int) (int, int, error) {
	var team1, team2 int
	if err := db.c.QueryRow(`SELECT team1_id, team2_id FROM events WHERE id = ?`, eventID).Scan(&team1, &team2); err != nil {
		return 0, 0, err
	}
	return team1, team2, nil
}

func (db *appdbimpl) GetEventOrganizationID(eventID int) (int, error) {
	var organizationID int
	if err := db.c.QueryRow(`SELECT organization_id FROM events WHERE id = ?`, eventID).Scan(&organizationID); err != nil {
		return 0, err
	}
	return organizationID, nil
}

func (db *appdbimpl) DeleteEvent(id int) error {
	return db.PurgeEventData(id)
}

func (db *appdbimpl) PurgeEventData(eventID int) error {
	if eventID <= 0 {
		return sql.ErrNoRows
	}

	tx, err := db.c.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`UPDATE event_prizes SET winner_vote_id = NULL, winner_assigned_at = NULL WHERE event_id = ?`, eventID); err != nil {
		return err
	}

	if _, err := tx.Exec(`DELETE FROM sponsor_clicks WHERE event_id = ?`, eventID); err != nil {
		return err
	}

	if _, err := tx.Exec(`DELETE FROM sponsor_exposures WHERE event_id = ?`, eventID); err != nil {
		return err
	}

	if _, err := tx.Exec(`DELETE FROM sponsor_sessions WHERE event_id = ?`, eventID); err != nil {
		return err
	}

	if _, err := tx.Exec(`DELETE FROM contact_events WHERE event_id = ?`, eventID); err != nil {
		return err
	}

	if _, err := tx.Exec(`DELETE FROM contacts WHERE event_id = ?`, eventID); err != nil {
		return err
	}

	if _, err := tx.Exec(`DELETE FROM votes WHERE event_id = ?`, eventID); err != nil {
		return err
	}

	if _, err := tx.Exec(`DELETE FROM selfies WHERE event_id = ?`, eventID); err != nil {
		return err
	}

	if _, err := tx.Exec(`DELETE FROM tickets WHERE event_id = ?`, eventID); err != nil {
		return err
	}

	if _, err := tx.Exec(`DELETE FROM event_prizes WHERE event_id = ?`, eventID); err != nil {
		return err
	}

	res, err := tx.Exec(`DELETE FROM events WHERE id = ?`, eventID)
	if err != nil {
		return err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return sql.ErrNoRows
	}

	return tx.Commit()
}

func (db *appdbimpl) SetActiveEvent(eventID int, organizationID int) error {
	tx, err := db.c.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`UPDATE events SET is_active = 0 WHERE organization_id = ?`, organizationID); err != nil {
		return err
	}

	res, err := tx.Exec(`UPDATE events SET is_active = 1, votes_closed = 0 WHERE id = ? AND organization_id = ? AND is_concluded = 0`, eventID, organizationID)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		var concluded int
		err := tx.QueryRow(`SELECT is_concluded FROM events WHERE id = ?`, eventID).Scan(&concluded)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return sql.ErrNoRows
			}
			return err
		}
		if concluded == 1 {
			return ErrEventAlreadyConcluded
		}
		return sql.ErrNoRows
	}

	return tx.Commit()
}

func (db *appdbimpl) ClearActiveEvent(organizationID int) error {
	_, err := db.c.Exec(`UPDATE events SET is_active = 0 WHERE organization_id = ?`, organizationID)
	return err
}

func (db *appdbimpl) CloseEventVoting(eventID int) error {
	res, err := db.c.Exec(`UPDATE events SET votes_closed = 1 WHERE id = ? AND is_active = 1 AND is_concluded = 0`, eventID)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (db *appdbimpl) ConcludeEvent(eventID int) error {
	tx, err := db.c.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var concluded int
	if err := tx.QueryRow(`SELECT is_concluded FROM events WHERE id = ?`, eventID).Scan(&concluded); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return sql.ErrNoRows
		}
		return err
	}
	if concluded == 1 {
		return ErrEventAlreadyConcluded
	}

	if _, err := tx.Exec(`UPDATE events SET is_active = 0, votes_closed = 1, is_concluded = 1 WHERE id = ?`, eventID); err != nil {
		return err
	}

	return tx.Commit()
}

func (db *appdbimpl) GetActiveEvent(organizationID int) (Event, error) {
	var e Event
	var isActive int
	var votesClosed int
	var isConcluded int
	var showReaction int
	var showSelfie int
	var showVoteTrend int
	var showFeedback int
	var showPreVoteSponsors int
	var showPreVoteBottomSponsors int
	var showVoteCounter int
	var surveyConfig sql.NullString
	err := db.c.QueryRow(`
SELECT e.id,
       e.organization_id,
       e.team1_id,
       e.team2_id,
       e.start_datetime,
       e.location,
       e.is_active,
       e.votes_closed,
       e.is_concluded,
       e.show_reaction_test,
       e.show_selfie,
       e.show_vote_trend,
       e.show_feedback_survey,
       e.show_pre_vote_sponsors,
       e.show_pre_vote_bottom_sponsors,
       e.show_vote_counter,
       e.feedback_survey_config,
       IFNULL(o.name, ''),
       IFNULL(o.logo_url, ''),
       IFNULL(t1.name, ''),
       IFNULL(t2.name, '')
FROM events e
LEFT JOIN organizations o ON o.id = e.organization_id
LEFT JOIN teams t1 ON t1.id = e.team1_id
LEFT JOIN teams t2 ON t2.id = e.team2_id
WHERE e.is_active = 1 AND e.organization_id = ?
LIMIT 1
`, organizationID).Scan(&e.ID, &e.OrganizationID, &e.Team1ID, &e.Team2ID, &e.StartDateTime, &e.Location, &isActive, &votesClosed, &isConcluded, &showReaction, &showSelfie, &showVoteTrend, &showFeedback, &showPreVoteSponsors, &showPreVoteBottomSponsors, &showVoteCounter, &surveyConfig, &e.OrganizationName, &e.OrganizationLogoURL, &e.Team1Name, &e.Team2Name)
	if err != nil {
		return Event{}, err
	}
	e.IsActive = isActive == 1
	e.VotesClosed = votesClosed == 1
	e.IsConcluded = isConcluded == 1
	e.ShowReactionTest = showReaction == 1
	e.ShowSelfie = showSelfie == 1
	e.ShowVoteTrend = showVoteTrend == 1
	e.ShowFeedbackSurvey = showFeedback == 1
	e.ShowPreVoteSponsors = showPreVoteSponsors == 1
	e.ShowPreVoteBottomSponsors = showPreVoteBottomSponsors == 1
	e.ShowVoteCounter = showVoteCounter == 1
	cfg := decodeEventFeedbackSurveyConfig(surveyConfig)
	e.FeedbackSurvey = &cfg
	return e, nil
}

// Votes listing and deletion
func (db *appdbimpl) ListVotes() ([]Vote, error) {
	rows, err := db.c.Query(`SELECT id, event_id, player_id, ticket_code, ticket_signature, device_id, created_at FROM votes`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var vs []Vote
	for rows.Next() {
		var v Vote
		if err := rows.Scan(&v.ID, &v.EventID, &v.PlayerID, &v.TicketCode, &v.TicketSignature, &v.DeviceID, &v.CreatedAt); err != nil {
			return nil, err
		}
		vs = append(vs, v)
	}
	return vs, nil
}

func (db *appdbimpl) GetVoteEventID(voteID int) (int, error) {
	var eventID int
	if err := db.c.QueryRow(`SELECT event_id FROM votes WHERE id = ?`, voteID).Scan(&eventID); err != nil {
		return 0, err
	}
	return eventID, nil
}

func (db *appdbimpl) ListVotesByOrganization(organizationID int) ([]Vote, error) {
	rows, err := db.c.Query(`
SELECT v.id, v.event_id, v.player_id, v.ticket_code, v.ticket_signature, v.device_id, v.created_at
FROM votes v
INNER JOIN events e ON e.id = v.event_id
WHERE e.organization_id = ?`, organizationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var vs []Vote
	for rows.Next() {
		var v Vote
		if err := rows.Scan(&v.ID, &v.EventID, &v.PlayerID, &v.TicketCode, &v.TicketSignature, &v.DeviceID, &v.CreatedAt); err != nil {
			return nil, err
		}
		vs = append(vs, v)
	}
	return vs, nil
}

func (db *appdbimpl) ListEventTickets(eventID int) ([]EventTicket, error) {
	rows, err := db.c.Query(`
SELECT v.id, v.ticket_code, v.ticket_signature, v.player_id, IFNULL(p.first_name, ''), IFNULL(p.last_name, ''), v.created_at
FROM votes v
LEFT JOIN players p ON p.id = v.player_id
LEFT JOIN event_prizes ep ON ep.winner_vote_id = v.id AND ep.event_id = ?
WHERE v.event_id = ? AND ep.id IS NULL
ORDER BY v.created_at ASC
`, eventID, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tickets []EventTicket
	for rows.Next() {
		var t EventTicket
		if err := rows.Scan(&t.VoteID, &t.TicketCode, &t.TicketSignature, &t.PlayerID, &t.PlayerFirstName, &t.PlayerLastName, &t.CreatedAt); err != nil {
			return nil, err
		}
		tickets = append(tickets, t)
	}
	return tickets, nil
}

func (db *appdbimpl) ValidateTicket(eventID int, code string) (TicketValidationResult, error) {
	var result TicketValidationResult

	row := db.c.QueryRow(`
SELECT v.id,
       v.event_id,
       v.player_id,
       v.ticket_code,
       v.ticket_signature,
       IFNULL(p.first_name, ''),
       IFNULL(p.last_name, ''),
       v.created_at,
       ep.id,
       IFNULL(ep.name, ''),
       IFNULL(ep.position, 0)
FROM votes v
LEFT JOIN players p ON p.id = v.player_id
LEFT JOIN event_prizes ep ON ep.winner_vote_id = v.id AND ep.event_id = v.event_id
WHERE v.event_id = ? AND v.ticket_code = ?
LIMIT 1
`, eventID, code)

	var prizeID sql.NullInt64
	var prizeName sql.NullString
	var prizePosition sql.NullInt64

	if err := row.Scan(
		&result.VoteID,
		&result.EventID,
		&result.PlayerID,
		&result.TicketCode,
		&result.TicketSignature,
		&result.PlayerFirstName,
		&result.PlayerLastName,
		&result.CreatedAt,
		&prizeID,
		&prizeName,
		&prizePosition,
	); err != nil {
		return TicketValidationResult{}, err
	}

	if prizeID.Valid {
		result.AssignedPrize = &TicketValidationPrize{
			ID:       int(prizeID.Int64),
			Name:     prizeName.String,
			Position: int(prizePosition.Int64),
		}
	}

	return result, nil
}

func (db *appdbimpl) RedeemTicket(eventID int, code, signature string) (bool, error) {
	var storedSignature sql.NullString
	var redeemedAt sql.NullString

	err := db.c.QueryRow(`SELECT signature, redeemed_at FROM tickets WHERE event_id = ? AND code = ? LIMIT 1`, eventID, code).Scan(&storedSignature, &redeemedAt)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		_, err = db.c.Exec(`INSERT INTO tickets (event_id, code, signature, redeemed_at) VALUES (?, ?, ?, CURRENT_TIMESTAMP)`, eventID, code, signature)
		if err != nil {
			if isTicketUniqueConstraintError(err) {
				return db.RedeemTicket(eventID, code, signature)
			}
			return false, err
		}
		return false, nil
	case err != nil:
		return false, err
	}

	if storedSignature.Valid && storedSignature.String != "" && !strings.EqualFold(storedSignature.String, signature) {
		return true, ErrTicketSignatureMismatch
	}

	if !redeemedAt.Valid || strings.TrimSpace(redeemedAt.String) == "" {
		_, err = db.c.Exec(`UPDATE tickets SET signature = ?, redeemed_at = CURRENT_TIMESTAMP WHERE event_id = ? AND code = ?`, signature, eventID, code)
		if err != nil {
			return false, err
		}
		return false, nil
	}

	return true, nil
}

func isTicketUniqueConstraintError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed")
}

func (db *appdbimpl) ListEventPrizes(eventID int) ([]EventPrize, error) {
	rows, err := db.c.Query(`
SELECT p.id,
       p.event_id,
       p.name,
        p.position,
       p.winner_vote_id,
       IFNULL(p.winner_assigned_at, ''),
       IFNULL(v.ticket_code, ''),
       IFNULL(v.player_id, 0),
       IFNULL(pl.first_name, ''),
       IFNULL(pl.last_name, '')
FROM event_prizes p
LEFT JOIN votes v ON v.id = p.winner_vote_id
LEFT JOIN players pl ON pl.id = v.player_id
WHERE p.event_id = ?
ORDER BY p.position ASC, p.id ASC
`, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prizes []EventPrize
	for rows.Next() {
		var p EventPrize
		var winnerID sql.NullInt64
		var assignedAt sql.NullString
		var ticketCode string
		var playerID int
		var playerFirstName string
		var playerLastName string
		if err := rows.Scan(&p.ID, &p.EventID, &p.Name, &p.Position, &winnerID, &assignedAt, &ticketCode, &playerID, &playerFirstName, &playerLastName); err != nil {
			return nil, err
		}
		if winnerID.Valid {
			p.Winner = &EventPrizeWinner{
				VoteID:          int(winnerID.Int64),
				TicketCode:      ticketCode,
				PlayerID:        playerID,
				PlayerFirstName: playerFirstName,
				PlayerLastName:  playerLastName,
				AssignedAt:      assignedAt.String,
			}
		}
		prizes = append(prizes, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return prizes, nil
}

func (db *appdbimpl) getEventPrize(prizeID int) (EventPrize, error) {
	var p EventPrize
	var winnerID sql.NullInt64
	var assignedAt sql.NullString
	var ticketCode string
	var playerID int
	var playerFirstName string
	var playerLastName string

	err := db.c.QueryRow(`
SELECT p.id,
       p.event_id,
       p.name,
       p.position,
       p.winner_vote_id,
       IFNULL(p.winner_assigned_at, ''),
       IFNULL(v.ticket_code, ''),
       IFNULL(v.player_id, 0),
       IFNULL(pl.first_name, ''),
       IFNULL(pl.last_name, '')
FROM event_prizes p
LEFT JOIN votes v ON v.id = p.winner_vote_id
LEFT JOIN players pl ON pl.id = v.player_id
WHERE p.id = ?
`, prizeID).Scan(&p.ID, &p.EventID, &p.Name, &p.Position, &winnerID, &assignedAt, &ticketCode, &playerID, &playerFirstName, &playerLastName)
	if err != nil {
		return EventPrize{}, err
	}
	if winnerID.Valid {
		p.Winner = &EventPrizeWinner{
			VoteID:          int(winnerID.Int64),
			TicketCode:      ticketCode,
			PlayerID:        playerID,
			PlayerFirstName: playerFirstName,
			PlayerLastName:  playerLastName,
			AssignedAt:      assignedAt.String,
		}
	}
	return p, nil
}

func (db *appdbimpl) AssignPrizeWinner(eventID, prizeID, voteID int) (EventPrize, error) {
	tx, err := db.c.Begin()
	if err != nil {
		return EventPrize{}, err
	}
	defer tx.Rollback()

	var prizeEventID int
	var winnerID sql.NullInt64
	if err := tx.QueryRow(`SELECT event_id, winner_vote_id FROM event_prizes WHERE id = ?`, prizeID).Scan(&prizeEventID, &winnerID); err != nil {
		return EventPrize{}, err
	}
	if prizeEventID != eventID {
		return EventPrize{}, sql.ErrNoRows
	}
	if winnerID.Valid && winnerID.Int64 > 0 {
		return EventPrize{}, ErrPrizeAlreadyAssigned
	}

	var voteEventID int
	if err := tx.QueryRow(`SELECT event_id FROM votes WHERE id = ?`, voteID).Scan(&voteEventID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return EventPrize{}, ErrPrizeVoteMismatch
		}
		return EventPrize{}, err
	}
	if voteEventID != eventID {
		return EventPrize{}, ErrPrizeVoteMismatch
	}

	var alreadyAssigned int
	if err := tx.QueryRow(`SELECT COUNT(1) FROM event_prizes WHERE event_id = ? AND winner_vote_id = ?`, eventID, voteID).Scan(&alreadyAssigned); err != nil {
		return EventPrize{}, err
	}
	if alreadyAssigned > 0 {
		return EventPrize{}, ErrPrizeWinnerConflict
	}

	if _, err := tx.Exec(`UPDATE event_prizes SET winner_vote_id = ?, winner_assigned_at = CURRENT_TIMESTAMP WHERE id = ?`, voteID, prizeID); err != nil {
		return EventPrize{}, err
	}

	if err := tx.Commit(); err != nil {
		return EventPrize{}, err
	}

	return db.getEventPrize(prizeID)
}

func (db *appdbimpl) ClearPrizeWinner(eventID, prizeID int) error {
	res, err := db.c.Exec(`UPDATE event_prizes SET winner_vote_id = NULL, winner_assigned_at = NULL WHERE id = ? AND event_id = ?`, prizeID, eventID)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// GetEventResults returns aggregated vote results for an event
func (db *appdbimpl) GetEventResults(eventID int) ([]EventVoteResult, error) {
	var exists int
	if err := db.c.QueryRow(`SELECT COUNT(1) FROM events WHERE id = ?`, eventID).Scan(&exists); err != nil {
		return nil, err
	}
	if exists == 0 {
		return nil, sql.ErrNoRows
	}

	rows, err := db.c.Query(`
SELECT player_id, COUNT(*) AS votes, IFNULL(MAX(created_at), '') AS last_vote_at
FROM votes
WHERE event_id = ?
GROUP BY player_id
ORDER BY votes DESC, last_vote_at ASC, player_id ASC
`, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []EventVoteResult
	for rows.Next() {
		var res EventVoteResult
		if err := rows.Scan(&res.PlayerID, &res.Votes, &res.LastVoteAt); err != nil {
			return nil, err
		}
		results = append(results, res)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

func (db *appdbimpl) DeleteVote(id int) error {
	_, err := db.c.Exec(`DELETE FROM votes WHERE id=?`, id)
	return err
}

// Admin operations
func (db *appdbimpl) CreateAdmin(a Admin) (int, error) {
	res, err := db.c.Exec(`INSERT INTO admins (username, password_hash, role, organization_id) VALUES (?, ?, ?, ?)`, a.Username, a.PasswordHash, a.Role, nullableOrgID(a.OrganizationID))
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	return int(id), nil
}

func (db *appdbimpl) ListAdmins(organizationID int) ([]Admin, error) {
	query := `SELECT id, username, password_hash, role, created_at, IFNULL(organization_id, 0) FROM admins`
	var args []interface{}
	if organizationID > 0 {
		query += ` WHERE organization_id = ?`
		args = append(args, organizationID)
	}

	rows, err := db.c.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var as []Admin
	for rows.Next() {
		var a Admin
		if err := rows.Scan(&a.ID, &a.Username, &a.PasswordHash, &a.Role, &a.CreatedAt, &a.OrganizationID); err != nil {
			return nil, err
		}
		as = append(as, a)
	}
	return as, nil
}

func (db *appdbimpl) ListPartners(organizationID int) ([]Admin, error) {
	query := `SELECT id, username, password_hash, role, created_at, IFNULL(organization_id, 0) FROM admins WHERE LOWER(role) = 'partner'`
	var args []interface{}
	if organizationID > 0 {
		query += ` AND organization_id = ?`
		args = append(args, organizationID)
	}

	rows, err := db.c.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var partners []Admin
	for rows.Next() {
		var a Admin
		if err := rows.Scan(&a.ID, &a.Username, &a.PasswordHash, &a.Role, &a.CreatedAt, &a.OrganizationID); err != nil {
			return nil, err
		}
		partners = append(partners, a)
	}
	return partners, nil
}

func (db *appdbimpl) UpdateAdmin(a Admin) error {
	_, err := db.c.Exec(`UPDATE admins SET username=?, password_hash=?, role=?, organization_id=? WHERE id=?`, a.Username, a.PasswordHash, a.Role, nullableOrgID(a.OrganizationID), a.ID)
	return err
}

func (db *appdbimpl) DeleteAdmin(id int) error {
	_, err := db.c.Exec(`DELETE FROM admins WHERE id=?`, id)
	return err
}

func (db *appdbimpl) GetAdminByUsername(username string, organizationID int) (Admin, error) {
	var admin Admin
	query := `SELECT id, username, password_hash, role, created_at, IFNULL(organization_id, 0) FROM admins WHERE username = ?`
	args := []interface{}{username}
	if organizationID > 0 {
		query += ` AND organization_id = ?`
		args = append(args, organizationID)
	} else {
		query += ` AND (organization_id IS NULL OR organization_id = 0)`
	}

	err := db.c.QueryRow(query, args...).Scan(&admin.ID, &admin.Username, &admin.PasswordHash, &admin.Role, &admin.CreatedAt, &admin.OrganizationID)
	if err != nil {
		return Admin{}, err
	}
	return admin, nil
}

func (db *appdbimpl) GetAdminByID(id int) (Admin, error) {
	var admin Admin
	err := db.c.QueryRow(`SELECT id, username, password_hash, role, created_at, IFNULL(organization_id, 0) FROM admins WHERE id = ?`, id).Scan(&admin.ID, &admin.Username, &admin.PasswordHash, &admin.Role, &admin.CreatedAt, &admin.OrganizationID)
	if err != nil {
		return Admin{}, err
	}
	return admin, nil
}

func (db *appdbimpl) scanOrganization(scanner rowScanner) (Organization, error) {
	var org Organization
	var isActive int
	var rosterSchema int
	var teamID sql.NullInt64
	if err := scanner.Scan(&org.ID, &org.Name, &org.Slug, &org.City, &org.LogoURL, &isActive, &rosterSchema, &teamID, &org.CreatedAt, &org.UpdatedAt); err != nil {
		return Organization{}, err
	}
	org.IsActive = isActive != 0
	org.RosterSchema = normalizeRosterSchema(rosterSchema)
	if teamID.Valid {
		org.TeamID = int(teamID.Int64)
	}
	return org, nil
}

func (db *appdbimpl) ListOrganizations() ([]Organization, error) {
	rows, err := db.c.Query(`SELECT id, name, slug, city, logo_url, is_active, roster_schema, IFNULL(team_id, 0), created_at, updated_at FROM organizations ORDER BY name COLLATE NOCASE ASC, id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orgs []Organization
	for rows.Next() {
		org, err := db.scanOrganization(rows)
		if err != nil {
			return nil, err
		}
		orgs = append(orgs, org)
	}
	return orgs, rows.Err()
}

func (db *appdbimpl) GetOrganization(id int) (Organization, error) {
	if id <= 0 {
		return Organization{}, sql.ErrNoRows
	}
	row := db.c.QueryRow(`SELECT id, name, slug, city, logo_url, is_active, roster_schema, IFNULL(team_id, 0), created_at, updated_at FROM organizations WHERE id = ?`, id)
	return db.scanOrganization(row)
}

func (db *appdbimpl) GetOrganizationBySlug(slug string) (Organization, error) {
	if slug == "" {
		return Organization{}, sql.ErrNoRows
	}
	row := db.c.QueryRow(`SELECT id, name, slug, city, logo_url, is_active, roster_schema, IFNULL(team_id, 0), created_at, updated_at FROM organizations WHERE slug = ?`, normalizeSlug(slug))
	return db.scanOrganization(row)
}

func (db *appdbimpl) CreateOrganization(org Organization) (Organization, error) {
	sanitizedName := strings.TrimSpace(org.Name)
	if sanitizedName == "" {
		return Organization{}, ErrInvalidOrganizationData
	}
	sanitizedCity := strings.TrimSpace(org.City)
	sanitizedLogo := strings.TrimSpace(org.LogoURL)
	sanitizedSlug := normalizeSlug(org.Slug)
	if sanitizedSlug == "" {
		sanitizedSlug = normalizeSlug(sanitizedName)
	}
	rosterSchema := normalizeRosterSchema(org.RosterSchema)
	if err := db.ensureOrganizationSlugAvailable(sanitizedSlug, 0); err != nil {
		return Organization{}, err
	}

	tx, err := db.c.Begin()
	if err != nil {
		return Organization{}, err
	}
	defer func() { _ = tx.Rollback() }()

	teamID := org.TeamID
	if teamID <= 0 {
		res, err := tx.Exec(`INSERT INTO teams (name) VALUES (?)`, sanitizedName)
		if err != nil {
			return Organization{}, err
		}
		newTeamID, _ := res.LastInsertId()
		teamID = int(newTeamID)
	}

	res, err := tx.Exec(`INSERT INTO organizations (name, slug, city, logo_url, is_active, roster_schema, team_id) VALUES (?, ?, ?, ?, ?, ?, ?)`, sanitizedName, sanitizedSlug, sanitizedCity, sanitizedLogo, boolToInt(org.IsActive), rosterSchema, teamID)
	if err != nil {
		return Organization{}, err
	}
	insertedID, _ := res.LastInsertId()

	if err := createDefaultStaffAdmin(tx, int(insertedID)); err != nil {
		return Organization{}, err
	}

	if err := tx.Commit(); err != nil {
		return Organization{}, err
	}

	created, err := db.GetOrganization(int(insertedID))
	if err != nil {
		return Organization{}, err
	}
	return created, nil
}

func (db *appdbimpl) UpdateOrganization(org Organization) (Organization, error) {
	if org.ID <= 0 {
		return Organization{}, sql.ErrNoRows
	}
	existing, err := db.GetOrganization(org.ID)
	if err != nil {
		return Organization{}, err
	}

	sanitizedName := strings.TrimSpace(org.Name)
	if sanitizedName == "" {
		return Organization{}, ErrInvalidOrganizationData
	}
	sanitizedCity := strings.TrimSpace(org.City)
	sanitizedLogo := strings.TrimSpace(org.LogoURL)
	isActive := org.IsActive
	teamID := org.TeamID
	if teamID == 0 {
		teamID = existing.TeamID
	}
	sanitizedSlug := normalizeSlug(org.Slug)
	if sanitizedSlug == "" {
		sanitizedSlug = normalizeSlug(sanitizedName)
	}
	if err := db.ensureOrganizationSlugAvailable(sanitizedSlug, org.ID); err != nil {
		return Organization{}, err
	}
	rosterSchema := normalizeRosterSchema(org.RosterSchema)
	if org.RosterSchema == 0 {
		rosterSchema = existing.RosterSchema
	}

	tx, err := db.c.Begin()
	if err != nil {
		return Organization{}, err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.Exec(`UPDATE organizations SET name=?, slug=?, city=?, logo_url=?, is_active=?, roster_schema=?, team_id=? WHERE id=?`, sanitizedName, sanitizedSlug, sanitizedCity, sanitizedLogo, boolToInt(isActive), rosterSchema, teamID, org.ID); err != nil {
		return Organization{}, err
	}

	if teamID > 0 {
		if _, err := tx.Exec(`UPDATE teams SET name=? WHERE id=?`, sanitizedName, teamID); err != nil {
			return Organization{}, err
		}
	}

	if err := tx.Commit(); err != nil {
		return Organization{}, err
	}

	updated, err := db.GetOrganization(org.ID)
	if err != nil {
		return Organization{}, err
	}
	return updated, nil
}

func (db *appdbimpl) UpdateOrganizationRosterSchema(organizationID int, rosterSchema int) error {
	if organizationID <= 0 {
		return sql.ErrNoRows
	}
	validSchema := normalizeRosterSchema(rosterSchema)
	_, err := db.c.Exec(`UPDATE organizations SET roster_schema = ? WHERE id = ?`, validSchema, organizationID)
	return err
}

func (db *appdbimpl) GetOrganizationRosterSchema(organizationID int) (int, error) {
	org, err := db.GetOrganization(organizationID)
	if err != nil {
		return 0, err
	}
	return normalizeRosterSchema(org.RosterSchema), nil
}

func (db *appdbimpl) GetOrganizationStats(id int) (OrganizationStats, error) {
	org, err := db.GetOrganization(id)
	if err != nil {
		return OrganizationStats{}, err
	}

	stats := OrganizationStats{OrganizationID: id}
	if org.TeamID == 0 {
		return stats, nil
	}

	if err := db.c.QueryRow(`SELECT COUNT(v.id) FROM votes v JOIN events e ON e.id = v.event_id WHERE e.team1_id = ? OR e.team2_id = ?`, org.TeamID, org.TeamID).Scan(&stats.TotalVotes); err != nil {
		return stats, err
	}
	if err := db.c.QueryRow(`SELECT COUNT(*) FROM events WHERE team1_id = ? OR team2_id = ?`, org.TeamID, org.TeamID).Scan(&stats.TotalMatches); err != nil {
		return stats, err
	}

	var lastEventID sql.NullInt64
	var lastEventDate sql.NullString
	err = db.c.QueryRow(`SELECT id, IFNULL(start_datetime, '') FROM events WHERE team1_id = ? OR team2_id = ? ORDER BY datetime(start_datetime) DESC, id DESC LIMIT 1`, org.TeamID, org.TeamID).Scan(&lastEventID, &lastEventDate)
	if errors.Is(err, sql.ErrNoRows) {
		return stats, nil
	}
	if err != nil {
		return stats, err
	}
	if lastEventDate.Valid {
		stats.LastMatchDate = lastEventDate.String
	}
	if lastEventID.Valid {
		if err := db.c.QueryRow(`SELECT COUNT(*) FROM votes WHERE event_id = ?`, lastEventID.Int64).Scan(&stats.LastMatchVotes); err != nil {
			return stats, err
		}
	}
	return stats, nil
}

func (db *appdbimpl) GetMasterDashboardSummary() (MasterDashboardSummary, error) {
	var summary MasterDashboardSummary
	if err := db.c.QueryRow(`SELECT COUNT(*) FROM organizations`).Scan(&summary.TotalOrganizations); err != nil {
		return summary, err
	}
	if err := db.c.QueryRow(`SELECT COUNT(*) FROM votes`).Scan(&summary.TotalVotes); err != nil {
		return summary, err
	}
	if err := db.c.QueryRow(`SELECT COUNT(*) FROM votes WHERE datetime(created_at) >= datetime('now', '-7 days')`).Scan(&summary.VotesLast7Days); err != nil {
		return summary, err
	}
	if err := db.c.QueryRow(`SELECT COUNT(*) FROM events`).Scan(&summary.TotalEvents); err != nil {
		return summary, err
	}
	return summary, nil
}

func (db *appdbimpl) GetMasterAnalytics() (MasterAnalytics, error) {
	analytics := MasterAnalytics{
		VoteTrends:   VoteTrendAnalytics{Global: []VoteTrendPoint{}, PerOrganization: []OrganizationVoteTrend{}},
		SponsorStats: SponsorMasterStats{Organizations: []SponsorOrganizationStat{}},
		TopEvents:    TopEventsAnalytics{AllTime: []TopEventEntry{}, Last7Days: []TopEventEntry{}},
	}

	orgs, err := db.ListOrganizations()
	if err != nil {
		return analytics, err
	}

	orgInfo := make(map[int]Organization)
	for _, org := range orgs {
		orgInfo[org.ID] = org
	}

	totalVotesByOrg := make(map[int]int)
	votesLastWeek := make(map[int]int)
	votesPrevWeek := make(map[int]int)
	eventsByOrg := make(map[int]int)

	rows, err := db.c.Query(`SELECT e.organization_id, COUNT(v.id) FROM votes v INNER JOIN events e ON e.id = v.event_id WHERE e.organization_id > 0 GROUP BY e.organization_id`)
	if err != nil {
		return analytics, err
	}
	for rows.Next() {
		var orgID, count int
		if err := rows.Scan(&orgID, &count); err != nil {
			rows.Close()
			return analytics, err
		}
		totalVotesByOrg[orgID] = count
	}
	rows.Close()

	rows, err = db.c.Query(`SELECT e.organization_id, COUNT(v.id) FROM votes v INNER JOIN events e ON e.id = v.event_id WHERE e.organization_id > 0 AND datetime(v.created_at) >= datetime('now', '-7 days') GROUP BY e.organization_id`)
	if err != nil {
		return analytics, err
	}
	for rows.Next() {
		var orgID, count int
		if err := rows.Scan(&orgID, &count); err != nil {
			rows.Close()
			return analytics, err
		}
		votesLastWeek[orgID] = count
	}
	rows.Close()

	rows, err = db.c.Query(`SELECT e.organization_id, COUNT(v.id) FROM votes v INNER JOIN events e ON e.id = v.event_id WHERE e.organization_id > 0 AND datetime(v.created_at) >= datetime('now', '-14 days') AND datetime(v.created_at) < datetime('now', '-7 days') GROUP BY e.organization_id`)
	if err != nil {
		return analytics, err
	}
	for rows.Next() {
		var orgID, count int
		if err := rows.Scan(&orgID, &count); err != nil {
			rows.Close()
			return analytics, err
		}
		votesPrevWeek[orgID] = count
	}
	rows.Close()

	rows, err = db.c.Query(`SELECT organization_id, COUNT(*) FROM events WHERE organization_id > 0 GROUP BY organization_id`)
	if err != nil {
		return analytics, err
	}
	for rows.Next() {
		var orgID, count int
		if err := rows.Scan(&orgID, &count); err != nil {
			rows.Close()
			return analytics, err
		}
		eventsByOrg[orgID] = count
	}
	rows.Close()

	for orgID, org := range orgInfo {
		if orgID <= 0 {
			continue
		}
		current := votesLastWeek[orgID]
		previous := votesPrevWeek[orgID]
		growth := 0.0
		if previous > 0 {
			growth = (float64(current-previous) / float64(previous)) * 100
		} else if current > 0 {
			growth = 100
		}
		analytics.OrganizationLeaderboard = append(analytics.OrganizationLeaderboard, OrganizationLeaderboardEntry{
			OrganizationID:   orgID,
			Name:             org.Name,
			Slug:             org.Slug,
			City:             org.City,
			LogoURL:          org.LogoURL,
			TotalVotes:       totalVotesByOrg[orgID],
			VotesLast7Days:   current,
			TotalEvents:      eventsByOrg[orgID],
			GrowthPercentage: growth,
		})
	}

	globalTrendRows, err := db.c.Query(`SELECT date(created_at), COUNT(*) FROM votes WHERE date(created_at) >= date('now', '-29 days') GROUP BY date(created_at) ORDER BY date(created_at)`)
	if err != nil {
		return analytics, err
	}
	for globalTrendRows.Next() {
		var date string
		var count int
		if err := globalTrendRows.Scan(&date, &count); err != nil {
			globalTrendRows.Close()
			return analytics, err
		}
		analytics.VoteTrends.Global = append(analytics.VoteTrends.Global, VoteTrendPoint{Date: date, Votes: count})
	}
	globalTrendRows.Close()

	orgTrendRows, err := db.c.Query(`SELECT e.organization_id, IFNULL(o.name, ''), IFNULL(o.slug, ''), date(v.created_at), COUNT(*) FROM votes v INNER JOIN events e ON e.id = v.event_id LEFT JOIN organizations o ON o.id = e.organization_id WHERE e.organization_id > 0 AND date(v.created_at) >= date('now', '-29 days') GROUP BY e.organization_id, date(v.created_at) ORDER BY e.organization_id, date(v.created_at)`)
	if err != nil {
		return analytics, err
	}
	trendsByOrg := make(map[int]*OrganizationVoteTrend)
	for orgTrendRows.Next() {
		var orgID int
		var name, slug, date string
		var count int
		if err := orgTrendRows.Scan(&orgID, &name, &slug, &date, &count); err != nil {
			orgTrendRows.Close()
			return analytics, err
		}
		trend := trendsByOrg[orgID]
		if trend == nil {
			trend = &OrganizationVoteTrend{OrganizationID: orgID, Name: name, Slug: slug, Data: []VoteTrendPoint{}}
			trendsByOrg[orgID] = trend
		}
		trend.Data = append(trend.Data, VoteTrendPoint{Date: date, Votes: count})
	}
	orgTrendRows.Close()
	for _, trend := range trendsByOrg {
		analytics.VoteTrends.PerOrganization = append(analytics.VoteTrends.PerOrganization, *trend)
	}

	analytics.TopEvents.AllTime, err = db.loadTopEvents(0)
	if err != nil {
		return analytics, err
	}
	analytics.TopEvents.Last7Days, err = db.loadTopEvents(7)
	if err != nil {
		return analytics, err
	}

	engagement, err := db.GetMasterEngagement()
	if err != nil {
		return analytics, err
	}
	analytics.Engagement = engagement

	if err := db.c.QueryRow(`SELECT COUNT(*) FROM sponsor_exposures`).Scan(&analytics.SponsorStats.TotalImpressions); err != nil {
		return analytics, err
	}
	if err := db.c.QueryRow(`SELECT COUNT(*) FROM sponsor_clicks`).Scan(&analytics.SponsorStats.TotalClicks); err != nil {
		return analytics, err
	}
	if analytics.SponsorStats.TotalImpressions > 0 && analytics.SponsorStats.TotalClicks > 0 {
		analytics.SponsorStats.AverageCTR = (float64(analytics.SponsorStats.TotalClicks) / float64(analytics.SponsorStats.TotalImpressions)) * 100
	}

	orgImpressions := make(map[int]int)
	orgClicks := make(map[int]int)

	sponsorRows, err := db.c.Query(`SELECT e.organization_id, COUNT(*) FROM sponsor_exposures se INNER JOIN events e ON e.id = se.event_id WHERE e.organization_id > 0 GROUP BY e.organization_id`)
	if err != nil {
		return analytics, err
	}
	for sponsorRows.Next() {
		var orgID, count int
		if err := sponsorRows.Scan(&orgID, &count); err != nil {
			sponsorRows.Close()
			return analytics, err
		}
		orgImpressions[orgID] = count
	}
	sponsorRows.Close()

	sponsorRows, err = db.c.Query(`SELECT e.organization_id, COUNT(*) FROM sponsor_clicks sc INNER JOIN events e ON e.id = sc.event_id WHERE e.organization_id > 0 GROUP BY e.organization_id`)
	if err != nil {
		return analytics, err
	}
	for sponsorRows.Next() {
		var orgID, count int
		if err := sponsorRows.Scan(&orgID, &count); err != nil {
			sponsorRows.Close()
			return analytics, err
		}
		orgClicks[orgID] = count
	}
	sponsorRows.Close()

	for orgID, org := range orgInfo {
		if orgID <= 0 {
			continue
		}
		impressions := orgImpressions[orgID]
		clicks := orgClicks[orgID]
		ctr := 0.0
		if impressions > 0 && clicks > 0 {
			ctr = (float64(clicks) / float64(impressions)) * 100
		}
		analytics.SponsorStats.Organizations = append(analytics.SponsorStats.Organizations, SponsorOrganizationStat{
			OrganizationID: orgID,
			Name:           org.Name,
			Slug:           org.Slug,
			Impressions:    impressions,
			Clicks:         clicks,
			CTR:            ctr,
		})
	}

	now := time.Now()
	currentStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	nextStart := currentStart.AddDate(0, 1, 0)
	previousStart := currentStart.AddDate(0, -1, 0)

	analytics.MonthlySummary.Current = MonthlyMetrics{
		Month:       currentStart.Format("2006-01"),
		Votes:       db.countVotesBetween(currentStart, nextStart),
		Events:      db.countEventsBetween(currentStart, nextStart),
		UniqueUsers: db.countUniqueVotersBetween(currentStart, nextStart),
	}
	analytics.MonthlySummary.Previous = MonthlyMetrics{
		Month:       previousStart.Format("2006-01"),
		Votes:       db.countVotesBetween(previousStart, currentStart),
		Events:      db.countEventsBetween(previousStart, currentStart),
		UniqueUsers: db.countUniqueVotersBetween(previousStart, currentStart),
	}

	analytics.MonthlySummary.VotesChange = buildMetricDelta(analytics.MonthlySummary.Previous.Votes, analytics.MonthlySummary.Current.Votes)
	analytics.MonthlySummary.EventsChange = buildMetricDelta(analytics.MonthlySummary.Previous.Events, analytics.MonthlySummary.Current.Events)
	analytics.MonthlySummary.UniqueUsersChange = buildMetricDelta(analytics.MonthlySummary.Previous.UniqueUsers, analytics.MonthlySummary.Current.UniqueUsers)

	return analytics, nil
}

func (db *appdbimpl) loadTopEvents(lastDays int) ([]TopEventEntry, error) {
	var entries []TopEventEntry
	baseQuery := `
SELECT e.id,
       e.organization_id,
       IFNULL(o.name, ''),
       IFNULL(o.slug, ''),
       IFNULL(e.start_datetime, ''),
       IFNULL(t1.name, ''),
       IFNULL(t2.name, ''),
       COUNT(v.id) as votes
FROM events e
LEFT JOIN votes v ON v.event_id = e.id
LEFT JOIN organizations o ON o.id = e.organization_id
LEFT JOIN teams t1 ON t1.id = e.team1_id
LEFT JOIN teams t2 ON t2.id = e.team2_id
`
	var args []interface{}
	if lastDays > 0 {
		baseQuery += " WHERE datetime(v.created_at) >= datetime('now', ?)"
		args = append(args, fmt.Sprintf("-%d days", lastDays))
	}
	baseQuery += " GROUP BY e.id ORDER BY votes DESC, e.id DESC LIMIT 5"

	rows, err := db.c.Query(baseQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var entry TopEventEntry
		var startDate, team1, team2 string
		if err := rows.Scan(&entry.EventID, &entry.OrganizationID, &entry.OrganizationName, &entry.OrganizationSlug, &startDate, &team1, &team2, &entry.TotalVotes); err != nil {
			return nil, err
		}
		entry.StartDate = startDate
		labelParts := []string{strings.TrimSpace(team1), strings.TrimSpace(team2)}
		if labelParts[0] != "" && labelParts[1] != "" {
			entry.Label = fmt.Sprintf("%s vs %s", labelParts[0], labelParts[1])
		} else if labelParts[0] != "" {
			entry.Label = labelParts[0]
		} else if labelParts[1] != "" {
			entry.Label = labelParts[1]
		} else if startDate != "" {
			entry.Label = startDate
		} else {
			entry.Label = fmt.Sprintf("Evento %d", entry.EventID)
		}
		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return entries, nil
}

func (db *appdbimpl) countVotesBetween(start, end time.Time) int {
	var count int
	if err := db.c.QueryRow(`SELECT COUNT(*) FROM votes WHERE datetime(created_at) >= datetime(?) AND datetime(created_at) < datetime(?)`, start.Format(time.RFC3339), end.Format(time.RFC3339)).Scan(&count); err != nil {
		return 0
	}
	return count
}

func (db *appdbimpl) countEventsBetween(start, end time.Time) int {
	var count int
	if err := db.c.QueryRow(`SELECT COUNT(*) FROM events WHERE datetime(start_datetime) >= datetime(?) AND datetime(start_datetime) < datetime(?)`, start.Format(time.RFC3339), end.Format(time.RFC3339)).Scan(&count); err != nil {
		return 0
	}
	return count
}

func (db *appdbimpl) countUniqueVotersBetween(start, end time.Time) int {
	var count int
	if err := db.c.QueryRow(`SELECT COUNT(DISTINCT device_id) FROM votes WHERE datetime(created_at) >= datetime(?) AND datetime(created_at) < datetime(?)`, start.Format(time.RFC3339), end.Format(time.RFC3339)).Scan(&count); err != nil {
		return 0
	}
	return count
}

func buildMetricDelta(previous, current int) MetricDelta {
	delta := MetricDelta{Absolute: current - previous}
	if previous > 0 {
		delta.Percent = (float64(delta.Absolute) / float64(previous)) * 100
	} else if current > 0 {
		delta.Percent = 100
	}
	return delta
}

// Sponsor operations
func (db *appdbimpl) CreateSponsor(s Sponsor) (int, error) {
	sanitizedName := strings.TrimSpace(s.Name)
	sanitizedReportName := strings.TrimSpace(s.ReportName)
	if strings.TrimSpace(s.LogoData) == "" || s.OrganizationID == 0 {
		return 0, ErrInvalidSponsorData
	}

	var total int
	if err := db.c.QueryRow(`SELECT COUNT(*) FROM sponsors WHERE organization_id = ? AND is_active = 1`, s.OrganizationID).Scan(&total); err != nil {
		return 0, err
	}
	if s.IsActive && total >= maxSponsorSlots {
		return 0, ErrMaxSponsors
	}

	position := s.Position
	if position <= 0 || position > maxSponsorSlots {
		nextPos, err := db.nextSponsorPosition(s.OrganizationID)
		if err != nil {
			return 0, err
		}
		position = nextPos
	}

	sanitizedLink := strings.TrimSpace(s.LinkURL)
	isActive := s.IsActive
	if position > total+1 {
		position = total + 1
	}

	res, err := db.c.Exec(`INSERT INTO sponsors (organization_id, position, name, report_name, logo_data, link_url, is_active) VALUES (?, ?, ?, ?, ?, ?, ?)`, s.OrganizationID, position, sanitizedName, sanitizedReportName, s.LogoData, sanitizedLink, boolToInt(isActive))
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed: sponsors.position") {
			return 0, ErrInvalidSponsorPos
		}
		return 0, err
	}
	id, _ := res.LastInsertId()
	return int(id), nil
}

func (db *appdbimpl) UpdateSponsor(s Sponsor) error {
	if s.ID <= 0 {
		return sql.ErrNoRows
	}

	sanitizedName := strings.TrimSpace(s.Name)
	sanitizedReportName := strings.TrimSpace(s.ReportName)
	if strings.TrimSpace(s.LogoData) == "" {
		return ErrInvalidSponsorData
	}

	if s.Position <= 0 || s.Position > maxSponsorSlots || s.OrganizationID == 0 {
		return ErrInvalidSponsorPos
	}

	sanitizedLink := strings.TrimSpace(s.LinkURL)

	res, err := db.c.Exec(`UPDATE sponsors SET position=?, name=?, report_name=?, logo_data=?, link_url=?, is_active=? WHERE id=? AND organization_id = ?`, s.Position, sanitizedName, sanitizedReportName, s.LogoData, sanitizedLink, boolToInt(s.IsActive), s.ID, s.OrganizationID)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed: sponsors.position") {
			return ErrInvalidSponsorPos
		}
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (db *appdbimpl) DeleteSponsor(id int, organizationID int) error {
	res, err := db.c.Exec(`DELETE FROM sponsors WHERE id=? AND organization_id = ?`, id, organizationID)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return sql.ErrNoRows
	}
	return db.normalizeSponsorPositions(organizationID)
}

func (db *appdbimpl) ListSponsors(organizationID int) ([]Sponsor, error) {
	return db.querySponsors(false, organizationID)
}

func (db *appdbimpl) ListActiveSponsors(organizationID int) ([]Sponsor, error) {
	return db.querySponsors(true, organizationID)
}

func (db *appdbimpl) RecordSponsorSession(eventID int, deviceID string) error {
	if eventID <= 0 {
		return sql.ErrNoRows
	}
	trimmed := strings.TrimSpace(deviceID)
	if trimmed == "" {
		return sql.ErrNoRows
	}

	_, err := db.c.Exec(`
INSERT INTO sponsor_sessions (event_id, device_id)
VALUES (?, ?)
ON CONFLICT(event_id, device_id) DO UPDATE SET last_seen = CURRENT_TIMESTAMP
`, eventID, trimmed)
	return err
}

func (db *appdbimpl) RecordSponsorExposure(eventID int, sponsorIDs []int, deviceID, exposureType string, durationMs int) error {
	if eventID <= 0 {
		return sql.ErrNoRows
	}
	trimmedDevice := strings.TrimSpace(deviceID)
	if trimmedDevice == "" {
		return sql.ErrNoRows
	}
	normalizedType := strings.ToLower(strings.TrimSpace(exposureType))
	if normalizedType != "seen" && normalizedType != "watched" {
		return ErrInvalidSponsorData
	}
	if len(sponsorIDs) == 0 {
		return sql.ErrNoRows
	}

	tx, err := db.c.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT INTO sponsor_exposures (event_id, sponsor_id, device_id, exposure_type, duration_ms) VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, sponsorID := range sponsorIDs {
		if sponsorID <= 0 {
			continue
		}
		var duration interface{}
		if normalizedType == "watched" && durationMs > 0 {
			duration = durationMs
		} else {
			duration = nil
		}
		if _, err := stmt.Exec(eventID, sponsorID, trimmedDevice, normalizedType, duration); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (db *appdbimpl) RecordSponsorClick(eventID, sponsorID int, deviceID string) error {
	if eventID <= 0 || sponsorID <= 0 {
		return sql.ErrNoRows
	}
	trimmed := strings.TrimSpace(deviceID)
	_, err := db.c.Exec(`INSERT INTO sponsor_clicks (event_id, sponsor_id, device_id) VALUES (?, ?, ?)`, eventID, sponsorID, trimmed)
	return err
}

func (db *appdbimpl) RecordEngagementSession(eventID int, deviceID string, durationSeconds int) error {
	if eventID <= 0 || durationSeconds <= 0 {
		return ErrInvalidSponsorData
	}

	normalizedDevice := strings.TrimSpace(deviceID)
	if normalizedDevice == "" {
		return ErrInvalidSponsorData
	}

	_, err := db.c.Exec(`INSERT INTO page_engagements (event_id, device_id, duration_seconds) VALUES (?, ?, ?)`, eventID, normalizedDevice, durationSeconds)
	return err
}

func (db *appdbimpl) RecordPostVoteAction(eventID int, deviceID, action string) error {
	if eventID <= 0 {
		return ErrInvalidSponsorData
	}

	normalizedDevice := strings.TrimSpace(deviceID)
	if normalizedDevice == "" {
		return ErrInvalidSponsorData
	}

	if _, ok := allowedPostVoteActions[action]; !ok {
		return ErrInvalidSponsorData
	}

	_, err := db.c.Exec(`INSERT INTO post_vote_actions (event_id, device_id, action) VALUES (?, ?, ?)`, eventID, normalizedDevice, action)
	return err
}

func (db *appdbimpl) GetEventEngagement(eventID int) (EventEngagementStats, error) {
	stats := EventEngagementStats{EventID: eventID}
	if eventID <= 0 {
		return stats, sql.ErrNoRows
	}

	var total int64
	var avg sql.NullFloat64
	var users int

	if err := db.c.QueryRow(`SELECT IFNULL(SUM(duration_seconds), 0), IFNULL(AVG(duration_seconds), 0), COUNT(DISTINCT device_id) FROM page_engagements WHERE event_id = ?`, eventID).Scan(&total, &avg, &users); err != nil {
		return stats, err
	}

	stats.TotalDurationSeconds = total
	if avg.Valid {
		stats.AverageDurationSeconds = avg.Float64
	}
	stats.TotalUsers = users

	rows, err := db.c.Query(`SELECT action, COUNT(DISTINCT device_id) FROM post_vote_actions WHERE event_id = ? GROUP BY action`, eventID)
	if err != nil {
		return stats, err
	}
	defer rows.Close()

	for rows.Next() {
		var action string
		var count int
		if err := rows.Scan(&action, &count); err != nil {
			return stats, err
		}
		switch action {
		case "vote_trend_open":
			stats.VoteTrendOpens = count
		case "selfie_open":
			stats.SelfieOpens = count
		case "selfie_abandon":
			stats.SelfieAbandons = count
		case "reaction_open":
			stats.ReactionOpens = count
		case "reaction_abandon":
			stats.ReactionAbandons = count
		case "experience_open":
			stats.ExperienceOpens = count
		case "experience_abandon":
			stats.ExperienceAbandons = count
		case "photo_edit_open":
			stats.PhotoEditOpens = count
		case "vote_edit_open":
			stats.VoteEditOpens = count
		case "vote_edit_abandon":
			stats.VoteEditAbandons = count
		case "vote_edit_complete":
			stats.VoteEditCompletions = count
		}
	}
	return stats, nil
}

func (db *appdbimpl) GetMasterEngagement() (MasterEngagementSummary, error) {
	summary := MasterEngagementSummary{Organizations: []OrganizationEngagementStat{}}

	rows, err := db.c.Query(`
SELECT e.organization_id,
       IFNULL(o.name, ''),
       IFNULL(o.slug, ''),
       COUNT(DISTINCT e.id) as events,
       IFNULL(SUM(pe.duration_seconds), 0) as total_duration,
       COUNT(DISTINCT pe.device_id) as total_users
FROM events e
LEFT JOIN organizations o ON o.id = e.organization_id
LEFT JOIN page_engagements pe ON pe.event_id = e.id
WHERE e.organization_id > 0
GROUP BY e.organization_id
        `)
	if err != nil {
		return summary, err
	}
	defer rows.Close()

	var totalEvents int
	var totalUsers int

	for rows.Next() {
		var orgID, events int
		var name, slug string
		var duration int64
		var users int

		if err := rows.Scan(&orgID, &name, &slug, &events, &duration, &users); err != nil {
			return summary, err
		}

		stat := OrganizationEngagementStat{
			OrganizationID:       orgID,
			Name:                 name,
			Slug:                 slug,
			TotalDurationSeconds: duration,
		}
		if events > 0 {
			stat.AverageDurationPerMatch = float64(duration) / float64(events)
			totalEvents += events
		}
		if users > 0 {
			stat.AverageDurationPerUser = float64(duration) / float64(users)
			totalUsers += users
		}

		summary.TotalDurationSeconds += duration
		summary.Organizations = append(summary.Organizations, stat)
	}

	if totalEvents > 0 {
		summary.AverageDurationPerMatch = float64(summary.TotalDurationSeconds) / float64(totalEvents)
	}
	if totalUsers > 0 {
		summary.AverageDurationPerUser = float64(summary.TotalDurationSeconds) / float64(totalUsers)
	}

	return summary, rows.Err()
}

func (db *appdbimpl) GetSponsorAnalytics(eventID int) (SponsorAnalytics, error) {
	summary := SponsorAnalytics{}
	if eventID <= 0 {
		return summary, sql.ErrNoRows
	}

	if err := db.c.QueryRow(`SELECT COUNT(*) FROM sponsor_sessions WHERE event_id = ?`, eventID).Scan(&summary.TotalSessions); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			summary.TotalSessions = 0
		} else {
			return summary, err
		}
	}

	if err := db.c.QueryRow(`SELECT COUNT(DISTINCT device_id) FROM sponsor_exposures WHERE event_id = ? AND exposure_type = 'seen'`, eventID).Scan(&summary.SeenSessions); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			summary.SeenSessions = 0
		} else {
			return summary, err
		}
	}

	if err := db.c.QueryRow(`SELECT COUNT(DISTINCT device_id) FROM sponsor_exposures WHERE event_id = ? AND exposure_type = 'watched'`, eventID).Scan(&summary.WatchedSessions); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			summary.WatchedSessions = 0
		} else {
			return summary, err
		}
	}

	var totalWatch sql.NullInt64
	if err := db.c.QueryRow(`SELECT SUM(COALESCE(duration_ms, 0)) FROM sponsor_exposures WHERE event_id = ? AND exposure_type = 'watched'`, eventID).Scan(&totalWatch); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return summary, err
		}
	}
	if totalWatch.Valid {
		summary.TotalWatchTimeMs = totalWatch.Int64
	}
	if summary.WatchedSessions > 0 && summary.TotalWatchTimeMs > 0 {
		summary.AverageWatchTime = float64(summary.TotalWatchTimeMs) / float64(summary.WatchedSessions)
	}

	if err := db.c.QueryRow(`SELECT COUNT(*) FROM sponsor_clicks WHERE event_id = ?`, eventID).Scan(&summary.TotalClicks); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			summary.TotalClicks = 0
		} else {
			return summary, err
		}
	}

	if err := db.c.QueryRow(`SELECT COUNT(DISTINCT device_id) FROM sponsor_clicks WHERE event_id = ?`, eventID).Scan(&summary.UniqueClickers); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			summary.UniqueClickers = 0
		} else {
			return summary, err
		}
	}

	row := db.c.QueryRow(`
SELECT e.sponsor_id,
       IFNULL(s.name, ''),
       IFNULL(s.report_name, ''),
       COUNT(e.id) AS views
FROM sponsor_exposures e
INNER JOIN sponsors s ON s.id = e.sponsor_id
WHERE e.event_id = ? AND e.exposure_type = 'seen'
GROUP BY e.sponsor_id, s.name, s.report_name
ORDER BY views DESC, s.id ASC
LIMIT 1
`, eventID)
	var top SponsorViewStat
	switch err := row.Scan(&top.SponsorID, &top.Name, &top.ReportName, &top.Views); err {
	case nil:
		summary.TopSponsor = &top
	case sql.ErrNoRows:
		summary.TopSponsor = nil
	default:
		return summary, err
	}

	timelineRows, err := db.c.Query(`
SELECT bucket,
       SUM(seen) AS seen,
        SUM(watched) AS watched,
        SUM(clicks) AS clicks
FROM (
        SELECT strftime('%Y-%m-%dT%H:%M:00Z', created_at) AS bucket,
               CASE WHEN exposure_type = 'seen' THEN 1 ELSE 0 END AS seen,
               CASE WHEN exposure_type = 'watched' THEN 1 ELSE 0 END AS watched,
               0 AS clicks
        FROM sponsor_exposures
        WHERE event_id = ?
        UNION ALL
        SELECT strftime('%Y-%m-%dT%H:%M:00Z', clicked_at) AS bucket,
               0 AS seen,
               0 AS watched,
               1 AS clicks
        FROM sponsor_clicks
        WHERE event_id = ?
)
GROUP BY bucket
ORDER BY bucket ASC
`, eventID, eventID)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return summary, err
		}
	} else {
		defer timelineRows.Close()
		for timelineRows.Next() {
			var point SponsorTimelinePoint
			if err := timelineRows.Scan(&point.Timestamp, &point.Seen, &point.Watched, &point.Clicks); err != nil {
				return summary, err
			}
			summary.Timeline = append(summary.Timeline, point)
		}
		if err := timelineRows.Err(); err != nil {
			return summary, err
		}
	}

	return summary, nil
}

func (db *appdbimpl) GetSponsorClickStats(eventID int) ([]SponsorClickStat, error) {
	rows, err := db.c.Query(`
WITH relevant_sponsors AS (
        SELECT DISTINCT sponsor_id
        FROM sponsor_clicks
        WHERE event_id = ?
        UNION
        SELECT DISTINCT sponsor_id
        FROM sponsor_exposures
        WHERE event_id = ?
), click_totals AS (
        SELECT sponsor_id, COUNT(id) AS clicks
        FROM sponsor_clicks
        WHERE event_id = ?
        GROUP BY sponsor_id
)
SELECT s.id,
       IFNULL(s.name, ''),
       IFNULL(s.report_name, ''),
       IFNULL(s.link_url, ''),
       COALESCE(ct.clicks, 0) AS clicks,
       IFNULL(s.position, 0)
FROM relevant_sponsors rs
INNER JOIN sponsors s ON s.id = rs.sponsor_id
LEFT JOIN click_totals ct ON ct.sponsor_id = s.id
ORDER BY s.position ASC, s.id ASC
`, eventID, eventID, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []SponsorClickStat
	for rows.Next() {
		var stat SponsorClickStat
		var position int
		if err := rows.Scan(&stat.SponsorID, &stat.Name, &stat.ReportName, &stat.LinkURL, &stat.Clicks, &position); err != nil {
			return nil, err
		}
		stats = append(stats, stat)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return stats, nil
}

func (db *appdbimpl) querySponsors(activeOnly bool, organizationID int) ([]Sponsor, error) {
	if organizationID == 0 {
		return []Sponsor{}, nil
	}

	baseQuery := `SELECT id, position, name, IFNULL(report_name, ''), logo_data, IFNULL(link_url, ''), is_active FROM sponsors WHERE organization_id = ?`
	if activeOnly {
		baseQuery += ` AND is_active = 1`
	}
	baseQuery += ` ORDER BY position ASC, id ASC`

	rows, err := db.c.Query(baseQuery, organizationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sponsors []Sponsor
	for rows.Next() {
		var s Sponsor
		var isActive int
		if err := rows.Scan(&s.ID, &s.Position, &s.Name, &s.ReportName, &s.LogoData, &s.LinkURL, &isActive); err != nil {
			return nil, err
		}
		s.IsActive = isActive == 1
		sponsors = append(sponsors, s)
	}
	return sponsors, nil
}

func (db *appdbimpl) nextSponsorPosition(organizationID int) (int, error) {
	rows, err := db.c.Query(`SELECT position FROM sponsors WHERE organization_id = ? ORDER BY position ASC`, organizationID)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	used := make(map[int]struct{})
	for rows.Next() {
		var pos int
		if err := rows.Scan(&pos); err != nil {
			return 0, err
		}
		used[pos] = struct{}{}
	}

	for i := 1; i <= maxSponsorSlots; i++ {
		if _, ok := used[i]; !ok {
			return i, nil
		}
	}

	return 0, ErrMaxSponsors
}

func (db *appdbimpl) normalizeSponsorPositions(organizationID int) error {
	rows, err := db.c.Query(`SELECT id FROM sponsors WHERE organization_id = ? ORDER BY position ASC, id ASC`, organizationID)
	if err != nil {
		return err
	}
	defer rows.Close()

	position := 1
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return err
		}
		if _, err := db.c.Exec(`UPDATE sponsors SET position=? WHERE id=?`, position, id); err != nil {
			return err
		}
		position++
	}
	return nil
}

func (db *appdbimpl) RecordContactSubmission(contact ContactSubmission) (ContactSubmission, error) {
	if contact.EventID <= 0 {
		return ContactSubmission{}, sql.ErrNoRows
	}

	normalizedDevice := strings.TrimSpace(contact.DeviceID)
	normalizedValue := strings.TrimSpace(contact.ContactValue)
	normalizedType := strings.ToLower(strings.TrimSpace(contact.ContactType))
	normalizedBonus := strings.TrimSpace(contact.BonusCode)
	normalizedSignature := strings.TrimSpace(contact.BonusSignature)

	if normalizedDevice == "" || normalizedValue == "" || normalizedType == "" {
		return ContactSubmission{}, ErrInvalidContactData
	}

	res := ContactSubmission{
		EventID:          contact.EventID,
		DeviceID:         normalizedDevice,
		ContactValue:     normalizedValue,
		ContactType:      normalizedType,
		MarketingConsent: contact.MarketingConsent,
		BonusCode:        normalizedBonus,
		BonusSignature:   normalizedSignature,
		IsVerified:       false,
	}

	result, err := db.c.Exec(`INSERT INTO contacts (event_id, device_id, contact_value, contact_type, marketing_consent, is_verified, bonus_code, bonus_signature) VALUES (?, ?, ?, ?, ?, 0, ?, ?)`, res.EventID, res.DeviceID, res.ContactValue, res.ContactType, boolToInt(res.MarketingConsent), res.BonusCode, res.BonusSignature)
	if err != nil {
		return ContactSubmission{}, err
	}

	insertID, err := result.LastInsertId()
	if err == nil {
		res.ID = int(insertID)
	}

	var createdAt string
	var isVerified int
	if err := db.c.QueryRow(`SELECT created_at, is_verified FROM contacts WHERE event_id = ? AND device_id = ?`, res.EventID, res.DeviceID).Scan(&createdAt, &isVerified); err == nil {
		res.CreatedAt = createdAt
		res.IsVerified = isVerified == 1
	}

	return res, nil
}

func (db *appdbimpl) GetContactSubmission(eventID int, deviceID string) (ContactSubmission, error) {
	var contact ContactSubmission
	if eventID <= 0 {
		return contact, sql.ErrNoRows
	}

	normalizedDevice := strings.TrimSpace(deviceID)
	if normalizedDevice == "" {
		return contact, sql.ErrNoRows
	}

	var marketingConsent int
	var isVerified int

	err := db.c.QueryRow(`SELECT id, event_id, device_id, contact_value, contact_type, marketing_consent, is_verified, IFNULL(bonus_code, ''), IFNULL(bonus_signature, ''), created_at FROM contacts WHERE event_id = ? AND device_id = ? LIMIT 1`, eventID, normalizedDevice).Scan(&contact.ID, &contact.EventID, &contact.DeviceID, &contact.ContactValue, &contact.ContactType, &marketingConsent, &isVerified, &contact.BonusCode, &contact.BonusSignature, &contact.CreatedAt)
	if err != nil {
		return ContactSubmission{}, err
	}

	contact.MarketingConsent = marketingConsent == 1
	contact.IsVerified = isVerified == 1
	return contact, nil
}

func (db *appdbimpl) ListContactBonuses(eventID int, deviceID string) ([]ContactSubmission, error) {
	if eventID <= 0 {
		return nil, sql.ErrNoRows
	}

	normalizedDevice := strings.TrimSpace(deviceID)
	if normalizedDevice == "" {
		return nil, ErrInvalidContactData
	}

	rows, err := db.c.Query(`SELECT id, event_id, device_id, contact_value, contact_type, marketing_consent, is_verified, IFNULL(bonus_code, ''), IFNULL(bonus_signature, ''), created_at FROM contacts WHERE event_id = ? AND device_id = ? ORDER BY created_at ASC`, eventID, normalizedDevice)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var results []ContactSubmission
	for rows.Next() {
		var contact ContactSubmission
		var marketingConsent int
		var isVerified int
		if err := rows.Scan(&contact.ID, &contact.EventID, &contact.DeviceID, &contact.ContactValue, &contact.ContactType, &marketingConsent, &isVerified, &contact.BonusCode, &contact.BonusSignature, &contact.CreatedAt); err != nil {
			return nil, err
		}
		contact.MarketingConsent = marketingConsent == 1
		contact.IsVerified = isVerified == 1
		results = append(results, contact)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

func (db *appdbimpl) RecordContactEvent(eventID int, deviceID, name string) error {
	if eventID <= 0 {
		return sql.ErrNoRows
	}

	normalizedDevice := strings.TrimSpace(deviceID)
	normalizedName := strings.TrimSpace(name)
	if normalizedDevice == "" || normalizedName == "" {
		return ErrInvalidContactData
	}

	_, err := db.c.Exec(`INSERT INTO contact_events (event_id, device_id, event_name) VALUES (?, ?, ?)`, eventID, normalizedDevice, normalizedName)
	return err
}

func parseSQLiteTimestamp(value string) (time.Time, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return time.Time{}, fmt.Errorf("empty timestamp")
	}

	candidates := []string{trimmed}
	if !strings.Contains(trimmed, "T") {
		candidates = append(candidates, strings.Replace(trimmed, " ", "T", 1))
	}

	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05.000000000",
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02T15:04:05.000000000",
		"2006-01-02T15:04:05",
		"2006-01-02T15:04",
	}

	for _, candidate := range candidates {
		for _, layout := range layouts {
			if ts, err := time.ParseInLocation(layout, candidate, time.UTC); err == nil {
				return ts, nil
			}
		}
	}

	return time.Time{}, fmt.Errorf("unsupported timestamp format: %s", value)
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func normalizeRosterSchema(value int) int {
	switch value {
	case 12, 13, 14:
		return value
	default:
		return 13
	}
}

func nullableOrgID(orgID int) interface{} {
	if orgID <= 0 {
		return nil
	}
	return orgID
}

func normalizeSlug(value string) string {
	sanitized := strings.TrimSpace(value)
	if sanitized == "" {
		return ""
	}
	lowerValue := strings.ToLower(sanitized)
	if strings.HasPrefix(lowerValue, "http://") || strings.HasPrefix(lowerValue, "https://") {
		return sanitized
	}
	slug := slugSanitizer.ReplaceAllString(lowerValue, "-")
	slug = strings.Trim(slug, "-")
	return slug
}

func (db *appdbimpl) ensureOrganizationSlugAvailable(slug string, excludeID int) error {
	if slug == "" {
		return ErrInvalidOrganizationData
	}
	var existingID int
	err := db.c.QueryRow(`SELECT id FROM organizations WHERE slug = ? AND id != ? LIMIT 1`, slug, excludeID).Scan(&existingID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	return ErrInvalidOrganizationData
}

func (db *appdbimpl) RecordEventFeedback(feedback EventFeedback) error {
	if feedback.EventID <= 0 {
		return sql.ErrNoRows
	}

	experience := strings.TrimSpace(feedback.Experience)
	teamSpirit := strings.TrimSpace(feedback.TeamSpirit)
	perksInterest := strings.TrimSpace(feedback.PerksInterest)
	miniGamesInterest := strings.TrimSpace(feedback.MiniGamesInterest)

	if experience == "" || teamSpirit == "" || perksInterest == "" || miniGamesInterest == "" {
		return fmt.Errorf("invalid feedback payload")
	}

	suggestion := strings.TrimSpace(feedback.Suggestion)
	if suggestion != "" {
		runes := []rune(suggestion)
		if len(runes) > 80 {
			suggestion = string(runes[:80])
		}
	}

	_, err := db.c.Exec(`INSERT INTO event_feedback (event_id, experience, team_spirit, perks_interest, mini_games_interest, suggestion) VALUES (?, ?, ?, ?, ?, ?)`, feedback.EventID, experience, teamSpirit, perksInterest, miniGamesInterest, suggestion)
	return err
}

func (db *appdbimpl) GetEventFeedbackSummary(eventID int) (EventFeedbackSummary, error) {
	summary := EventFeedbackSummary{
		ExperienceCounts:        map[string]int{"very_easy": 0, "easy": 0, "complex": 0, "hard": 0},
		TeamSpiritCounts:        map[string]int{"high": 0, "medium": 0, "low": 0},
		PerksInterestCounts:     map[string]int{"yes": 0, "maybe": 0, "no": 0},
		MiniGamesInterestCounts: map[string]int{"super_excited": 0, "maybe": 0, "no": 0},
		Suggestions:             []string{},
	}

	if eventID <= 0 {
		return summary, fmt.Errorf("invalid event id")
	}

	if err := db.c.QueryRow(`SELECT COUNT(*) FROM event_feedback WHERE event_id=?`, eventID).Scan(&summary.TotalResponses); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			summary.TotalResponses = 0
			return summary, nil
		}
		return summary, err
	}

	if err := db.loadFeedbackCounts(eventID, `SELECT experience, COUNT(*) FROM event_feedback WHERE event_id=? GROUP BY experience`, summary.ExperienceCounts); err != nil {
		return summary, err
	}
	if err := db.loadFeedbackCounts(eventID, `SELECT team_spirit, COUNT(*) FROM event_feedback WHERE event_id=? GROUP BY team_spirit`, summary.TeamSpiritCounts); err != nil {
		return summary, err
	}
	if err := db.loadFeedbackCounts(eventID, `SELECT perks_interest, COUNT(*) FROM event_feedback WHERE event_id=? GROUP BY perks_interest`, summary.PerksInterestCounts); err != nil {
		return summary, err
	}
	if err := db.loadFeedbackCounts(eventID, `SELECT mini_games_interest, COUNT(*) FROM event_feedback WHERE event_id=? GROUP BY mini_games_interest`, summary.MiniGamesInterestCounts); err != nil {
		return summary, err
	}

	rows, err := db.c.Query(`SELECT suggestion FROM event_feedback WHERE event_id=? AND TRIM(suggestion) <> '' ORDER BY created_at DESC, id DESC`, eventID)
	if err != nil {
		return summary, err
	}
	defer rows.Close()

	for rows.Next() {
		var suggestion string
		if err := rows.Scan(&suggestion); err != nil {
			return summary, err
		}
		trimmed := strings.TrimSpace(suggestion)
		if trimmed != "" {
			summary.Suggestions = append(summary.Suggestions, trimmed)
		}
	}

	if err := rows.Err(); err != nil {
		return summary, err
	}

	return summary, nil
}

func (db *appdbimpl) loadFeedbackCounts(eventID int, query string, counts map[string]int) error {
	rows, err := db.c.Query(query, eventID)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var value string
		var total int
		if err := rows.Scan(&value, &total); err != nil {
			return err
		}
		key := strings.TrimSpace(strings.ToLower(value))
		if key == "" {
			continue
		}
		if _, ok := counts[key]; ok {
			counts[key] = total
		}
	}

	return rows.Err()
}

func (db *appdbimpl) ListShopProducts() ([]ShopProduct, error) {
	rows, err := db.c.Query(`SELECT id, name, description, price_cents, image_url, IFNULL(created_at, '') FROM shop_products ORDER BY id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []ShopProduct
	for rows.Next() {
		var product ShopProduct
		if err := rows.Scan(&product.ID, &product.Name, &product.Description, &product.PriceCents, &product.ImageURL, &product.CreatedAt); err != nil {
			return nil, err
		}
		products = append(products, product)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return products, nil
}

func (db *appdbimpl) GetShopProduct(id int) (ShopProduct, error) {
	if id <= 0 {
		return ShopProduct{}, sql.ErrNoRows
	}

	var product ShopProduct
	err := db.c.QueryRow(`SELECT id, name, description, price_cents, image_url, IFNULL(created_at, '') FROM shop_products WHERE id = ?`, id).
		Scan(&product.ID, &product.Name, &product.Description, &product.PriceCents, &product.ImageURL, &product.CreatedAt)
	if err != nil {
		return ShopProduct{}, err
	}

	return product, nil
}

func (db *appdbimpl) CreateShopProduct(product ShopProduct) (ShopProduct, error) {
	name := strings.TrimSpace(product.Name)
	if name == "" {
		return ShopProduct{}, fmt.Errorf("product name is required")
	}

	description := strings.TrimSpace(product.Description)
	imageURL := strings.TrimSpace(product.ImageURL)
	price := product.PriceCents
	if price <= 0 {
		return ShopProduct{}, fmt.Errorf("product price must be greater than zero")
	}

	res, err := db.c.Exec(`INSERT INTO shop_products (name, description, price_cents, image_url) VALUES (?, ?, ?, ?)`, name, description, price, imageURL)
	if err != nil {
		return ShopProduct{}, err
	}

	productID, err := res.LastInsertId()
	if err != nil {
		return ShopProduct{}, err
	}

	created := ShopProduct{
		ID:          int(productID),
		Name:        name,
		Description: description,
		PriceCents:  price,
		ImageURL:    imageURL,
	}

	if err := db.c.QueryRow(`SELECT IFNULL(created_at, '') FROM shop_products WHERE id = ?`, created.ID).Scan(&created.CreatedAt); err != nil {
		return ShopProduct{}, err
	}

	return created, nil
}

func (db *appdbimpl) ListShopOrders() ([]ShopOrder, error) {
	rows, err := db.c.Query(`SELECT id, customer_name, customer_email, customer_notes, total_cents, IFNULL(created_at, '') FROM shop_orders ORDER BY id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []ShopOrder
	for rows.Next() {
		var order ShopOrder
		if err := rows.Scan(&order.ID, &order.CustomerName, &order.CustomerEmail, &order.CustomerNotes, &order.TotalCents, &order.CreatedAt); err != nil {
			return nil, err
		}

		itemRows, err := db.c.Query(`SELECT id, order_id, product_id, product_name, product_image_url, quantity, unit_price_cents FROM shop_order_items WHERE order_id = ? ORDER BY id ASC`, order.ID)
		if err != nil {
			return nil, err
		}

		var items []ShopOrderItem
		for itemRows.Next() {
			var item ShopOrderItem
			if err := itemRows.Scan(&item.ID, &item.OrderID, &item.ProductID, &item.ProductName, &item.ProductImageURL, &item.Quantity, &item.UnitPriceCents); err != nil {
				itemRows.Close()
				return nil, err
			}
			items = append(items, item)
		}

		if err := itemRows.Err(); err != nil {
			itemRows.Close()
			return nil, err
		}
		itemRows.Close()

		order.Items = items
		orders = append(orders, order)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return orders, nil
}

func (db *appdbimpl) CreateShopOrder(order ShopOrder, items []ShopOrderItem) (ShopOrder, error) {
	if len(items) == 0 {
		return ShopOrder{}, fmt.Errorf("order must contain at least one item")
	}

	customerName := strings.TrimSpace(order.CustomerName)
	customerEmail := strings.TrimSpace(order.CustomerEmail)
	customerNotes := strings.TrimSpace(order.CustomerNotes)

	if customerName == "" || customerEmail == "" {
		return ShopOrder{}, fmt.Errorf("customer information is required")
	}

	tx, err := db.c.Begin()
	if err != nil {
		return ShopOrder{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	res, err := tx.Exec(`INSERT INTO shop_orders (customer_name, customer_email, customer_notes, total_cents) VALUES (?, ?, ?, ?)`, customerName, customerEmail, customerNotes, order.TotalCents)
	if err != nil {
		return ShopOrder{}, err
	}

	orderID, err := res.LastInsertId()
	if err != nil {
		return ShopOrder{}, err
	}

	order.ID = int(orderID)
	order.CustomerName = customerName
	order.CustomerEmail = customerEmail
	order.CustomerNotes = customerNotes

	storedItems := make([]ShopOrderItem, 0, len(items))
	for _, item := range items {
		if item.ProductID <= 0 || item.Quantity <= 0 {
			return ShopOrder{}, fmt.Errorf("invalid order item")
		}

		cleanName := strings.TrimSpace(item.ProductName)
		cleanImage := strings.TrimSpace(item.ProductImageURL)

		result, err := tx.Exec(`INSERT INTO shop_order_items (order_id, product_id, product_name, product_image_url, quantity, unit_price_cents) VALUES (?, ?, ?, ?, ?, ?)`, order.ID, item.ProductID, cleanName, cleanImage, item.Quantity, item.UnitPriceCents)
		if err != nil {
			return ShopOrder{}, err
		}

		itemID, err := result.LastInsertId()
		if err != nil {
			return ShopOrder{}, err
		}

		storedItems = append(storedItems, ShopOrderItem{
			ID:              int(itemID),
			OrderID:         order.ID,
			ProductID:       item.ProductID,
			ProductName:     cleanName,
			ProductImageURL: cleanImage,
			Quantity:        item.Quantity,
			UnitPriceCents:  item.UnitPriceCents,
		})
	}

	if err := tx.QueryRow(`SELECT IFNULL(created_at, '') FROM shop_orders WHERE id = ?`, order.ID).Scan(&order.CreatedAt); err != nil {
		return ShopOrder{}, err
	}

	if err := tx.Commit(); err != nil {
		return ShopOrder{}, err
	}
	committed = true

	order.Items = storedItems

	return order, nil
}

// QR redirect management

func (db *appdbimpl) UpsertQRRedirect(sourcePath, targetPath string) (QRRedirect, error) {
	cleanSource := strings.TrimSpace(sourcePath)
	cleanTarget := strings.TrimSpace(targetPath)
	if cleanSource == "" || cleanTarget == "" {
		return QRRedirect{}, fmt.Errorf("invalid qr redirect payload")
	}

	_, err := db.c.Exec(`INSERT INTO qr_redirects (source_path, target_path, hits, created_at, updated_at)
VALUES (?, ?, 0, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT(source_path) DO UPDATE SET target_path = excluded.target_path, updated_at = CURRENT_TIMESTAMP`, cleanSource, cleanTarget)
	if err != nil {
		return QRRedirect{}, err
	}

	return db.GetQRRedirectBySource(cleanSource)
}

func (db *appdbimpl) ListQRRedirects() ([]QRRedirect, error) {
	rows, err := db.c.Query(`SELECT id, source_path, target_path, hits, created_at, updated_at FROM qr_redirects ORDER BY datetime(updated_at) DESC, id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var redirects []QRRedirect
	for rows.Next() {
		var entry QRRedirect
		if err := rows.Scan(&entry.ID, &entry.SourcePath, &entry.TargetPath, &entry.Hits, &entry.CreatedAt, &entry.UpdatedAt); err != nil {
			return nil, err
		}
		redirects = append(redirects, entry)
	}
	return redirects, rows.Err()
}

func (db *appdbimpl) GetQRRedirectBySource(sourcePath string) (QRRedirect, error) {
	cleanSource := strings.TrimSpace(sourcePath)
	if cleanSource == "" {
		return QRRedirect{}, sql.ErrNoRows
	}
	var entry QRRedirect
	err := db.c.QueryRow(`SELECT id, source_path, target_path, hits, created_at, updated_at FROM qr_redirects WHERE source_path = ?`, cleanSource).Scan(&entry.ID, &entry.SourcePath, &entry.TargetPath, &entry.Hits, &entry.CreatedAt, &entry.UpdatedAt)
	if err != nil {
		return QRRedirect{}, err
	}
	return entry, nil
}

func (db *appdbimpl) IncrementQRRedirectHit(id int) error {
	if id <= 0 {
		return fmt.Errorf("invalid qr redirect id")
	}
	_, err := db.c.Exec(`UPDATE qr_redirects SET hits = hits + 1 WHERE id = ?`, id)
	return err
}

func (db *appdbimpl) DeleteQRRedirect(id int) error {
	if id <= 0 {
		return fmt.Errorf("invalid qr redirect id")
	}
	result, err := db.c.Exec(`DELETE FROM qr_redirects WHERE id = ?`, id)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// Coupon management

func (db *appdbimpl) CreateCoupon(coupon Coupon) (Coupon, error) {
	cleanTitle := strings.TrimSpace(coupon.Title)
	if cleanTitle == "" || coupon.SponsorID <= 0 {
		return Coupon{}, fmt.Errorf("invalid coupon payload")
	}

	matchIDs := joinMatchIDs(coupon.MatchIDs)
	now := time.Now().UTC().Format(time.RFC3339)
	result, err := db.c.Exec(`INSERT INTO coupons (title, short_desc, sponsor_id, merchant_id, match_ids, start_date, end_date, max_uses, status, image_url, highlight, segmentation, organization_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, cleanTitle, strings.TrimSpace(coupon.ShortDesc), coupon.SponsorID, coupon.MerchantID, matchIDs, coupon.StartDate, coupon.EndDate, coupon.MaxUses, strings.TrimSpace(coupon.Status), strings.TrimSpace(coupon.ImageURL), boolToInt(coupon.Highlight), strings.TrimSpace(coupon.Segmentation), coupon.OrganizationID, now, now)
	if err != nil {
		return Coupon{}, err
	}

	couponID, err := result.LastInsertId()
	if err != nil {
		return Coupon{}, err
	}

	return db.GetCouponByID(int(couponID))
}

func (db *appdbimpl) UpdateCoupon(coupon Coupon) (Coupon, error) {
	if coupon.ID <= 0 {
		return Coupon{}, fmt.Errorf("invalid coupon id")
	}

	matchIDs := joinMatchIDs(coupon.MatchIDs)
	_, err := db.c.Exec(`UPDATE coupons SET title=?, short_desc=?, sponsor_id=?, merchant_id=?, match_ids=?, start_date=?, end_date=?, max_uses=?, status=?, image_url=?, highlight=?, segmentation=?, organization_id=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`, strings.TrimSpace(coupon.Title), strings.TrimSpace(coupon.ShortDesc), coupon.SponsorID, coupon.MerchantID, matchIDs, coupon.StartDate, coupon.EndDate, coupon.MaxUses, strings.TrimSpace(coupon.Status), strings.TrimSpace(coupon.ImageURL), boolToInt(coupon.Highlight), strings.TrimSpace(coupon.Segmentation), coupon.OrganizationID, coupon.ID)
	if err != nil {
		return Coupon{}, err
	}
	return db.GetCouponByID(coupon.ID)
}

func (db *appdbimpl) DeleteCoupon(id int, organizationID int) error {
	if id <= 0 {
		return fmt.Errorf("invalid coupon id")
	}
	res, err := db.c.Exec(`DELETE FROM coupons WHERE id=? AND (organization_id = ? OR ? = 0)`, id, organizationID, organizationID)
	if err != nil {
		return err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (db *appdbimpl) ListCoupons(organizationID int) ([]Coupon, error) {
	rows, err := db.c.Query(`SELECT id, title, short_desc, sponsor_id, merchant_id, match_ids, start_date, end_date, max_uses, status, image_url, highlight, segmentation, total_views, total_claims, total_redemptions, created_at, updated_at, organization_id FROM coupons WHERE organization_id = ? OR ? = 0 ORDER BY created_at DESC`, organizationID, organizationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var coupons []Coupon
	for rows.Next() {
		var c Coupon
		var matchIDs string
		var highlight int
		if err := rows.Scan(&c.ID, &c.Title, &c.ShortDesc, &c.SponsorID, &c.MerchantID, &matchIDs, &c.StartDate, &c.EndDate, &c.MaxUses, &c.Status, &c.ImageURL, &highlight, &c.Segmentation, &c.TotalViews, &c.TotalClaims, &c.TotalRedemptions, &c.CreatedAt, &c.UpdatedAt, &c.OrganizationID); err != nil {
			return nil, err
		}
		c.MatchIDs = parseMatchIDs(matchIDs)
		c.Highlight = highlight == 1
		coupons = append(coupons, c)
	}
	return coupons, rows.Err()
}

func (db *appdbimpl) GetCouponByID(id int) (Coupon, error) {
	var c Coupon
	var matchIDs string
	var highlight int
	err := db.c.QueryRow(`SELECT id, title, short_desc, sponsor_id, merchant_id, match_ids, start_date, end_date, max_uses, status, image_url, highlight, segmentation, total_views, total_claims, total_redemptions, created_at, updated_at, organization_id FROM coupons WHERE id=?`, id).Scan(&c.ID, &c.Title, &c.ShortDesc, &c.SponsorID, &c.MerchantID, &matchIDs, &c.StartDate, &c.EndDate, &c.MaxUses, &c.Status, &c.ImageURL, &highlight, &c.Segmentation, &c.TotalViews, &c.TotalClaims, &c.TotalRedemptions, &c.CreatedAt, &c.UpdatedAt, &c.OrganizationID)
	if err != nil {
		return Coupon{}, err
	}
	c.MatchIDs = parseMatchIDs(matchIDs)
	c.Highlight = highlight == 1
	return c, nil
}

func (db *appdbimpl) RecordCouponView(id int) error {
	if id <= 0 {
		return fmt.Errorf("invalid coupon id")
	}
	_, err := db.c.Exec(`UPDATE coupons SET total_views = total_views + 1 WHERE id=?`, id)
	return err
}

func (db *appdbimpl) ClaimCoupon(couponID int, userID *int, matchID int) (UserCoupon, error) {
	coupon, err := db.GetCouponByID(couponID)
	if err != nil {
		return UserCoupon{}, err
	}

	if strings.ToLower(coupon.Status) != "active" {
		return UserCoupon{}, ErrCouponUnavailable
	}

	now := time.Now()
	if coupon.StartDate != "" {
		if start, err := time.Parse(time.RFC3339, coupon.StartDate); err == nil && now.Before(start) {
			return UserCoupon{}, ErrCouponUnavailable
		}
	}
	if coupon.EndDate != "" {
		if end, err := time.Parse(time.RFC3339, coupon.EndDate); err == nil && now.After(end) {
			return UserCoupon{}, ErrCouponUnavailable
		}
	}

	if coupon.MaxUses > 0 {
		var used int
		if err := db.c.QueryRow(`SELECT COUNT(*) FROM user_coupons WHERE coupon_id = ?`, couponID).Scan(&used); err != nil {
			return UserCoupon{}, err
		}
		if used >= coupon.MaxUses {
			return UserCoupon{}, ErrCouponMaxReached
		}
	}

	code, err := generateCouponCode(db.c)
	if err != nil {
		return UserCoupon{}, err
	}

	var userValue interface{}
	if userID != nil {
		userValue = *userID
	}

	result, err := db.c.Exec(`INSERT INTO user_coupons (coupon_id, user_id, match_id, code, claimed_at, created_at) VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`, couponID, userValue, matchID, code)
	if err != nil {
		return UserCoupon{}, err
	}

	if _, err := db.c.Exec(`UPDATE coupons SET total_claims = total_claims + 1 WHERE id=?`, couponID); err != nil {
		return UserCoupon{}, err
	}

	claimID, err := result.LastInsertId()
	if err != nil {
		return UserCoupon{}, err
	}

	return db.getUserCouponByID(int(claimID))
}

func (db *appdbimpl) RedeemCoupon(code string, sponsorID int) (UserCoupon, error) {
	var uc UserCoupon
	var usedAt sql.NullString
	var userID sql.NullInt64
	var usedBy sql.NullInt64
	var couponID int
	err := db.c.QueryRow(`SELECT id, coupon_id, user_id, match_id, code, claimed_at, used_at, used_by_sponsor_id, created_at FROM user_coupons WHERE code=?`, strings.TrimSpace(code)).Scan(&uc.ID, &couponID, &userID, &uc.MatchID, &uc.Code, &uc.ClaimedAt, &usedAt, &usedBy, &uc.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return UserCoupon{}, sql.ErrNoRows
	}
	if err != nil {
		return UserCoupon{}, err
	}

	coupon, err := db.GetCouponByID(couponID)
	if err != nil {
		return UserCoupon{}, err
	}

	if coupon.MerchantID > 0 && coupon.MerchantID != sponsorID {
		return UserCoupon{}, ErrCouponWrongSponsor
	}
	if !strings.EqualFold(coupon.Status, "active") {
		return UserCoupon{}, ErrCouponUnavailable
	}
	if usedAt.Valid {
		return UserCoupon{}, ErrCouponAlreadyUsed
	}

	now := time.Now()
	if coupon.EndDate != "" {
		if end, err := time.Parse(time.RFC3339, coupon.EndDate); err == nil && now.After(end) {
			return UserCoupon{}, ErrCouponExpired
		}
	}
	if coupon.StartDate != "" {
		if start, err := time.Parse(time.RFC3339, coupon.StartDate); err == nil && now.Before(start) {
			return UserCoupon{}, ErrCouponUnavailable
		}
	}

	if coupon.MaxUses > 0 {
		var redeemed int
		if err := db.c.QueryRow(`SELECT COUNT(*) FROM user_coupons WHERE coupon_id=? AND used_at IS NOT NULL`, couponID).Scan(&redeemed); err != nil {
			return UserCoupon{}, err
		}
		if redeemed >= coupon.MaxUses {
			return UserCoupon{}, ErrCouponMaxReached
		}
	}

	_, err = db.c.Exec(`UPDATE user_coupons SET used_at=CURRENT_TIMESTAMP, used_by_sponsor_id=? WHERE id=?`, sponsorID, uc.ID)
	if err != nil {
		return UserCoupon{}, err
	}
	_, _ = db.c.Exec(`UPDATE coupons SET total_redemptions = total_redemptions + 1 WHERE id=?`, couponID)

	if userID.Valid {
		val := int(userID.Int64)
		uc.UserID = &val
	}
	if usedBy.Valid {
		val := int(usedBy.Int64)
		uc.UsedBySponsorID = &val
	}
	if usedAt.Valid {
		uc.UsedAt = &usedAt.String
	} else {
		nowStr := time.Now().UTC().Format(time.RFC3339)
		uc.UsedAt = &nowStr
	}
	uc.CouponID = couponID
	uc.Coupon = &coupon
	return uc, nil
}

func (db *appdbimpl) ListUserCoupons(userID *int, sponsorID int) ([]UserCoupon, error) {
	query := `SELECT id, coupon_id, user_id, match_id, code, claimed_at, used_at, used_by_sponsor_id, created_at FROM user_coupons WHERE 1=1`
	var args []interface{}
	if userID != nil {
		query += ` AND user_id = ?`
		args = append(args, *userID)
	}
	if sponsorID > 0 {
		query += ` AND coupon_id IN (SELECT id FROM coupons WHERE sponsor_id = ?)`
		args = append(args, sponsorID)
	}
	query += ` ORDER BY claimed_at DESC`

	rows, err := db.c.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var coupons []UserCoupon
	cachedCoupons := make(map[int]Coupon)
	for rows.Next() {
		var uc UserCoupon
		var user sql.NullInt64
		var usedAt sql.NullString
		var usedBy sql.NullInt64
		if err := rows.Scan(&uc.ID, &uc.CouponID, &user, &uc.MatchID, &uc.Code, &uc.ClaimedAt, &usedAt, &usedBy, &uc.CreatedAt); err != nil {
			return nil, err
		}
		if user.Valid {
			val := int(user.Int64)
			uc.UserID = &val
		}
		if usedAt.Valid {
			uc.UsedAt = &usedAt.String
		}
		if usedBy.Valid {
			val := int(usedBy.Int64)
			uc.UsedBySponsorID = &val
		}

		coupon, ok := cachedCoupons[uc.CouponID]
		if !ok {
			stored, err := db.GetCouponByID(uc.CouponID)
			if err != nil {
				return nil, err
			}
			cachedCoupons[uc.CouponID] = stored
			coupon = stored
		}
		uc.Coupon = &coupon

		if usedAt.Valid {
			continue
		}
		if strings.ToLower(coupon.Status) != "active" {
			continue
		}
		now := time.Now()
		if coupon.StartDate != "" {
			if start, err := time.Parse(time.RFC3339, coupon.StartDate); err == nil && now.Before(start) {
				continue
			}
		}
		if coupon.EndDate != "" {
			if end, err := time.Parse(time.RFC3339, coupon.EndDate); err == nil && now.After(end) {
				continue
			}
		}

		coupons = append(coupons, uc)
	}

	return coupons, rows.Err()
}

func (db *appdbimpl) getUserCouponByID(id int) (UserCoupon, error) {
	var uc UserCoupon
	var user sql.NullInt64
	var usedAt sql.NullString
	var usedBy sql.NullInt64
	err := db.c.QueryRow(`SELECT id, coupon_id, user_id, match_id, code, claimed_at, used_at, used_by_sponsor_id, created_at FROM user_coupons WHERE id=?`, id).Scan(&uc.ID, &uc.CouponID, &user, &uc.MatchID, &uc.Code, &uc.ClaimedAt, &usedAt, &usedBy, &uc.CreatedAt)
	if err != nil {
		return UserCoupon{}, err
	}
	if user.Valid {
		val := int(user.Int64)
		uc.UserID = &val
	}
	if usedAt.Valid {
		uc.UsedAt = &usedAt.String
	}
	if usedBy.Valid {
		val := int(usedBy.Int64)
		uc.UsedBySponsorID = &val
	}

	if coupon, err := db.GetCouponByID(uc.CouponID); err == nil {
		uc.Coupon = &coupon
	}
	return uc, nil
}

func joinMatchIDs(ids []int) string {
	if len(ids) == 0 {
		return ""
	}
	parts := make([]string, 0, len(ids))
	for _, id := range ids {
		if id > 0 {
			parts = append(parts, fmt.Sprintf("%d", id))
		}
	}
	return strings.Join(parts, ",")
}

func parseMatchIDs(raw string) []int {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	var ids []int
	for _, p := range parts {
		if p == "" {
			continue
		}
		if v, err := strconv.Atoi(strings.TrimSpace(p)); err == nil && v > 0 {
			ids = append(ids, v)
		}
	}
	return ids
}

func generateCouponCode(dbConn *sql.DB) (string, error) {
	for i := 0; i < couponCodeAttempts; i++ {
		buf := make([]byte, 6)
		if _, err := rand.Read(buf); err != nil {
			return "", err
		}
		code := strings.ToUpper(hex.EncodeToString(buf))
		var existing int
		if err := dbConn.QueryRow(`SELECT COUNT(*) FROM user_coupons WHERE code=?`, code).Scan(&existing); err != nil {
			return "", err
		}
		if existing == 0 {
			return code, nil
		}
	}
	return "", fmt.Errorf("unable to generate unique coupon code")
}

func ensureFanProfileTables(db *sql.DB) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS fan_profiles (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			organization_id INTEGER NOT NULL DEFAULT 0,
			nickname TEXT NOT NULL,
			gender TEXT,
			phone TEXT NOT NULL,
			phone_e164 TEXT,
			phone_verified_at TEXT,
			accepted_terms INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(organization_id, phone)
		);`,
		`CREATE TABLE IF NOT EXISTS fan_sessions (
			token TEXT PRIMARY KEY,
			fan_id INTEGER NOT NULL,
			device_id TEXT,
			created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
			last_seen_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (fan_id) REFERENCES fan_profiles(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS fan_wallets (
			fan_id INTEGER PRIMARY KEY,
			coins INTEGER NOT NULL DEFAULT 0,
			updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (fan_id) REFERENCES fan_profiles(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS guest_wallets (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			event_id INTEGER NOT NULL,
			organization_id INTEGER NOT NULL DEFAULT 0,
			device_id TEXT NOT NULL,
			coins INTEGER NOT NULL DEFAULT 0,
			updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(event_id, organization_id, device_id)
		);`,
		`CREATE TABLE IF NOT EXISTS fan_lottery_entries (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			event_id INTEGER NOT NULL,
			fan_id INTEGER NOT NULL,
			source TEXT NOT NULL DEFAULT 'manual',
			created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(event_id, fan_id),
			FOREIGN KEY (fan_id) REFERENCES fan_profiles(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS fan_reward_redemptions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			event_id INTEGER NOT NULL,
			fan_id INTEGER NOT NULL,
			reward_key TEXT NOT NULL,
			cost_coins INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (fan_id) REFERENCES fan_profiles(id) ON DELETE CASCADE
		);`,
		`ALTER TABLE fan_profiles ADD COLUMN phone_e164 TEXT;`,
		`ALTER TABLE fan_profiles ADD COLUMN phone_verified_at TEXT;`,
		`UPDATE fan_profiles SET phone_e164 = phone WHERE phone_e164 IS NULL OR TRIM(phone_e164) = '';`,
		`ALTER TABLE votes ADD COLUMN user_id INTEGER REFERENCES fan_profiles(id);`,
		`CREATE INDEX IF NOT EXISTS idx_fan_sessions_fan ON fan_sessions(fan_id);`,
		`CREATE INDEX IF NOT EXISTS idx_guest_wallets_device ON guest_wallets(event_id, organization_id, device_id);`,
		`CREATE INDEX IF NOT EXISTS idx_fan_wallets_coins ON fan_wallets(coins DESC);`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_fan_profiles_phone_e164_unique ON fan_profiles(phone_e164);`,
	}

	for _, stmt := range statements {
		if _, err := db.Exec(stmt); err != nil {
			if strings.Contains(err.Error(), "duplicate column name") {
				continue
			}
			return fmt.Errorf("error ensuring fan profile tables: %w", err)
		}
	}
	if _, err := db.Exec(`CREATE TRIGGER IF NOT EXISTS trg_fan_profiles_updated_at AFTER UPDATE ON fan_profiles BEGIN UPDATE fan_profiles SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id; END;`); err != nil {
		return fmt.Errorf("error ensuring fan profile trigger: %w", err)
	}
	return nil
}

func (db *appdbimpl) RegisterFan(input FanRegisterInput) (FanProfileSummary, error) {
	input.Phone = strings.TrimSpace(input.Phone)
	input.Nickname = strings.TrimSpace(input.Nickname)
	input.Gender = strings.TrimSpace(input.Gender)
	if input.Phone == "" || input.Nickname == "" {
		return FanProfileSummary{}, ErrInvalidSponsorData
	}

	tx, err := db.c.Begin()
	if err != nil {
		return FanProfileSummary{}, err
	}
	defer tx.Rollback()

	profileID := 0
	if input.SessionToken != "" {
		err = tx.QueryRow(`SELECT fan_id FROM fan_sessions WHERE token = ?`, input.SessionToken).Scan(&profileID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return FanProfileSummary{}, err
		}
	}

	if profileID == 0 {
		err = tx.QueryRow(`SELECT id FROM fan_profiles WHERE phone_e164 = ? OR phone = ? LIMIT 1`, input.Phone, input.Phone).Scan(&profileID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return FanProfileSummary{}, err
		}
	}

	if profileID > 0 {
		if _, err = tx.Exec(`UPDATE fan_profiles
			SET organization_id = ?, nickname = ?, gender = ?, phone = ?, phone_e164 = ?, accepted_terms = ?, updated_at = CURRENT_TIMESTAMP
			WHERE id = ?`,
			input.OrganizationID, input.Nickname, input.Gender, input.Phone, input.Phone, boolToInt(input.AcceptedTerms), profileID); err != nil {
			return FanProfileSummary{}, err
		}
	} else {
		res, execErr := tx.Exec(`INSERT INTO fan_profiles (organization_id, nickname, gender, phone, phone_e164, accepted_terms)
			VALUES (?, ?, ?, ?, ?, ?)`,
			input.OrganizationID, input.Nickname, input.Gender, input.Phone, input.Phone, boolToInt(input.AcceptedTerms))
		if execErr != nil {
			return FanProfileSummary{}, execErr
		}
		id, idErr := res.LastInsertId()
		if idErr != nil || id <= 0 {
			return FanProfileSummary{}, fmt.Errorf("unable to fetch inserted fan id: %w", idErr)
		}
		profileID = int(id)
	}

	var profile FanProfile
	var acceptedTerms int
	err = tx.QueryRow(`SELECT id, organization_id, nickname, gender, phone, accepted_terms, created_at, updated_at FROM fan_profiles WHERE id = ?`, profileID).
		Scan(&profile.ID, &profile.OrganizationID, &profile.Nickname, &profile.Gender, &profile.Phone, &acceptedTerms, &profile.CreatedAt, &profile.UpdatedAt)
	if err != nil {
		return FanProfileSummary{}, err
	}
	profile.AcceptedTerms = acceptedTerms == 1

	if _, err = tx.Exec(`INSERT INTO fan_wallets (fan_id, coins) VALUES (?, ?)
		ON CONFLICT(fan_id) DO UPDATE SET coins = fan_wallets.coins + excluded.coins, updated_at=CURRENT_TIMESTAMP`, profile.ID, nonNegativeInt(input.GuestCoins)); err != nil {
		return FanProfileSummary{}, err
	}

	if input.DeviceID != "" {
		if _, err = tx.Exec(`UPDATE votes SET user_id = ? WHERE device_id = ? AND (? = 0 OR event_id = ?)`, profile.ID, input.DeviceID, input.EventID, input.EventID); err != nil {
			return FanProfileSummary{}, err
		}
		if _, err = tx.Exec(`DELETE FROM guest_wallets WHERE device_id = ? AND (? = 0 OR event_id = ?)`, input.DeviceID, input.EventID, input.EventID); err != nil {
			return FanProfileSummary{}, err
		}
	}

	if input.SessionToken != "" {
		if _, err = tx.Exec(`INSERT INTO fan_sessions (token, fan_id, device_id) VALUES (?, ?, ?)
			ON CONFLICT(token) DO UPDATE SET fan_id=excluded.fan_id, device_id=excluded.device_id, last_seen_at=CURRENT_TIMESTAMP`, input.SessionToken, profile.ID, input.DeviceID); err != nil {
			return FanProfileSummary{}, err
		}
	}

	if input.EnterLottery && input.EventID > 0 {
		if _, err = tx.Exec(`INSERT OR IGNORE INTO fan_lottery_entries (event_id, fan_id, source) VALUES (?, ?, 'after_vote')`, input.EventID, profile.ID); err != nil {
			return FanProfileSummary{}, err
		}
	}

	if err = tx.Commit(); err != nil {
		return FanProfileSummary{}, err
	}

	wallet, err := db.getFanWallet(profile.ID)
	if err != nil {
		return FanProfileSummary{}, err
	}
	return FanProfileSummary{Profile: profile, Wallet: wallet}, nil
}

func (db *appdbimpl) getFanWallet(fanID int) (int, error) {
	var coins int
	err := db.c.QueryRow(`SELECT coins FROM fan_wallets WHERE fan_id = ?`, fanID).Scan(&coins)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	return nonNegativeInt(coins), err
}

func (db *appdbimpl) getFanSummaryByWhere(clause string, args ...interface{}) (FanProfileSummary, error) {
	var out FanProfileSummary
	var accepted int
	query := `SELECT p.id, p.organization_id, p.nickname, p.gender, p.phone, p.accepted_terms, p.created_at, p.updated_at
	FROM fan_profiles p ` + clause
	if err := db.c.QueryRow(query, args...).Scan(&out.Profile.ID, &out.Profile.OrganizationID, &out.Profile.Nickname, &out.Profile.Gender, &out.Profile.Phone, &accepted, &out.Profile.CreatedAt, &out.Profile.UpdatedAt); err != nil {
		return FanProfileSummary{}, err
	}
	out.Profile.AcceptedTerms = accepted == 1
	wallet, err := db.getFanWallet(out.Profile.ID)
	if err != nil {
		return FanProfileSummary{}, err
	}
	out.Wallet = wallet
	return out, nil
}

func (db *appdbimpl) GetFanByPhoneE164(phone string) (FanProfile, error) {
	phone = strings.TrimSpace(phone)
	if phone == "" {
		return FanProfile{}, sql.ErrNoRows
	}
	var out FanProfile
	var accepted int
	err := db.c.QueryRow(`SELECT id, organization_id, nickname, gender, phone, accepted_terms, created_at, updated_at
		FROM fan_profiles WHERE phone_e164 = ?`, phone).
		Scan(&out.ID, &out.OrganizationID, &out.Nickname, &out.Gender, &out.Phone, &accepted, &out.CreatedAt, &out.UpdatedAt)
	if err != nil {
		return FanProfile{}, err
	}
	out.AcceptedTerms = accepted == 1
	return out, nil
}

func (db *appdbimpl) CreateFanWithPhoneE164(phone string) (FanProfile, error) {
	phone = strings.TrimSpace(phone)
	if phone == "" {
		return FanProfile{}, ErrInvalidSponsorData
	}
	nickname := fmt.Sprintf("user_%s", strings.TrimPrefix(phone, "+"))
	if len(nickname) > 50 {
		nickname = nickname[:50]
	}
	_, err := db.c.Exec(`INSERT INTO fan_profiles (organization_id, nickname, gender, phone, phone_e164, accepted_terms)
		VALUES (0, ?, '', ?, ?, 0)`, nickname, phone, phone)
	if err != nil {
		return FanProfile{}, err
	}
	return db.GetFanByPhoneE164(phone)
}

func (db *appdbimpl) MarkFanPhoneVerified(phone string, verifiedAt time.Time) error {
	phone = strings.TrimSpace(phone)
	if phone == "" {
		return sql.ErrNoRows
	}
	res, err := db.c.Exec(`UPDATE fan_profiles SET phone_verified_at = ?, updated_at = CURRENT_TIMESTAMP WHERE phone_e164 = ?`, verifiedAt.UTC().Format(time.RFC3339), phone)
	if err != nil {
		return err
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (db *appdbimpl) UpsertFanSession(token string, fanID int, deviceID string) error {
	token = strings.TrimSpace(token)
	if token == "" || fanID <= 0 {
		return ErrInvalidSponsorData
	}
	_, err := db.c.Exec(`INSERT INTO fan_sessions (token, fan_id, device_id) VALUES (?, ?, ?)
		ON CONFLICT(token) DO UPDATE SET fan_id=excluded.fan_id, device_id=excluded.device_id, last_seen_at=CURRENT_TIMESTAMP`, token, fanID, strings.TrimSpace(deviceID))
	return err
}

func (db *appdbimpl) GetFanBySessionToken(token string) (FanProfileSummary, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return FanProfileSummary{}, sql.ErrNoRows
	}
	if _, err := db.c.Exec(`UPDATE fan_sessions SET last_seen_at = CURRENT_TIMESTAMP WHERE token = ?`, token); err != nil {
		return FanProfileSummary{}, err
	}
	return db.getFanSummaryByWhere(`JOIN fan_sessions s ON s.fan_id = p.id WHERE s.token = ?`, token)
}

func (db *appdbimpl) GetFanByDevice(eventID int, organizationID int, deviceID string) (FanProfileSummary, error) {
	_ = organizationID
	if strings.TrimSpace(deviceID) == "" {
		return FanProfileSummary{}, sql.ErrNoRows
	}
	if eventID > 0 {
		return db.getFanSummaryByWhere(`JOIN votes v ON v.user_id = p.id WHERE v.event_id = ? AND v.device_id = ? ORDER BY v.id DESC LIMIT 1`, eventID, deviceID)
	}
	return db.getFanSummaryByWhere(`JOIN fan_sessions s ON s.fan_id = p.id WHERE s.device_id = ? LIMIT 1`, deviceID)
}

func (db *appdbimpl) GetGuestCoins(eventID int, organizationID int, deviceID string) (int, error) {
	var coins int
	err := db.c.QueryRow(`SELECT coins FROM guest_wallets WHERE event_id = ? AND organization_id = ? AND device_id = ?`, eventID, organizationID, strings.TrimSpace(deviceID)).Scan(&coins)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	return nonNegativeInt(coins), err
}

func (db *appdbimpl) UpsertGuestCoins(eventID int, organizationID int, deviceID string, coins int) error {
	_, err := db.c.Exec(`INSERT INTO guest_wallets (event_id, organization_id, device_id, coins) VALUES (?, ?, ?, ?)
	ON CONFLICT(event_id, organization_id, device_id) DO UPDATE SET coins = excluded.coins, updated_at=CURRENT_TIMESTAMP`, eventID, organizationID, strings.TrimSpace(deviceID), nonNegativeInt(coins))
	return err
}

func (db *appdbimpl) GetFanLeaderboard(eventID int, organizationID int, limit int) ([]FanLeaderboardEntry, error) {
	if limit <= 0 {
		limit = 3
	}
	rows, err := db.c.Query(`SELECT p.id, p.nickname, w.coins, p.created_at
	FROM fan_profiles p
	JOIN fan_wallets w ON w.fan_id = p.id
	WHERE p.organization_id = ?
	ORDER BY w.coins DESC, p.id ASC
	LIMIT ?`, organizationID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []FanLeaderboardEntry{}
	rank := 1
	for rows.Next() {
		var e FanLeaderboardEntry
		if err := rows.Scan(&e.FanID, &e.Nickname, &e.Coins, &e.CreatedAt); err != nil {
			return nil, err
		}
		e.Rank = rank
		rank++
		out = append(out, e)
	}
	return out, rows.Err()
}

func (db *appdbimpl) GetFanRank(eventID int, organizationID int, fanID int) (FanLeaderboardEntry, error) {
	rows, err := db.c.Query(`SELECT p.id, p.nickname, w.coins
	FROM fan_profiles p
	JOIN fan_wallets w ON w.fan_id = p.id
	WHERE p.organization_id = ?
	ORDER BY w.coins DESC, p.id ASC`, organizationID)
	if err != nil {
		return FanLeaderboardEntry{}, err
	}
	defer rows.Close()
	rank := 1
	for rows.Next() {
		var e FanLeaderboardEntry
		if err := rows.Scan(&e.FanID, &e.Nickname, &e.Coins); err != nil {
			return FanLeaderboardEntry{}, err
		}
		if e.FanID == fanID {
			e.Rank = rank
			return e, nil
		}
		rank++
	}
	return FanLeaderboardEntry{}, sql.ErrNoRows
}

func (db *appdbimpl) ListFanRewardRedemptions(eventID int, fanID int) ([]FanRewardRedemption, error) {
	rows, err := db.c.Query(`SELECT id, event_id, fan_id, reward_key, cost_coins, created_at
	FROM fan_reward_redemptions
	WHERE fan_id = ? AND (? = 0 OR event_id = ?)
	ORDER BY id DESC`, fanID, eventID, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []FanRewardRedemption{}
	for rows.Next() {
		var redemption FanRewardRedemption
		if err := rows.Scan(&redemption.ID, &redemption.EventID, &redemption.FanID, &redemption.RewardKey, &redemption.CostCoins, &redemption.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, redemption)
	}

	return out, rows.Err()
}

func (db *appdbimpl) GetFanLotteryTicket(eventID int, fanID int) (EventTicket, error) {
	var ticket EventTicket
	err := db.c.QueryRow(`SELECT v.id, v.ticket_code, v.ticket_signature, v.player_id,
		IFNULL(p.first_name, ''), IFNULL(p.last_name, ''), v.created_at
	FROM votes v
	LEFT JOIN players p ON p.id = v.player_id
	WHERE v.event_id = ? AND v.user_id = ?
	ORDER BY v.id DESC
	LIMIT 1`, eventID, fanID).
		Scan(
			&ticket.VoteID,
			&ticket.TicketCode,
			&ticket.TicketSignature,
			&ticket.PlayerID,
			&ticket.PlayerFirstName,
			&ticket.PlayerLastName,
			&ticket.CreatedAt,
		)

	if err != nil {
		return EventTicket{}, err
	}

	return ticket, nil
}

func (db *appdbimpl) RecordFanLotteryEntry(eventID int, fanID int, source string) error {
	_, err := db.c.Exec(`INSERT OR IGNORE INTO fan_lottery_entries (event_id, fan_id, source) VALUES (?, ?, ?)`, eventID, fanID, strings.TrimSpace(source))
	return err
}

func (db *appdbimpl) RecordFanRewardRedemption(eventID int, fanID int, rewardKey string, costCoins int) error {
	tx, err := db.c.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	res, err := tx.Exec(`UPDATE fan_wallets SET coins = coins - ? WHERE fan_id = ? AND coins >= ?`, nonNegativeInt(costCoins), fanID, nonNegativeInt(costCoins))
	if err != nil {
		return err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return sql.ErrNoRows
	}
	if _, err = tx.Exec(`INSERT INTO fan_reward_redemptions (event_id, fan_id, reward_key, cost_coins) VALUES (?, ?, ?, ?)`, eventID, fanID, strings.TrimSpace(rewardKey), nonNegativeInt(costCoins)); err != nil {
		return err
	}
	return tx.Commit()
}

func nonNegativeInt(value int) int {
	if value < 0 {
		return 0
	}
	return value
}
