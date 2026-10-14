package database

import "strings"

const (
	DefaultOrganizationPrimaryColor   = "#e4212e"
	DefaultOrganizationSecondaryColor = "#f5b321"
	DefaultOrganizationTertiaryColor  = "#fff7f2"
)

func normalizeOrganizationColor(value, fallback string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return fallback, nil
	}
	if len(value) != 7 || value[0] != '#' {
		return "", ErrInvalidOrganizationData
	}
	for _, ch := range value[1:] {
		if !(ch >= '0' && ch <= '9' || ch >= 'a' && ch <= 'f') {
			return "", ErrInvalidOrganizationData
		}
	}
	return value, nil
}

func normalizeOrganizationColors(org *Organization, fallback Organization) error {
	var err error
	org.PrimaryColor, err = normalizeOrganizationColor(org.PrimaryColor, fallback.PrimaryColor)
	if err != nil {
		return err
	}
	org.SecondaryColor, err = normalizeOrganizationColor(org.SecondaryColor, fallback.SecondaryColor)
	if err != nil {
		return err
	}
	org.TertiaryColor, err = normalizeOrganizationColor(org.TertiaryColor, fallback.TertiaryColor)
	return err
}
