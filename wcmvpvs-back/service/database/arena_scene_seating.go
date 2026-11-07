package database

import (
	"database/sql"
	"encoding/json"
	"errors"
	"math"
)

const arenaVirtualSeatCount = 1330

type arenaSceneSector struct {
	Slug         string   `json:"slug"`
	Label        *string  `json:"label"`
	Side         string   `json:"side"`
	Type         string   `json:"sector_type"`
	IsActive     *bool    `json:"is_active"`
	Rows         float64  `json:"rows_count"`
	SeatsPerRow  float64  `json:"seats_per_row"`
	Length       float64  `json:"length"`
	FrontOffset  float64  `json:"front_offset"`
	CenterOffset *float64 `json:"center_offset"`
	BaseHeight   *float64 `json:"base_height"`
	RowDepth     float64  `json:"row_depth"`
	RowRise      *float64 `json:"row_rise"`
}

type arenaSceneSeat struct {
	side    string
	x, y, z float64
}

func configuredArenaSeats(tx *sql.Tx, organizationID int) ([]arenaSceneSeat, error) {
	var config string
	err := tx.QueryRow(`SELECT a.scene_config_json `+arenaForOrganization+` AND a.is_active = 1`, organizationID).Scan(&config)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return arenaSceneSeats(config), nil
}

func arenaSceneSeats(config string) []arenaSceneSeat {
	var scene struct {
		Version float64 `json:"schema_version"`
		Court   struct {
			Sport  string  `json:"sport"`
			Length float64 `json:"length"`
			Width  float64 `json:"width"`
		} `json:"court"`
		Layout struct {
			Length float64 `json:"length"`
			Width  float64 `json:"width"`
			Height float64 `json:"height"`
		} `json:"layout"`
		Sectors []*arenaSceneSector `json:"sectors"`
	}
	if err := json.Unmarshal([]byte(config), &scene); err != nil || scene.Version != 1 || scene.Court.Sport != "basket" {
		return nil
	}
	within := func(value, low, high float64) bool { return value >= low && value <= high }
	if !within(scene.Court.Length, 24, 36) || !within(scene.Court.Width, 13, 19) ||
		!within(scene.Layout.Length, scene.Court.Length+4, 65) || !within(scene.Layout.Width, scene.Court.Width+4, 55) ||
		!within(scene.Layout.Height, 6, 25) || len(scene.Sectors) > 40 {
		return nil
	}

	// Match the scene's row-major seat order and its stable mapping of virtual slots.
	seen := make(map[string]bool, len(scene.Sectors))
	var seats []arenaSceneSeat
	seatCount := 0
	for _, sector := range scene.Sectors {
		if sector == nil {
			return nil
		}
		if sector.IsActive != nil && !*sector.IsActive {
			continue
		}
		switch sector.Side {
		case "long_1", "long_2", "short_1", "short_2":
		default:
			return nil
		}
		if sector.Slug == "" || seen[sector.Slug] || sector.Label == nil ||
			!within(sector.Rows, 1, 35) || math.Trunc(sector.Rows) != sector.Rows ||
			!within(sector.SeatsPerRow, 1, 90) || math.Trunc(sector.SeatsPerRow) != sector.SeatsPerRow ||
			!within(sector.Length, 1, 45) || !within(sector.FrontOffset, 1, 32) || !within(sector.RowDepth, 0.2, 1.5) ||
			sector.CenterOffset == nil || !within(*sector.CenterOffset, -20, 20) ||
			sector.BaseHeight == nil || !within(*sector.BaseHeight, 0, 12) || sector.RowRise == nil || !within(*sector.RowRise, 0, 0.8) {
			return nil
		}
		seen[sector.Slug] = true
		seatCount += int(sector.Rows * sector.SeatsPerRow)
		if seatCount > 6000 {
			return nil
		}
		if sector.Type == "press" || sector.Type == "dirigenza" {
			continue
		}
		spacing := sector.Length / sector.SeatsPerRow
		rowRise := *sector.RowRise
		for row := 0; row < int(sector.Rows); row++ {
			y := *sector.BaseHeight + float64(row)*rowRise
			depth := sector.FrontOffset + float64(row)*sector.RowDepth
			for column := 0; column < int(sector.SeatsPerRow); column++ {
				offset := *sector.CenterOffset + (float64(column)+0.5)*spacing - sector.Length/2
				seat := arenaSceneSeat{side: sector.Side, y: y}
				switch sector.Side {
				case "long_1":
					seat.x, seat.z = offset, depth
				case "long_2":
					seat.x, seat.z = -offset, -depth
				case "short_1":
					seat.x, seat.z = -depth, offset
				case "short_2":
					seat.x, seat.z = depth, -offset
				}
				seats = append(seats, seat)
			}
		}
	}
	if len(seats) <= arenaVirtualSeatCount {
		return seats
	}
	slots := make([]arenaSceneSeat, arenaVirtualSeatCount)
	for slot := range slots {
		slots[slot] = seats[slot*len(seats)/arenaVirtualSeatCount]
	}
	return slots
}

func nearestConfiguredArenaSlot(seats []arenaSceneSeat, hostSlot int, used map[int]bool) int {
	host := seats[hostSlot]
	best := -1
	distance := math.Inf(1)
	for slot, seat := range seats {
		if used[slot] {
			continue
		}
		sameSide := seat.side == host.side
		bestSameSide := best >= 0 && seats[best].side == host.side
		if bestSameSide && !sameSide {
			continue
		}
		dx, dy, dz := seat.x-host.x, seat.y-host.y, seat.z-host.z
		squared := dx*dx + dy*dy + dz*dz
		if (sameSide && !bestSameSide) || squared < distance {
			best, distance = slot, squared
		}
	}
	return best
}

func nearestLegacyArenaSlot(hostSlot int, used map[int]bool) int {
	const seatsPerStand = arenaVirtualSeatCount / 2
	const seatsPerRow = 35
	best := -1
	distance := 1 << 30
	standStart := hostSlot / seatsPerStand * seatsPerStand
	for slot := standStart; slot < standStart+seatsPerStand; slot++ {
		if used[slot] {
			continue
		}
		dr := slot%seatsPerStand/seatsPerRow - hostSlot%seatsPerStand/seatsPerRow
		dc := slot%seatsPerRow - hostSlot%seatsPerRow
		d := dr*dr*80 + dc*dc*100
		if d < distance {
			best, distance = slot, d
		}
	}
	return best
}
