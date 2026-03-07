package api

import (
	"net/http/httptest"
	"testing"
)

func TestResolveCheckoutRedirectBaseConfiguredTakesPrecedence(t *testing.T) {
	req := httptest.NewRequest("POST", "http://api.internal/bar/checkout/session", nil)
	req.Header.Set("Origin", "https://origin.example")

	base := resolveCheckoutRedirectBase("https://configured.example/newui/", req)
	if base != "https://configured.example/newui" {
		t.Fatalf("unexpected base: %s", base)
	}
}

func TestResolveCheckoutRedirectBaseUsesOrigin(t *testing.T) {
	req := httptest.NewRequest("POST", "http://api.internal/bar/checkout/session", nil)
	req.Header.Set("Origin", "https://mvp.wearingcash.it")

	base := resolveCheckoutRedirectBase("", req)
	if base != "https://mvp.wearingcash.it/newui" {
		t.Fatalf("unexpected base: %s", base)
	}
}

func TestResolveCheckoutRedirectBaseUsesForwardedHostAndProto(t *testing.T) {
	req := httptest.NewRequest("POST", "http://api.internal/bar/checkout/session", nil)
	req.Header.Set("X-Forwarded-Host", "mvp.wearingcash.it")
	req.Header.Set("X-Forwarded-Proto", "https")

	base := resolveCheckoutRedirectBase("", req)
	if base != "https://mvp.wearingcash.it/newui" {
		t.Fatalf("unexpected base: %s", base)
	}
}

func TestNormalizeAbsoluteURLRejectsRelative(t *testing.T) {
	_, ok := normalizeAbsoluteURL("/newui")
	if ok {
		t.Fatal("expected relative URL to be rejected")
	}
}

func TestSanitizeHostDropsPortAndComma(t *testing.T) {
	host := sanitizeHost("mvp.wearingcash.it:443, proxy.local")
	if host != "mvp.wearingcash.it" {
		t.Fatalf("unexpected host: %s", host)
	}
}
