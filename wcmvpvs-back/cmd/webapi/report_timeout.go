package main

import (
	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api"
	"time"
)

// An analysis uses the report-specific budget, independently of the older AI
// service. Leave enough time to send the validated result or its fallback.
func apiServerTimeouts(cfg WebAPIConfiguration) (time.Duration, time.Duration) {
	read, write := cfg.Web.ReadTimeout, cfg.Web.WriteTimeout
	processing := cfg.AI.RequestTimeout
	if report := api.RecipientReportRequestTimeout(); report > processing {
		processing = report
	}
	minimum := processing + 5*time.Second
	if read < minimum {
		read = minimum
	}
	if write < minimum {
		write = minimum
	}
	return read, write
}
