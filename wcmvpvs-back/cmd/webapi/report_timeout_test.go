package main

import (
	"testing"
	"time"
)

func TestReportAnalysisFitsHTTPServerTimeout(t *testing.T) {
	var cfg WebAPIConfiguration
	cfg.Web.ReadTimeout, cfg.Web.WriteTimeout = 5*time.Second, 5*time.Second
	cfg.AI.RequestTimeout = 30 * time.Second
	t.Setenv("OPENAI_REPORT_TIMEOUT", "45s")
	read, write := apiServerTimeouts(cfg)
	if read != 50*time.Second || write != 50*time.Second {
		t.Fatalf("HTTP timeouts %s/%s truncate the report's 45s processing budget", read, write)
	}
	t.Setenv("OPENAI_REPORT_TIMEOUT", "90s")
	_, write = apiServerTimeouts(cfg)
	if write != 95*time.Second {
		t.Fatal("configured report timeout ignored", write)
	}
	cfg.Web.ReadTimeout, cfg.Web.WriteTimeout = 3*time.Minute, 3*time.Minute
	read, write = apiServerTimeouts(cfg)
	if read != 3*time.Minute || write != 3*time.Minute {
		t.Fatal("longer configured server timeouts reduced")
	}
}
