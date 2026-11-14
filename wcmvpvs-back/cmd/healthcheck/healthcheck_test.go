package main

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"testing/iotest"
)

type healthcheckTransport func(*http.Request) (*http.Response, error)

func (transport healthcheckTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	return transport(r)
}

func TestCheckHealth(t *testing.T) {
	for _, tc := range []struct {
		name      string
		status    int
		truncated bool
		wantError bool
	}{
		{name: "ready", status: http.StatusNoContent},
		{name: "ok", status: http.StatusOK},
		{name: "unavailable", status: http.StatusServiceUnavailable, wantError: true},
		{name: "truncated", status: http.StatusOK, truncated: true, wantError: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := &http.Client{Transport: healthcheckTransport(func(_ *http.Request) (*http.Response, error) {
				var body io.Reader = strings.NewReader("")
				if tc.truncated {
					body = io.MultiReader(strings.NewReader("short"), iotest.ErrReader(io.ErrUnexpectedEOF))
				}
				return &http.Response{Status: http.StatusText(tc.status), StatusCode: tc.status, Body: io.NopCloser(body)}, nil
			})}

			if err := checkHealth(client, "http://backend:3000/readiness"); (err != nil) != tc.wantError {
				t.Fatalf("checkHealth() = %v, want error %t", err, tc.wantError)
			}
		})
	}
}
