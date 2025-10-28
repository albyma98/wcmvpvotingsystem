package api

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"
)

type deviceFingerprintPayload struct {
	Browser             string `json:"browser"`
	Platform            string `json:"platform"`
	Screen              string `json:"screen"`
	ColorDepth          int    `json:"color_depth"`
	Timezone            string `json:"timezone"`
	TimezoneOffset      int    `json:"timezone_offset"`
	DeviceMemory        string `json:"device_memory"`
	HardwareConcurrency int    `json:"hardware_concurrency"`
	Languages           string `json:"languages"`
	Graphics            string `json:"graphics"`
	TouchSupport        string `json:"touch_support"`
}

func (fp deviceFingerprintPayload) normalizedValues() map[string]string {
	values := map[string]string{
		"browser":              sanitizeFingerprintValue(fp.Browser),
		"platform":             sanitizeFingerprintValue(fp.Platform),
		"screen":               sanitizeFingerprintValue(fp.Screen),
		"color_depth":          sanitizeFingerprintValue(fmt.Sprint(fp.ColorDepth)),
		"timezone":             sanitizeFingerprintValue(fp.Timezone),
		"timezone_offset":      sanitizeFingerprintValue(fmt.Sprint(fp.TimezoneOffset)),
		"device_memory":        sanitizeFingerprintValue(fp.DeviceMemory),
		"hardware_concurrency": sanitizeFingerprintValue(fmt.Sprint(fp.HardwareConcurrency)),
		"languages":            sanitizeFingerprintValue(fp.Languages),
		"graphics":             sanitizeFingerprintValue(fp.Graphics),
		"touch_support":        sanitizeFingerprintValue(fp.TouchSupport),
	}
	if values["color_depth"] == "0" {
		values["color_depth"] = "unknown"
	}
	if values["hardware_concurrency"] == "0" {
		values["hardware_concurrency"] = "unknown"
	}
	if values["device_memory"] == "" || values["device_memory"] == "0" {
		values["device_memory"] = "unknown"
	}
	if values["screen"] == "x" || values["screen"] == "unknownxunknown" {
		values["screen"] = "unknown"
	}
	return values
}

func sanitizeFingerprintValue(raw string) string {
	trimmed := strings.TrimSpace(strings.ToLower(raw))
	if trimmed == "" || trimmed == "null" || trimmed == "undefined" {
		return "unknown"
	}
	return trimmed
}

func (fp deviceFingerprintPayload) signature() string {
	values := fp.normalizedValues()
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	builder := strings.Builder{}
	for idx, key := range keys {
		if idx > 0 {
			builder.WriteString("|")
		}
		builder.WriteString(key)
		builder.WriteString(":")
		builder.WriteString(values[key])
	}
	return builder.String()
}

func (fp deviceFingerprintPayload) validate() error {
	values := fp.normalizedValues()
	entropy := 0
	for _, value := range values {
		if value != "unknown" && value != "" {
			entropy++
		}
	}
	if entropy < 3 {
		return fmt.Errorf("fingerprint contains insufficient entropy")
	}
	return nil
}

func generateDailyFingerprintHash(eventID int, fp deviceFingerprintPayload, now time.Time) string {
	dayToken := now.UTC().Format("2006-01-02")
	base := fmt.Sprintf("v1|%d|%s|%s", eventID, dayToken, fp.signature())
	digest := sha256.Sum256([]byte(base))
	return hex.EncodeToString(digest[:])
}
