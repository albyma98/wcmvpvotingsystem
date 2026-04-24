package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
	"github.com/go-chi/chi/v5"
)

// validGameTypes is the closed set of accepted game_type values.
const (
	brandedGameResultIPLimit     = 20           // max submit per device per finestra
	brandedGameResultWindow      = time.Minute
)

var validGameTypes = map[string]struct{}{
	"tap_challenge": {},
	"memory_flash":  {},
	"sponsor_rush":  {},
	"stack_it":      {},
}

// validRewardTypes is the closed set of accepted reward_type values.
var validRewardTypes = map[string]struct{}{
	"coins":  {},
	"coupon": {},
	"none":   {},
}

// StackItConfig holds stack_it-specific tuning parameters.
type StackItConfig struct {
	BlockTextureURL       string   `json:"block_texture_url,omitempty"`
	BlockColors           []string `json:"block_colors,omitempty"`
	BackgroundImageURL    string   `json:"background_image_url,omitempty"`
	PerfectStackSoundURL  string   `json:"perfect_stack_sound_url,omitempty"`
	GameOverSoundURL      string   `json:"game_over_sound_url,omitempty"`
	RewardPerBlock        float64  `json:"reward_per_block"`
	PerfectBonusCoins     float64  `json:"perfect_bonus_coins"`
	InitialPendulumSpeedMs int     `json:"initial_pendulum_speed_ms"`
	SpeedCurve            string   `json:"speed_curve,omitempty"`
	CTALabel              string   `json:"cta_label,omitempty"`
	CTAURL                string   `json:"cta_url,omitempty"`
}

// BrandedGameConfig is the parsed representation of the branded_game_config JSON column.
type BrandedGameConfig struct {
	SponsorID       string         `json:"sponsor_id"`
	SponsorName     string         `json:"sponsor_name"`
	SponsorLogoURL  string         `json:"sponsor_logo_url"`
	PrimaryColor    string         `json:"primary_color"`
	SecondaryColor  string         `json:"secondary_color"`
	GameType        string         `json:"game_type"`
	CTALabel        string         `json:"cta_label,omitempty"`
	CTAURL          string         `json:"cta_url,omitempty"`
	RewardType      string         `json:"reward_type"`
	RewardCoins     int            `json:"reward_coins,omitempty"`
	MaxPlaysPerUser int            `json:"max_plays_per_user"`
	StackItConfig   *StackItConfig `json:"stack_it_config,omitempty"`
}

func parseBrandedGameConfig(raw string) (BrandedGameConfig, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return BrandedGameConfig{}, errors.New("branded_game_config is empty")
	}
	var cfg BrandedGameConfig
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return BrandedGameConfig{}, err
	}
	return cfg, nil
}

// validateBrandedGameConfig returns a non-nil error if cfg fails any business rule.
func validateBrandedGameConfig(cfg BrandedGameConfig) error {
	if strings.TrimSpace(cfg.SponsorID) == "" {
		return errors.New("sponsor_id is required")
	}
	if _, ok := validGameTypes[cfg.GameType]; !ok {
		return errors.New("game_type must be one of: tap_challenge, memory_flash, sponsor_rush, stack_it")
	}
	if _, ok := validRewardTypes[cfg.RewardType]; !ok {
		return errors.New("reward_type must be one of: coins, coupon, none")
	}
	if cfg.RewardType == "coins" && cfg.RewardCoins < 0 {
		return errors.New("reward_coins must be >= 0 when reward_type is coins")
	}
	if cfg.MaxPlaysPerUser < 1 {
		return errors.New("max_plays_per_user must be >= 1")
	}
	// Rifiuta javascript: e data: URL nel campo CTA — prevenzione XSS
	if u := strings.ToLower(strings.TrimSpace(cfg.CTAURL)); u != "" {
		if strings.HasPrefix(u, "javascript:") || strings.HasPrefix(u, "data:") {
			return errors.New("cta_url must be a valid http/https URL")
		}
	}
	return nil
}

// brandedGamePublicResponse is what the GET handler returns — no admin-only fields.
type brandedGamePublicResponse struct {
	SponsorID      string `json:"sponsor_id"`
	SponsorName    string `json:"sponsor_name"`
	SponsorLogoURL string `json:"sponsor_logo_url"`
	PrimaryColor   string `json:"primary_color"`
	SecondaryColor string `json:"secondary_color"`
	GameType       string `json:"game_type"`
	CTALabel       string `json:"cta_label,omitempty"`
	CTAURL         string `json:"cta_url,omitempty"`
	RewardType     string `json:"reward_type"`
	RewardCoins    int    `json:"reward_coins,omitempty"`
	CanPlay        bool   `json:"can_play"`
	PlaysUsed      int    `json:"plays_used"`
	PlaysRemaining int    `json:"plays_remaining"`
}

// GET /events/{eventId}/branded-game
func (rt *_router) getBrandedGame(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	eventID, err := strconv.Atoi(chi.URLParam(r, "eventId"))
	if err != nil || eventID <= 0 {
		_ = writeJSONMessage(w, http.StatusBadRequest, "event id non valido")
		return
	}

	event, err := rt.db.GetEventByID(eventID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			_ = writeJSONMessage(w, http.StatusNotFound, "evento non trovato")
			return
		}
		ctx.Logger.WithError(err).Error("cannot load event for branded game")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if !event.ShowBrandedGame {
		_ = writeJSONMessage(w, http.StatusNotFound, "branded game non disponibile per questo evento")
		return
	}

	cfg, err := parseBrandedGameConfig(event.BrandedGameConfig)
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot parse branded_game_config")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if err := validateBrandedGameConfig(cfg); err != nil {
		ctx.Logger.WithError(err).Error("invalid branded_game_config in db")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	deviceID := rt.deviceIDFromRequest(r)
	playsUsed := 0
	if deviceID != "" {
		playsUsed, err = rt.db.GetBrandedGameParticipationCount(eventID, deviceID)
		if err != nil {
			ctx.Logger.WithError(err).Error("cannot count branded game participations")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	remaining := cfg.MaxPlaysPerUser - playsUsed
	if remaining < 0 {
		remaining = 0
	}

	resp := brandedGamePublicResponse{
		SponsorID:      cfg.SponsorID,
		SponsorName:    cfg.SponsorName,
		SponsorLogoURL: cfg.SponsorLogoURL,
		PrimaryColor:   cfg.PrimaryColor,
		SecondaryColor: cfg.SecondaryColor,
		GameType:       cfg.GameType,
		CTALabel:       cfg.CTALabel,
		CTAURL:         cfg.CTAURL,
		RewardType:     cfg.RewardType,
		RewardCoins:    cfg.RewardCoins,
		CanPlay:        remaining > 0,
		PlaysUsed:      playsUsed,
		PlaysRemaining: remaining,
	}
	_ = writeJSON(w, http.StatusOK, resp)
}

// stackItPayload is the game-specific metadata sent by StackItGame.vue.
type stackItPayload struct {
	GameScore     int `json:"game_score"`
	PerfectStacks int `json:"perfect_stacks"`
	MaxHeight     int `json:"max_height"`
}

// POST /events/{eventId}/branded-game/result
type brandedGameResultRequest struct {
	Score      int             `json:"score"`
	DurationMs int             `json:"duration_ms"`
	Completed  bool            `json:"completed"`
	Payload    json.RawMessage `json:"payload"`
	SessionID  string          `json:"session_id"`
}

type brandedGameResultResponse struct {
	RewardedCoins  int  `json:"rewarded_coins"`
	RemainingPlays int  `json:"remaining_plays"`
}

func (rt *_router) postBrandedGameResult(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	eventID, err := strconv.Atoi(chi.URLParam(r, "eventId"))
	if err != nil || eventID <= 0 {
		_ = writeJSONMessage(w, http.StatusBadRequest, "event id non valido")
		return
	}

	deviceID := rt.deviceIDFromRequest(r)
	if deviceID == "" {
		_ = writeJSONMessage(w, http.StatusBadRequest, "device id mancante")
		return
	}

	// Rate limit: max brandedGameResultIPLimit submit per device per minuto
	rt.brandedGameRateMu.Lock()
	allowed := allowRate(rt.brandedGameRateByDevice, deviceID, brandedGameResultIPLimit, brandedGameResultWindow, time.Now())
	rt.brandedGameRateMu.Unlock()
	if !allowed {
		_ = writeJSONMessage(w, http.StatusTooManyRequests, "troppi tentativi, riprova tra poco")
		return
	}

	event, err := rt.db.GetEventByID(eventID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			_ = writeJSONMessage(w, http.StatusNotFound, "evento non trovato")
			return
		}
		ctx.Logger.WithError(err).Error("cannot load event for branded game result")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if !event.ShowBrandedGame {
		_ = writeJSONMessage(w, http.StatusNotFound, "branded game non disponibile per questo evento")
		return
	}

	cfg, err := parseBrandedGameConfig(event.BrandedGameConfig)
	if err != nil || validateBrandedGameConfig(cfg) != nil {
		ctx.Logger.WithError(err).Error("invalid branded_game_config")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	playsUsed, err := rt.db.GetBrandedGameParticipationCount(eventID, deviceID)
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot count branded game participations")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if playsUsed >= cfg.MaxPlaysPerUser {
		_ = writeJSONMessage(w, http.StatusConflict, "hai raggiunto il numero massimo di partite per questo evento")
		return
	}

	var req brandedGameResultRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64*1024)).Decode(&req); err != nil {
		_ = writeJSONMessage(w, http.StatusBadRequest, "payload non valido")
		return
	}

	payloadJSON := "{}"
	if len(req.Payload) > 0 {
		payloadJSON = string(req.Payload)
	}

	// Compute reward — stack_it uses per-block formula; other games use flat config reward.
	rewardedCoins := 0
	if req.Completed && cfg.RewardType == "coins" {
		if cfg.GameType == "stack_it" && cfg.StackItConfig != nil {
			var sip stackItPayload
			if len(req.Payload) > 0 {
				_ = json.Unmarshal(req.Payload, &sip)
			}
			perBlock := cfg.StackItConfig.RewardPerBlock
			if perBlock <= 0 {
				perBlock = 0.5
			}
			perfBonus := cfg.StackItConfig.PerfectBonusCoins
			if perfBonus <= 0 {
				perfBonus = 2
			}
			rewardedCoins = int(float64(sip.GameScore)*perBlock + float64(sip.PerfectStacks)*perfBonus)
		} else if cfg.RewardCoins > 0 {
			rewardedCoins = cfg.RewardCoins
		}
	}

	fanSessionToken := rt.fanSessionTokenFromRequest(r)
	var fanID *int
	if fanSessionToken != "" {
		if me, fanErr := rt.db.GetFanBySessionToken(fanSessionToken, deviceID); fanErr == nil {
			id := me.Profile.ID
			fanID = &id
		}
	}

	now := time.Now().UTC()
	participation := database.BrandedGameParticipation{
		EventID:       eventID,
		DeviceID:      deviceID,
		UserID:        fanID,
		SessionID:     strings.TrimSpace(req.SessionID),
		EndedAt:       now.Format(time.RFC3339),
		Score:         req.Score,
		Completed:     req.Completed,
		PayloadJSON:   payloadJSON,
		RewardedCoins: rewardedCoins,
	}

	created, err := rt.db.CreateBrandedGameParticipation(participation)
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot store branded game participation")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Credit coins
	if rewardedCoins > 0 {
		if fanID != nil {
			if coinErr := rt.db.AddFanWalletCoins(*fanID, rewardedCoins); coinErr != nil {
				ctx.Logger.WithError(coinErr).Error("cannot credit fan wallet coins for branded game")
			}
		} else if event.OrganizationID > 0 {
			if coinErr := rt.db.UpsertGuestCoins(eventID, event.OrganizationID, deviceID, rewardedCoins); coinErr != nil {
				ctx.Logger.WithError(coinErr).Error("cannot credit guest coins for branded game")
			}
		}
	}

	// Track events
	now2 := time.Now().UTC()
	// Build tracking metadata; include stack_it-specific fields when available.
	completedMeta := map[string]interface{}{
		"sponsor_id":     cfg.SponsorID,
		"game_type":      cfg.GameType,
		"score":          req.Score,
		"duration_ms":    req.DurationMs,
		"completed":      req.Completed,
		"rewarded_coins": rewardedCoins,
	}
	if cfg.GameType == "stack_it" && len(req.Payload) > 0 {
		var sip stackItPayload
		if err2 := json.Unmarshal(req.Payload, &sip); err2 == nil {
			completedMeta["perfect_stacks"] = sip.PerfectStacks
			completedMeta["max_height"] = sip.MaxHeight
		}
	}
	completedMetaJSON, _ := json.Marshal(completedMeta)

	trackItems := []database.TrackingEvent{
		{
			Name:         "branded_game.completed",
			Domain:       "branded_game",
			EventID:      eventID,
			DeviceID:     deviceID,
			SessionID:    strings.TrimSpace(req.SessionID),
			MetadataJSON: string(completedMetaJSON),
			OccurredAt:   now2.Format(time.RFC3339),
		},
	}
	if rewardedCoins > 0 {
		trackItems = append(trackItems, database.TrackingEvent{
			Name:      "branded_game.reward_claimed",
			Domain:    "branded_game",
			EventID:   eventID,
			DeviceID:  deviceID,
			SessionID: strings.TrimSpace(req.SessionID),
			MetadataJSON: func() string {
				m := map[string]interface{}{
					"sponsor_id":     cfg.SponsorID,
					"reward_type":    cfg.RewardType,
					"rewarded_coins": rewardedCoins,
				}
				b, _ := json.Marshal(m)
				return string(b)
			}(),
			OccurredAt: now2.Format(time.RFC3339),
		})
	}
	if trackErr := rt.db.RecordTrackingEvents(eventID, trackItems); trackErr != nil {
		ctx.Logger.WithError(trackErr).Warn("cannot record branded game tracking events")
	}

	remaining := cfg.MaxPlaysPerUser - (playsUsed + 1)
	if remaining < 0 {
		remaining = 0
	}

	_ = created // ID is stored; not needed in response
	_ = writeJSON(w, http.StatusOK, brandedGameResultResponse{
		RewardedCoins:  rewardedCoins,
		RemainingPlays: remaining,
	})
}

// stackItLeaderboardEntry is a single row in the Stack It top-10.
type stackItLeaderboardEntry struct {
	Score int  `json:"score"`
	IsMe  bool `json:"is_me"`
}

type stackItLeaderboardResponse struct {
	Entries []stackItLeaderboardEntry `json:"entries"`
}

// GET /events/{eventId}/branded-game/stackit-leaderboard
func (rt *_router) getStackItLeaderboard(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	eventID, err := strconv.Atoi(chi.URLParam(r, "eventId"))
	if err != nil || eventID <= 0 {
		_ = writeJSONMessage(w, http.StatusBadRequest, "event id non valido")
		return
	}

	deviceID := rt.deviceIDFromRequest(r)

	entries, err := rt.db.GetBrandedGameLeaderboard(eventID, 10)
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot load stackit leaderboard")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	resp := stackItLeaderboardResponse{Entries: make([]stackItLeaderboardEntry, 0, len(entries))}
	for _, e := range entries {
		resp.Entries = append(resp.Entries, stackItLeaderboardEntry{
			Score: e.Score,
			IsMe:  deviceID != "" && e.DeviceID == deviceID,
		})
	}

	_ = writeJSON(w, http.StatusOK, resp)
}
