package database

import (
	"database/sql"
	"errors"
	"math"
	"strings"
	"time"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/globaltime"
)

// Tycoon constants. Tune these to balance the idle game server-side.
const (
	tycoonBasePointsPerTick = 1
	tycoonBaseTickInterval  = 2 * time.Second
	tycoonOfflineCap        = 2 * time.Hour
	tycoonCostGrowthFactor  = 1.6
	tycoonClickCooldown     = 300 * time.Millisecond
	tycoonMinTickInterval   = 600 * time.Millisecond
)

// TycoonConfig exposes the tunable parameters for accrual and actions.
type TycoonConfig struct {
	BasePointsPerTick int
	BaseTickInterval  time.Duration
	OfflineCap        time.Duration
	CostGrowthFactor  float64
	ClickCooldown     time.Duration
}

// TycoonUpgradeBlueprint describes how an upgrade behaves and is priced.
type TycoonUpgradeBlueprint struct {
	Key         string
	Name        string
	Description string
	Icon        string
	BaseCost    int
	BonusType   string // flat | multiplier | speed
	BonusValue  float64
}

// TycoonCouponDefinition represents a redeemable coupon for the idle game.
type TycoonCouponDefinition struct {
	ID   string
	Name string
	Cost int
}

// TycoonUpgrade stores the current level for a device.
type TycoonUpgrade struct {
	DeviceID string
	Key      string
	Level    int
}

// TycoonCouponRedemption records a redeemed coupon for a device.
type TycoonCouponRedemption struct {
	ID         int
	DeviceID   string
	CouponID   string
	RedeemedAt time.Time
}

// TycoonState collects the state and computed fields required by the API.
type TycoonState struct {
	DeviceID       string
	Points         int
	LastAccrualAt  time.Time
	LastClickAt    *time.Time
	Upgrades       []TycoonUpgrade
	Redemptions    []TycoonCouponRedemption
	PointsPerTick  int
	TickIntervalMs int64
	PointsPerSec   float64
}

// DefaultTycoonConfig returns the tuned defaults for the idle tycoon.
func DefaultTycoonConfig() TycoonConfig {
	return TycoonConfig{
		BasePointsPerTick: tycoonBasePointsPerTick,
		BaseTickInterval:  tycoonBaseTickInterval,
		OfflineCap:        tycoonOfflineCap,
		CostGrowthFactor:  tycoonCostGrowthFactor,
		ClickCooldown:     tycoonClickCooldown,
	}
}

// DefaultTycoonUpgradeBlueprints lists available upgrades in display order.
func DefaultTycoonUpgradeBlueprints() []TycoonUpgradeBlueprint {
	return []TycoonUpgradeBlueprint{
		{Key: "tamburello", Name: "Tamburello Pro", Description: "+1 punto per tick", Icon: "🥁", BaseCost: 8, BonusType: "flat", BonusValue: 1},
		{Key: "megafono", Name: "Megafono Curva", Description: "+3 punti per tick", Icon: "📣", BaseCost: 18, BonusType: "flat", BonusValue: 3},
		{Key: "coro", Name: "Coro Coordinato", Description: "+12% produzione", Icon: "🎶", BaseCost: 36, BonusType: "multiplier", BonusValue: 0.12},
		{Key: "banda", Name: "Banda Ritmo", Description: "Tick più rapido (-5%)", Icon: "🥁🥁", BaseCost: 55, BonusType: "speed", BonusValue: 0.05},
	}
}

// DefaultTycoonCoupons lists the coupons supported by the backend for redemption.
func DefaultTycoonCoupons() []TycoonCouponDefinition {
	return []TycoonCouponDefinition{
		{ID: "sconto-10", Name: "Coupon Sconto 10%", Cost: 45},
		{ID: "drink-omaggio", Name: "Drink omaggio", Cost: 65},
		{ID: "merch-small", Name: "Mini merch curva", Cost: 95},
		{ID: "fan-kit", Name: "Fan kit completo", Cost: 140},
	}
}

// Tycoon errors exposed to the API layer for proper HTTP mapping.
var (
	ErrTycoonDeviceRequired     = errors.New("device id required")
	ErrTycoonUnknownUpgrade     = errors.New("unknown upgrade")
	ErrTycoonUnknownCoupon      = errors.New("unknown coupon")
	ErrTycoonInsufficientPoints = errors.New("insufficient points")
	ErrTycoonCouponAlreadyUsed  = errors.New("coupon already redeemed")
	ErrTycoonClickRateLimited   = errors.New("click too fast")
)

func blueprintMap(items []TycoonUpgradeBlueprint) map[string]TycoonUpgradeBlueprint {
	res := make(map[string]TycoonUpgradeBlueprint, len(items))
	for _, bp := range items {
		res[bp.Key] = bp
	}
	return res
}

func couponMap(items []TycoonCouponDefinition) map[string]TycoonCouponDefinition {
	res := make(map[string]TycoonCouponDefinition, len(items))
	for _, c := range items {
		res[c.ID] = c
	}
	return res
}

func computeTycoonPointsPerTick(upgrades []TycoonUpgrade, blueprints map[string]TycoonUpgradeBlueprint, cfg TycoonConfig) int {
	flatBonus := 0.0
	multiplierBonus := 0.0

	for _, upg := range upgrades {
		bp, ok := blueprints[upg.Key]
		if !ok {
			continue
		}
		switch bp.BonusType {
		case "flat":
			flatBonus += float64(upg.Level) * bp.BonusValue
		case "multiplier":
			multiplierBonus += float64(upg.Level) * bp.BonusValue
		}
	}

	flatBase := float64(cfg.BasePointsPerTick) + flatBonus
	multiplier := 1 + multiplierBonus
	value := flatBase * multiplier
	if value < 1 {
		return 1
	}
	return int(math.Round(value))
}

func computeTycoonTickInterval(upgrades []TycoonUpgrade, blueprints map[string]TycoonUpgradeBlueprint, cfg TycoonConfig) time.Duration {
	reduction := 0.0
	for _, upg := range upgrades {
		bp, ok := blueprints[upg.Key]
		if !ok || bp.BonusType != "speed" {
			continue
		}
		reduction += float64(upg.Level) * bp.BonusValue
	}
	if reduction > 0.5 {
		reduction = 0.5
	}
	interval := float64(cfg.BaseTickInterval) * (1 - reduction)
	if interval < float64(tycoonMinTickInterval) {
		interval = float64(tycoonMinTickInterval)
	}
	return time.Duration(interval)
}

func computeTycoonPointsPerSecond(pointsPerTick int, tickInterval time.Duration) float64 {
	if tickInterval <= 0 {
		return 0
	}
	value := float64(pointsPerTick) / tickInterval.Seconds()
	return math.Round(value*100) / 100
}

func ComputeTycoonUpgradeCost(bp TycoonUpgradeBlueprint, level int, cfg TycoonConfig) int {
	return int(math.Round(float64(bp.BaseCost) * math.Pow(cfg.CostGrowthFactor, float64(level))))
}

func shouldPersistTycoonState(err error) bool {
	return errors.Is(err, ErrTycoonInsufficientPoints) ||
		errors.Is(err, ErrTycoonUnknownUpgrade) ||
		errors.Is(err, ErrTycoonUnknownCoupon) ||
		errors.Is(err, ErrTycoonCouponAlreadyUsed) ||
		errors.Is(err, ErrTycoonClickRateLimited)
}

func (db *appdbimpl) GetTycoonState(deviceID string, cfg TycoonConfig) (TycoonState, error) {
	return db.withTycoonState(deviceID, cfg, nil)
}

func (db *appdbimpl) RecordTycoonClick(deviceID string, cfg TycoonConfig) (TycoonState, error) {
	return db.withTycoonState(deviceID, cfg, func(tx *sql.Tx, state *TycoonState, now time.Time, blueprints map[string]TycoonUpgradeBlueprint, coupons map[string]TycoonCouponDefinition) error {
		if state.LastClickAt != nil {
			if now.Sub(*state.LastClickAt) < cfg.ClickCooldown {
				return ErrTycoonClickRateLimited
			}
		}
		state.Points += 1
		state.LastClickAt = &now
		return db.saveTycoonState(tx, state)
	})
}

func (db *appdbimpl) PurchaseTycoonUpgrade(deviceID, upgradeKey string, cfg TycoonConfig) (TycoonState, error) {
	return db.withTycoonState(deviceID, cfg, func(tx *sql.Tx, state *TycoonState, now time.Time, blueprints map[string]TycoonUpgradeBlueprint, coupons map[string]TycoonCouponDefinition) error {
		bp, ok := blueprints[upgradeKey]
		if !ok {
			return ErrTycoonUnknownUpgrade
		}
		currentLevel := 0
		for _, upg := range state.Upgrades {
			if upg.Key == upgradeKey {
				currentLevel = upg.Level
				break
			}
		}

		cost := ComputeTycoonUpgradeCost(bp, currentLevel, cfg)
		if state.Points < cost {
			return ErrTycoonInsufficientPoints
		}

		nextLevel := currentLevel + 1
		state.Points -= cost
		updated := false
		for idx, upg := range state.Upgrades {
			if upg.Key == upgradeKey {
				state.Upgrades[idx].Level = nextLevel
				updated = true
				break
			}
		}
		if !updated {
			state.Upgrades = append(state.Upgrades, TycoonUpgrade{DeviceID: deviceID, Key: upgradeKey, Level: nextLevel})
		}

		if _, err := tx.Exec(`INSERT INTO tycoon_upgrades (device_id, upgrade_key, level, created_at, updated_at)
VALUES (?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT(device_id, upgrade_key) DO UPDATE SET level=excluded.level, updated_at=CURRENT_TIMESTAMP`, deviceID, upgradeKey, currentLevel+1); err != nil {
			return err
		}

		return db.saveTycoonState(tx, state)
	})
}

func (db *appdbimpl) RedeemTycoonCoupon(deviceID, couponID string, cfg TycoonConfig) (TycoonState, error) {
	return db.withTycoonState(deviceID, cfg, func(tx *sql.Tx, state *TycoonState, now time.Time, blueprints map[string]TycoonUpgradeBlueprint, coupons map[string]TycoonCouponDefinition) error {
		coupon, ok := coupons[couponID]
		if !ok {
			return ErrTycoonUnknownCoupon
		}

		for _, redemption := range state.Redemptions {
			if redemption.CouponID == couponID {
				return ErrTycoonCouponAlreadyUsed
			}
		}
		if state.Points < coupon.Cost {
			return ErrTycoonInsufficientPoints
		}
		state.Points -= coupon.Cost

		result, err := tx.Exec(`INSERT INTO tycoon_coupon_redemptions (device_id, coupon_id, redeemed_at) VALUES (?, ?, CURRENT_TIMESTAMP)`, deviceID, couponID)
		if err != nil {
			return err
		}

		redemption := TycoonCouponRedemption{DeviceID: deviceID, CouponID: couponID}
		if redemptionID, err := result.LastInsertId(); err == nil {
			redemption.ID = int(redemptionID)
		}
		redemption.RedeemedAt = now
		state.Redemptions = append(state.Redemptions, redemption)

		return db.saveTycoonState(tx, state)
	})
}

func (db *appdbimpl) withTycoonState(deviceID string, cfg TycoonConfig, mutate func(tx *sql.Tx, state *TycoonState, now time.Time, blueprints map[string]TycoonUpgradeBlueprint, coupons map[string]TycoonCouponDefinition) error) (TycoonState, error) {
	if strings.TrimSpace(deviceID) == "" {
		return TycoonState{}, ErrTycoonDeviceRequired
	}

	blueprints := blueprintMap(DefaultTycoonUpgradeBlueprints())
	coupons := couponMap(DefaultTycoonCoupons())
	now := globaltime.Now().UTC()

	tx, err := db.c.Begin()
	if err != nil {
		return TycoonState{}, err
	}
	defer tx.Rollback()

	state, err := db.loadTycoonState(tx, deviceID, now)
	if err != nil {
		return TycoonState{}, err
	}

	if err := db.applyTycoonAccrual(tx, &state, cfg, blueprints, now); err != nil {
		return TycoonState{}, err
	}

	if mutate != nil {
		if err := mutate(tx, &state, now, blueprints, coupons); err != nil {
			if shouldPersistTycoonState(err) {
				if errCommit := tx.Commit(); errCommit != nil {
					return TycoonState{}, errCommit
				}
				state.PointsPerTick = computeTycoonPointsPerTick(state.Upgrades, blueprints, cfg)
				tickInterval := computeTycoonTickInterval(state.Upgrades, blueprints, cfg)
				state.TickIntervalMs = tickInterval.Milliseconds()
				state.PointsPerSec = computeTycoonPointsPerSecond(state.PointsPerTick, tickInterval)
				return state, err
			}
			return TycoonState{}, err
		}
	}

	if err := tx.Commit(); err != nil {
		return TycoonState{}, err
	}

	state.PointsPerTick = computeTycoonPointsPerTick(state.Upgrades, blueprints, cfg)
	tickInterval := computeTycoonTickInterval(state.Upgrades, blueprints, cfg)
	state.TickIntervalMs = tickInterval.Milliseconds()
	state.PointsPerSec = computeTycoonPointsPerSecond(state.PointsPerTick, tickInterval)

	return state, nil
}

func (db *appdbimpl) loadTycoonState(tx *sql.Tx, deviceID string, now time.Time) (TycoonState, error) {
	var state TycoonState
	state.DeviceID = deviceID

	var lastAccrualRaw sql.NullString
	var lastClickRaw sql.NullString

	err := tx.QueryRow(`SELECT points, last_accrual_at, last_click_at FROM tycoon_state WHERE device_id=?`, deviceID).Scan(&state.Points, &lastAccrualRaw, &lastClickRaw)
	if errors.Is(err, sql.ErrNoRows) {
		if _, err := tx.Exec(`INSERT INTO tycoon_state (device_id, points, last_accrual_at, last_click_at) VALUES (?, 0, ?, NULL)`, deviceID, now.Format(time.RFC3339)); err != nil {
			return TycoonState{}, err
		}
		state.LastAccrualAt = now
	} else if err != nil {
		return TycoonState{}, err
	} else {
		if lastAccrualRaw.Valid {
			if parsed, perr := parseSQLiteTimestamp(lastAccrualRaw.String); perr == nil {
				state.LastAccrualAt = parsed
			}
		}
		if lastClickRaw.Valid {
			if parsed, perr := parseSQLiteTimestamp(lastClickRaw.String); perr == nil {
				state.LastClickAt = &parsed
			}
		}
		if state.LastAccrualAt.IsZero() {
			state.LastAccrualAt = now
		}
	}

	upgrades, err := db.loadTycoonUpgrades(tx, deviceID)
	if err != nil {
		return TycoonState{}, err
	}
	state.Upgrades = upgrades

	redemptions, err := db.loadTycoonRedemptions(tx, deviceID)
	if err != nil {
		return TycoonState{}, err
	}
	state.Redemptions = redemptions

	return state, nil
}

func (db *appdbimpl) loadTycoonUpgrades(tx *sql.Tx, deviceID string) ([]TycoonUpgrade, error) {
	rows, err := tx.Query(`SELECT device_id, upgrade_key, level FROM tycoon_upgrades WHERE device_id=?`, deviceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var upgrades []TycoonUpgrade
	for rows.Next() {
		var up TycoonUpgrade
		if err := rows.Scan(&up.DeviceID, &up.Key, &up.Level); err != nil {
			return nil, err
		}
		upgrades = append(upgrades, up)
	}
	return upgrades, rows.Err()
}

func (db *appdbimpl) loadTycoonRedemptions(tx *sql.Tx, deviceID string) ([]TycoonCouponRedemption, error) {
	rows, err := tx.Query(`SELECT id, device_id, coupon_id, redeemed_at FROM tycoon_coupon_redemptions WHERE device_id=?`, deviceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var redemptions []TycoonCouponRedemption
	for rows.Next() {
		var r TycoonCouponRedemption
		var redeemedAtRaw string
		if err := rows.Scan(&r.ID, &r.DeviceID, &r.CouponID, &redeemedAtRaw); err != nil {
			return nil, err
		}
		if parsed, perr := parseSQLiteTimestamp(redeemedAtRaw); perr == nil {
			r.RedeemedAt = parsed
		}
		redemptions = append(redemptions, r)
	}
	return redemptions, rows.Err()
}

func (db *appdbimpl) applyTycoonAccrual(tx *sql.Tx, state *TycoonState, cfg TycoonConfig, blueprints map[string]TycoonUpgradeBlueprint, now time.Time) error {
	if state.LastAccrualAt.IsZero() {
		state.LastAccrualAt = now
	}
	elapsed := now.Sub(state.LastAccrualAt)
	if elapsed < 0 {
		elapsed = 0
	}
	if cfg.OfflineCap > 0 && elapsed > cfg.OfflineCap {
		elapsed = cfg.OfflineCap
	}
	if elapsed > 0 {
		tickInterval := computeTycoonTickInterval(state.Upgrades, blueprints, cfg)
		if tickInterval <= 0 {
			tickInterval = cfg.BaseTickInterval
		}
		ticks := int(elapsed / tickInterval)
		if ticks > 0 {
			pointsPerTick := computeTycoonPointsPerTick(state.Upgrades, blueprints, cfg)
			state.Points += ticks * pointsPerTick
		}
	}
	state.LastAccrualAt = now
	return db.saveTycoonState(tx, state)
}

func (db *appdbimpl) saveTycoonState(tx *sql.Tx, state *TycoonState) error {
	lastClick := interface{}(nil)
	if state.LastClickAt != nil {
		lastClick = state.LastClickAt.Format(time.RFC3339)
	}
	_, err := tx.Exec(`UPDATE tycoon_state SET points=?, last_accrual_at=?, last_click_at=? WHERE device_id=?`, state.Points, state.LastAccrualAt.Format(time.RFC3339), lastClick, state.DeviceID)
	return err
}
