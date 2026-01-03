package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
)

type tycoonActionPayload struct {
	DeviceID   string `json:"deviceId"`
	UpgradeKey string `json:"upgradeKey,omitempty"`
	CouponID   string `json:"couponId,omitempty"`
}

type tycoonUpgradeView struct {
	Key         string  `json:"key"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Icon        string  `json:"icon"`
	Level       int     `json:"level"`
	NextCost    int     `json:"nextCost"`
	BonusType   string  `json:"bonusType"`
	BonusValue  float64 `json:"bonusValue"`
}

type tycoonCouponView struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	Cost       int     `json:"cost"`
	Redeemed   bool    `json:"redeemed"`
	RedeemedAt *string `json:"redeemedAt,omitempty"`
}

type tycoonConfigView struct {
	BaseTickMs        int64   `json:"baseTickMs"`
	OfflineCapSeconds int64   `json:"offlineCapSeconds"`
	ClickCooldownMs   int64   `json:"clickCooldownMs"`
	CostGrowthFactor  float64 `json:"costGrowthFactor"`
	BasePointsPerTick int     `json:"basePointsPerTick"`
}

type tycoonStateResponse struct {
	DeviceID       string              `json:"deviceId"`
	Points         int                 `json:"points"`
	PointsPerTick  int                 `json:"pointsPerTick"`
	PointsPerSec   float64             `json:"pointsPerSecond"`
	TickIntervalMs int64               `json:"tickIntervalMs"`
	LastAccrualAt  string              `json:"lastAccrualAt"`
	LastClickAt    *string             `json:"lastClickAt,omitempty"`
	Upgrades       []tycoonUpgradeView `json:"upgrades"`
	Coupons        []tycoonCouponView  `json:"coupons"`
	Config         tycoonConfigView    `json:"config"`
}

func (rt *_router) tycoonConfig() database.TycoonConfig {
	return database.DefaultTycoonConfig()
}

func (rt *_router) tycoonState(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	deviceID := strings.TrimSpace(r.URL.Query().Get("deviceId"))
	if deviceID == "" {
		deviceID = strings.TrimSpace(r.Header.Get("X-Device-ID"))
	}
	rt.handleTycoonStateResponse(w, ctx, deviceID, func(id string, cfg database.TycoonConfig) (database.TycoonState, error) {
		return rt.db.GetTycoonState(id, cfg)
	})
}

func (rt *_router) tycoonSync(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	var payload tycoonActionPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		ctx.Logger.WithError(err).Warn("invalid tycoon sync payload")
		_ = writeJSONMessage(w, http.StatusBadRequest, "payload non valido")
		return
	}
	rt.handleTycoonStateResponse(w, ctx, payload.DeviceID, func(id string, cfg database.TycoonConfig) (database.TycoonState, error) {
		return rt.db.GetTycoonState(id, cfg)
	})
}

func (rt *_router) tycoonClick(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	var payload tycoonActionPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		ctx.Logger.WithError(err).Warn("invalid tycoon click payload")
		_ = writeJSONMessage(w, http.StatusBadRequest, "payload non valido")
		return
	}
	rt.handleTycoonStateResponse(w, ctx, payload.DeviceID, func(id string, cfg database.TycoonConfig) (database.TycoonState, error) {
		return rt.db.RecordTycoonClick(id, cfg)
	})
}

func (rt *_router) tycoonBuyUpgrade(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	var payload tycoonActionPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		ctx.Logger.WithError(err).Warn("invalid tycoon upgrade payload")
		_ = writeJSONMessage(w, http.StatusBadRequest, "payload non valido")
		return
	}
	rt.handleTycoonStateResponse(w, ctx, payload.DeviceID, func(id string, cfg database.TycoonConfig) (database.TycoonState, error) {
		return rt.db.PurchaseTycoonUpgrade(id, payload.UpgradeKey, cfg)
	})
}

func (rt *_router) tycoonRedeemCoupon(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	var payload tycoonActionPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		ctx.Logger.WithError(err).Warn("invalid tycoon coupon payload")
		_ = writeJSONMessage(w, http.StatusBadRequest, "payload non valido")
		return
	}
	rt.handleTycoonStateResponse(w, ctx, payload.DeviceID, func(id string, cfg database.TycoonConfig) (database.TycoonState, error) {
		return rt.db.RedeemTycoonCoupon(id, payload.CouponID, cfg)
	})
}

func (rt *_router) handleTycoonStateResponse(w http.ResponseWriter, ctx reqcontext.RequestContext, deviceID string, exec func(deviceID string, cfg database.TycoonConfig) (database.TycoonState, error)) {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		_ = writeJSONMessage(w, http.StatusBadRequest, "deviceId mancante")
		return
	}

	cfg := rt.tycoonConfig()
	state, err := exec(deviceID, cfg)
	if err != nil {
		status, msg := mapTycoonError(err)
		_ = writeJSON(w, status, map[string]interface{}{
			"message": msg,
			"state":   buildTycoonStateResponse(state, cfg),
		})
		return
	}

	_ = writeJSON(w, http.StatusOK, buildTycoonStateResponse(state, cfg))
	ctx.Logger.WithFields(map[string]interface{}{
		"device_id": deviceID,
		"points":    state.Points,
	}).Info("tycoon state delivered")
}

func buildTycoonStateResponse(state database.TycoonState, cfg database.TycoonConfig) tycoonStateResponse {
	blueprints := database.DefaultTycoonUpgradeBlueprints()
	upgradeLevels := make(map[string]int, len(state.Upgrades))
	for _, upg := range state.Upgrades {
		upgradeLevels[upg.Key] = upg.Level
	}

	upgrades := make([]tycoonUpgradeView, 0, len(blueprints))
	for _, bp := range blueprints {
		level := upgradeLevels[bp.Key]
		nextCost := database.ComputeTycoonUpgradeCost(bp, level, cfg)
		upgrades = append(upgrades, tycoonUpgradeView{
			Key:         bp.Key,
			Name:        bp.Name,
			Description: bp.Description,
			Icon:        bp.Icon,
			Level:       level,
			NextCost:    nextCost,
			BonusType:   bp.BonusType,
			BonusValue:  bp.BonusValue,
		})
	}

	coupons := make([]tycoonCouponView, 0, len(database.DefaultTycoonCoupons()))
	redemptionMap := make(map[string]time.Time, len(state.Redemptions))
	for _, redemption := range state.Redemptions {
		redemptionMap[redemption.CouponID] = redemption.RedeemedAt
	}

	for _, coupon := range database.DefaultTycoonCoupons() {
		var redeemedAtPtr *string
		redeemed := false
		if redeemedAt, ok := redemptionMap[coupon.ID]; ok {
			val := redeemedAt.UTC().Format(time.RFC3339)
			redeemedAtPtr = &val
			redeemed = true
		}
		coupons = append(coupons, tycoonCouponView{
			ID:         coupon.ID,
			Name:       coupon.Name,
			Cost:       coupon.Cost,
			Redeemed:   redeemed,
			RedeemedAt: redeemedAtPtr,
		})
	}

	var lastClick *string
	if state.LastClickAt != nil {
		val := state.LastClickAt.UTC().Format(time.RFC3339)
		lastClick = &val
	}

	return tycoonStateResponse{
		DeviceID:       state.DeviceID,
		Points:         state.Points,
		PointsPerTick:  state.PointsPerTick,
		PointsPerSec:   state.PointsPerSec,
		TickIntervalMs: state.TickIntervalMs,
		LastAccrualAt:  state.LastAccrualAt.UTC().Format(time.RFC3339),
		LastClickAt:    lastClick,
		Upgrades:       upgrades,
		Coupons:        coupons,
		Config: tycoonConfigView{
			BaseTickMs:        cfg.BaseTickInterval.Milliseconds(),
			OfflineCapSeconds: int64(cfg.OfflineCap.Seconds()),
			ClickCooldownMs:   cfg.ClickCooldown.Milliseconds(),
			CostGrowthFactor:  cfg.CostGrowthFactor,
			BasePointsPerTick: cfg.BasePointsPerTick,
		},
	}
}

func mapTycoonError(err error) (int, string) {
	switch {
	case errors.Is(err, database.ErrTycoonDeviceRequired):
		return http.StatusBadRequest, "deviceId mancante"
	case errors.Is(err, database.ErrTycoonUnknownUpgrade):
		return http.StatusBadRequest, "upgrade non valido"
	case errors.Is(err, database.ErrTycoonUnknownCoupon):
		return http.StatusBadRequest, "coupon non valido"
	case errors.Is(err, database.ErrTycoonInsufficientPoints):
		return http.StatusBadRequest, "punti insufficienti"
	case errors.Is(err, database.ErrTycoonCouponAlreadyUsed):
		return http.StatusConflict, "coupon già riscattato"
	case errors.Is(err, database.ErrTycoonClickRateLimited):
		return http.StatusTooManyRequests, "click troppo rapido"
	default:
		return http.StatusInternalServerError, "errore interno"
	}
}
