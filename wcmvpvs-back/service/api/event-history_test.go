package api

import "testing"

func TestSanitizeFilenameComponent(t *testing.T) {
	cases := map[string]string{
		"":                       "",
		"  Finale Scudetto  ":    "finale-scudetto",
		"Evento!!! MVP":          "evento-mvp",
		"  ---Multi   space--- ": "multi-space",
		"Serata Ćampioni 2024":   "serata-ćampioni-2024",
		"Nome_con_trattini":      "nome-con-trattini",
	}

	for input, expected := range cases {
		if got := sanitizeFilenameComponent(input); got != expected {
			t.Fatalf("sanitizeFilenameComponent(%q) = %q, want %q", input, got, expected)
		}
	}
}

func TestBuildHistoryReportFilename(t *testing.T) {
	entry := eventHistoryEntry{
		ID:            7,
		Title:         "Finale Scudetto",
		StartDateTime: "2024-06-01T19:30:00Z",
	}

	expected := "20240601_finale-scudetto_report.pdf"
	if got := buildHistoryReportFilename(entry); got != expected {
		t.Fatalf("buildHistoryReportFilename() = %q, want %q", got, expected)
	}

	entryNoDate := eventHistoryEntry{ID: 5, Title: ""}
	if got := buildHistoryReportFilename(entryNoDate); got != "evento-5_report.pdf" {
		t.Fatalf("buildHistoryReportFilename() fallback mismatch: got %q", got)
	}
}
