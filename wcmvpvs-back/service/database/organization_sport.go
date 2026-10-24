package database

import "strings"

func normalizeOrganizationSport(org *Organization, fallback string) error {
	org.Sport = strings.TrimSpace(org.Sport)
	if org.Sport == "" {
		org.Sport = fallback
	}
	switch org.Sport {
	case "", "Calcio", "Pallacanestro", "Pallavolo":
		return nil
	default:
		return ErrInvalidOrganizationData
	}
}
