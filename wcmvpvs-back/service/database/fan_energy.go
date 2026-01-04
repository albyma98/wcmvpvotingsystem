package database

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/globaltime"
)

const (
	fanEnergyCap           = 999
	fanEnergyBaseGain      = 5
	fanEnergyBoostGain     = 20
	fanEnergyTapGain       = 1
	fanEnergyTick          = 5 * time.Second
	fanEnergyBoostDuration = 30 * time.Second
	fanEnergyBoostCooldown = 5 * time.Minute
	fanEnergyTapCooldown   = 2 * time.Second
)

type FanEnergyState struct {
	DeviceID         string
	Energy           int
	LastTs           time.Time
	BoostReadyAt     time.Time
	BoostActiveUntil time.Time
	LastTapAt        time.Time
	Tickets          int
}

var (
	ErrFanEnergyBoostNotReady      = errors.New("fan energy boost not ready")
	ErrFanEnergyInsufficientEnergy = errors.New("insufficient fan energy")
	ErrFanEnergyDeviceRequired     = errors.New("fan energy device id required")
	ErrFanEnergyTapCooldown        = errors.New("fan energy tap cooldown active")
)

func (db *appdbimpl) GetFanEnergyState(deviceID string) (FanEnergyState, error) {
	return db.withFanEnergyState(deviceID, nil)
}

func (db *appdbimpl) ActivateFanEnergyBoost(deviceID string) (FanEnergyState, error) {
	return db.withFanEnergyState(deviceID, func(state *FanEnergyState, now time.Time) error {
		if now.Before(state.BoostReadyAt) {
			return ErrFanEnergyBoostNotReady
		}
		state.BoostActiveUntil = now.Add(fanEnergyBoostDuration)
		state.BoostReadyAt = now.Add(fanEnergyBoostCooldown)
		state.LastTs = now
		return nil
	})
}

func (db *appdbimpl) ClaimFanEnergy(deviceID string, amount int) (FanEnergyState, error) {
	if amount <= 0 {
		amount = 100
	}
	return db.withFanEnergyState(deviceID, func(state *FanEnergyState, now time.Time) error {
		if state.Energy < amount {
			return ErrFanEnergyInsufficientEnergy
		}
		state.Energy -= amount
		state.Tickets += amount / 100
		state.LastTs = now
		return nil
	})
}

func (db *appdbimpl) TapFanEnergy(deviceID string) (FanEnergyState, error) {
	return db.withFanEnergyState(deviceID, func(state *FanEnergyState, now time.Time) error {
		if !state.LastTapAt.IsZero() && now.Sub(state.LastTapAt) < fanEnergyTapCooldown {
			return ErrFanEnergyTapCooldown
		}

		if state.Energy < fanEnergyCap {
			state.Energy += fanEnergyTapGain
			if state.Energy > fanEnergyCap {
				state.Energy = fanEnergyCap
			}
		}

		state.LastTapAt = now
		state.LastTs = now
		return nil
	})
}

func shouldPersistFanEnergy(err error) bool {
	return errors.Is(err, ErrFanEnergyBoostNotReady) ||
		errors.Is(err, ErrFanEnergyInsufficientEnergy) ||
		errors.Is(err, ErrFanEnergyTapCooldown)
}

func (db *appdbimpl) withFanEnergyState(deviceID string, mutate func(state *FanEnergyState, now time.Time) error) (FanEnergyState, error) {
	if strings.TrimSpace(deviceID) == "" {
		return FanEnergyState{}, ErrFanEnergyDeviceRequired
	}

	now := globaltime.Now().UTC()
	tx, err := db.c.Begin()
	if err != nil {
		return FanEnergyState{}, err
	}
	defer tx.Rollback()

	state, err := db.loadFanEnergyState(tx, deviceID, now)
	if err != nil {
		return FanEnergyState{}, err
	}

	if err := db.applyFanEnergyAccrual(&state, now); err != nil {
		return FanEnergyState{}, err
	}

	if mutate != nil {
		if err := mutate(&state, now); err != nil {
			if shouldPersistFanEnergy(err) {
				if errSave := db.saveFanEnergyState(tx, &state); errSave != nil {
					return FanEnergyState{}, errSave
				}
				if errCommit := tx.Commit(); errCommit != nil {
					return FanEnergyState{}, errCommit
				}
				return state, err
			}
			return FanEnergyState{}, err
		}
	}

	if err := db.saveFanEnergyState(tx, &state); err != nil {
		return FanEnergyState{}, err
	}

	if err := tx.Commit(); err != nil {
		return FanEnergyState{}, err
	}

	return state, nil
}

func (db *appdbimpl) loadFanEnergyState(tx *sql.Tx, deviceID string, now time.Time) (FanEnergyState, error) {
	var state FanEnergyState
	state.DeviceID = deviceID

	var lastTsRaw, readyRaw, activeRaw, tapRaw sql.NullString
	err := tx.QueryRow(`SELECT energy, last_ts, boost_ready_at, boost_active_until, tickets, last_tap_at FROM fan_energy WHERE device_id=?`, deviceID).
		Scan(&state.Energy, &lastTsRaw, &readyRaw, &activeRaw, &state.Tickets, &tapRaw)
	if errors.Is(err, sql.ErrNoRows) {
		state.Energy = 0
		state.LastTs = now
		state.BoostReadyAt = now
		state.BoostActiveUntil = time.Time{}
		state.Tickets = 0
		if _, err := tx.Exec(`INSERT INTO fan_energy (device_id, energy, last_ts, boost_ready_at, boost_active_until, tickets, last_tap_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			deviceID, state.Energy, now.Format(time.RFC3339), now.Format(time.RFC3339), nil, state.Tickets, nil); err != nil {
			return FanEnergyState{}, err
		}
		return state, nil
	} else if err != nil {
		return FanEnergyState{}, err
	}

	if lastTsRaw.Valid {
		if parsed, perr := parseSQLiteTimestamp(lastTsRaw.String); perr == nil {
			state.LastTs = parsed
		}
	}
	if readyRaw.Valid {
		if parsed, perr := parseSQLiteTimestamp(readyRaw.String); perr == nil {
			state.BoostReadyAt = parsed
		}
	}
	if tapRaw.Valid {
		if parsed, perr := parseSQLiteTimestamp(tapRaw.String); perr == nil {
			state.LastTapAt = parsed
		}
	}
	if activeRaw.Valid {
		if parsed, perr := parseSQLiteTimestamp(activeRaw.String); perr == nil {
			state.BoostActiveUntil = parsed
		}
	}

	if state.LastTs.IsZero() {
		state.LastTs = now
	}
	if state.BoostReadyAt.IsZero() {
		state.BoostReadyAt = now
	}

	return state, nil
}

func (db *appdbimpl) saveFanEnergyState(tx *sql.Tx, state *FanEnergyState) error {
	_, err := tx.Exec(`UPDATE fan_energy SET energy=?, last_ts=?, boost_ready_at=?, boost_active_until=?, tickets=?, last_tap_at=? WHERE device_id=?`,
		state.Energy,
		state.LastTs.Format(time.RFC3339),
		state.BoostReadyAt.Format(time.RFC3339),
		nullableTime(state.BoostActiveUntil),
		state.Tickets,
		nullableTime(state.LastTapAt),
		state.DeviceID,
	)
	return err
}

func nullableTime(t time.Time) interface{} {
	if t.IsZero() {
		return nil
	}
	return t.Format(time.RFC3339)
}

func (db *appdbimpl) applyFanEnergyAccrual(state *FanEnergyState, now time.Time) error {
	if !now.After(state.LastTs) {
		state.LastTs = now
		return nil
	}

	gained := computeFanEnergyGain(state.LastTs, now, state.BoostActiveUntil)
	if gained > 0 {
		state.Energy += gained
		if state.Energy > fanEnergyCap {
			state.Energy = fanEnergyCap
		}
	}
	state.LastTs = now
	return nil
}

func computeFanEnergyGain(start, end time.Time, boostActiveUntil time.Time) int {
	if !end.After(start) {
		return 0
	}
	totalGain := 0

	boostStart := boostActiveUntil.Add(-fanEnergyBoostDuration)
	// boost window overlap
	if !boostActiveUntil.IsZero() && boostActiveUntil.After(start) {
		windowStart := maxTime(start, boostStart)
		windowEnd := minTime(end, boostActiveUntil)
		if windowEnd.After(windowStart) {
			boostTicks := int(windowEnd.Sub(windowStart) / fanEnergyTick)
			totalGain += boostTicks * fanEnergyBoostGain
		}
	}

	// base gain before boost window
	baseStart := start
	baseEnd := end
	if !boostActiveUntil.IsZero() && boostActiveUntil.After(start) {
		windowStart := maxTime(start, boostStart)
		windowEnd := minTime(end, boostActiveUntil)
		if windowStart.After(baseStart) {
			baseEndSegment := minTime(windowStart, end)
			if baseEndSegment.After(baseStart) {
				baseTicks := int(baseEndSegment.Sub(baseStart) / fanEnergyTick)
				totalGain += baseTicks * fanEnergyBaseGain
			}
		}
		if end.After(windowEnd) && windowEnd.After(start) {
			baseTicks := int(end.Sub(windowEnd) / fanEnergyTick)
			totalGain += baseTicks * fanEnergyBaseGain
		}
	} else {
		baseTicks := int(baseEnd.Sub(baseStart) / fanEnergyTick)
		totalGain += baseTicks * fanEnergyBaseGain
	}

	return totalGain
}

func maxTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}
