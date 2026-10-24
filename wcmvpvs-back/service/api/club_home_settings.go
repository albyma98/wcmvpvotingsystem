package api

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/mail"
	"net/url"
	"strings"
)

const maxClubIconBytes = 160 * 1024
const maxClubSettingsBytes = 2 * 1024 * 1024

func validateClubHomeSettings(site *ClubSite, teams []ClubTeam) error {
	if len(site.Contacts) > 50 {
		return fmt.Errorf("too_many_contacts")
	}
	if site.Contacts == nil {
		site.Contacts = []ClubContact{}
	}
	for i := range site.Contacts {
		c := &site.Contacts[i]
		c.Name = strings.TrimSpace(c.Name)
		c.Role = strings.TrimSpace(c.Role)
		c.Phone = strings.TrimSpace(c.Phone)
		c.Email = strings.TrimSpace(c.Email)
		c.Address = strings.TrimSpace(c.Address)
		c.Website = strings.TrimSpace(c.Website)
		if c.Name == "" || len(c.Name) > 200 || len(c.Role) > 200 || len(c.Phone) > 100 || len(c.Address) > 500 {
			return fmt.Errorf("invalid_contact")
		}
		if c.Email != "" {
			address, err := mail.ParseAddress(c.Email)
			if err != nil || address.Address != c.Email || len(c.Email) > 254 {
				return fmt.Errorf("invalid_contact_email")
			}
		}
		if c.Website != "" {
			u, err := url.Parse(c.Website)
			if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Hostname() == "" || len(c.Website) > 2000 {
				return fmt.Errorf("invalid_contact_website")
			}
		}
	}
	allowedTiles := map[string]bool{"matches": true, "announcements": true, "contacts": true}
	for _, team := range teams {
		allowedTiles[fmt.Sprintf("team-%d", team.ID)] = true
	}
	if site.TileIcons == nil {
		site.TileIcons = map[string]string{}
	}
	for key, icon := range site.TileIcons {
		if !allowedTiles[key] {
			// A deleted team must not prevent saving the remaining homepage.
			if strings.HasPrefix(key, "team-") {
				delete(site.TileIcons, key)
				continue
			}
			return fmt.Errorf("invalid_tile")
		}
		if icon == "" {
			delete(site.TileIcons, key)
			continue
		}
		if err := validateClubTileIcon(icon); err != nil {
			return err
		}
	}
	return nil
}

func validateClubTileIcon(icon string) error {
	if len(icon) > maxClubIconBytes {
		return fmt.Errorf("tile_icon_too_large")
	}
	parts := strings.SplitN(icon, ",", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid_tile_icon")
	}
	types := map[string]string{
		"data:image/png;base64":  "image/png",
		"data:image/jpeg;base64": "image/jpeg",
		"data:image/webp;base64": "image/webp",
		"data:image/gif;base64":  "image/gif",
	}
	mimeType, ok := types[parts[0]]
	if !ok {
		return fmt.Errorf("invalid_tile_icon")
	}
	content, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil || http.DetectContentType(content) != mimeType {
		return fmt.Errorf("invalid_tile_icon")
	}
	return nil
}
