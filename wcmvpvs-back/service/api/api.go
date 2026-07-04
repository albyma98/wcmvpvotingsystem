/*
Package api exposes the main API engine. All HTTP APIs are handled here - so-called "business logic" should be here, or
in a dedicated package (if that logic is complex enough).

To use this package, you should create a new instance with New() passing a valid Config. The resulting Router will have
the Router.Handler() function that returns a handler that can be used in a http.Server (or in other middlewares).

Example:

	// Create the API router
	apirouter, err := api.New(api.Config{
		Logger:   logger,
		Database: appdb,
	})
	if err != nil {
		logger.WithError(err).Error("error creating the API server instance")
		return fmt.Errorf("error creating the API server instance: %w", err)
	}
	router := apirouter.Handler()

	// ... other stuff here, like middleware chaining, etc.

	// Create the API server
	apiserver := http.Server{
		Addr:              cfg.Web.APIHost,
		Handler:           router,
		ReadTimeout:       cfg.Web.ReadTimeout,
		ReadHeaderTimeout: cfg.Web.ReadTimeout,
		WriteTimeout:      cfg.Web.WriteTimeout,
	}

	// Start the service listening for requests in a separate goroutine
	apiserver.ListenAndServe()

See the `main.go` file inside the `cmd/webapi` for a full usage example.
*/
package api

import (
	"errors"
	"sync"
	"time"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"
)

// Config is used to provide dependencies and configuration to the New function.
type Config struct {
	// Logger where log entries are sent
	Logger logrus.FieldLogger

	// Database is the instance of database.AppDatabase where data are saved
	Database database.AppDatabase

	// Secret is used to sign vote codes
	VoteSecret string

	// TicketValidationBaseURL is the public base URL used to generate ticket validation links
	TicketValidationBaseURL string

	TwilioAccountSID                  string
	TwilioAuthToken                   string
	TwilioVerifySID                   string
	TwilioMessagingServiceSID         string
	TwilioWhatsAppMessagingServiceSID string
	TwilioWinnerContentTemplateSID    string

	StripeSecretKey     string
	StripeWebhookSecret string
	StripeSuccessURL    string

	AIEnabled          bool
	AIProviderBaseURL  string
	AIAPIKey           string
	AIModel            string
	AIRequestTimeout   time.Duration
	AICacheTTL         time.Duration
	AIMaxPopupsSession int
	AIPopupCooldown    time.Duration
}

// Router is the package API interface representing an API handler builder
type Router interface {
	// Handler returns a chi router for APIs provided in this package
	Handler() chi.Router

	// Close terminates any resource used in the package
	Close() error
}

// New returns a new Router instance
func New(cfg Config) (Router, error) {
	// Check if the configuration is correct
	if cfg.Logger == nil {
		return nil, errors.New("logger is required")
	}
	if cfg.Database == nil {
		return nil, errors.New("database is required")
	}

	// Create a new router where we will register HTTP endpoints. The server will pass requests to this router to be
	// handled.
	router := chi.NewRouter()

	return &_router{
		router:                  router,
		baseLogger:              cfg.Logger,
		db:                      cfg.Database,
		VoteSecret:              cfg.VoteSecret,
		ticketValidationBaseURL: cfg.TicketValidationBaseURL,
		adminSessions:           map[string]adminSession{},
		partnerSessions:         map[string]adminSession{},
		sessionTimeout:          12 * time.Hour,
		voteRateByDevice:        map[string][]time.Time{},
		voteRateByIP:            map[string][]time.Time{},
		authRateByPhone:         map[string][]time.Time{},
		authRateByIP:            map[string][]time.Time{},
		loginRateByIP:           map[string][]time.Time{},
		loginRateByUser:         map[string][]time.Time{},
		brandedGameRateByDevice: map[string][]time.Time{},
		marketingRateByOrg:      map[int][]time.Time{},
		twilioVerify: newTwilioVerifyClient(twilioVerifyConfig{
			AccountSID: cfg.TwilioAccountSID,
			AuthToken:  cfg.TwilioAuthToken,
			ServiceSID: cfg.TwilioVerifySID,
		}),
		twilioMessaging: newTwilioMessagingClient(twilioMessagingConfig{
			AccountSID:                  cfg.TwilioAccountSID,
			AuthToken:                   cfg.TwilioAuthToken,
			MessagingServiceSID:         cfg.TwilioMessagingServiceSID,
			WhatsAppMessagingServiceSID: cfg.TwilioWhatsAppMessagingServiceSID,
			WhatsAppWinnerContentSID:    cfg.TwilioWinnerContentTemplateSID,
		}),
		stripeSecretKey:     cfg.StripeSecretKey,
		stripeWebhookSecret: cfg.StripeWebhookSecret,
		stripeSuccessURL:    cfg.StripeSuccessURL,
		aiService: newAIService(aiServiceConfig{
			Enabled:          cfg.AIEnabled,
			ProviderBaseURL:  cfg.AIProviderBaseURL,
			APIKey:           cfg.AIAPIKey,
			Model:            cfg.AIModel,
			RequestTimeout:   cfg.AIRequestTimeout,
			CacheTTL:         cfg.AICacheTTL,
			MaxPopupsSession: cfg.AIMaxPopupsSession,
			PopupCooldown:    cfg.AIPopupCooldown,
			Logger:           cfg.Logger,
		}),
		votesHub:       newSSEHub(),
		coinsHub:       newSSEHub(),
		barOrdersHub:   newSSEHub(),
		tapLiveHub:     newSSEHub(),
		tapLive:        newTapLiveManager(),
		tapLiveRematch: newTapLiveRematchManager(),

		// Tournament Mode: store SQLite (raw *sql.DB condiviso) + cache TTL +
		// hub SSE (push live al posto del polling, keyed per event ID).
		store:         NewStore(cfg.Database.SQLConn()),
		liveCache:     NewTTLCache(),
		tournamentHub: newSSEHub(),
	}, nil
}

type _router struct {
	router *chi.Mux

	// baseLogger is a logger for non-requests contexts, like goroutines or background tasks not started by a request.
	// Use context logger if available (e.g., in requests) instead of this logger.
	baseLogger logrus.FieldLogger

	db database.AppDatabase

	VoteSecret string

	ticketValidationBaseURL string

	adminSessionsMu sync.RWMutex
	adminSessions   map[string]adminSession
	sessionTimeout  time.Duration

	partnerSessionsMu sync.RWMutex
	partnerSessions   map[string]adminSession

	voteRateMu       sync.Mutex
	voteRateByDevice map[string][]time.Time
	voteRateByIP     map[string][]time.Time

	authRateMu      sync.Mutex
	authRateByPhone map[string][]time.Time
	authRateByIP    map[string][]time.Time

	loginRateMu      sync.Mutex
	loginRateByIP    map[string][]time.Time
	loginRateByUser  map[string][]time.Time

	brandedGameRateMu sync.Mutex
	brandedGameRateByDevice map[string][]time.Time

	marketingRateMu    sync.Mutex
	marketingRateByOrg map[int][]time.Time

	twilioVerify    *twilioVerifyClient
	twilioMessaging *twilioMessagingClient

	stripeSecretKey     string
	stripeWebhookSecret string
	stripeSuccessURL    string

	aiService *aiService

	votesHub       *sseHub
	coinsHub       *sseHub
	barOrdersHub   *sseHub
	tapLiveHub     *sseHub
	tapLive        *tapLiveManager
	tapLiveRematch *tapLiveRematchManager

	// Tournament Mode (modalità torneo): data layer + cache live + hub SSE.
	store         *Store
	liveCache     *TTLCache
	tournamentHub *sseHub
}

type adminSession struct {
	AdminID            int
	Username           string
	Role               string
	ExpiresAt          time.Time
	OrganizationID     int
	OrganizationSlug   string
	OrganizationTeamID int
}
