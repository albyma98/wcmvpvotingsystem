package api

import (
	"encoding/json"
	"testing"
)

// ---------- validateBrandedGameConfig ----------

func validConfig() BrandedGameConfig {
	return BrandedGameConfig{
		SponsorID:       "sponsor-123",
		SponsorName:     "ACME",
		SponsorLogoURL:  "https://example.com/logo.png",
		PrimaryColor:    "#ff0000",
		SecondaryColor:  "#ffffff",
		GameType:        "tap_challenge",
		RewardType:      "coins",
		RewardCoins:     50,
		MaxPlaysPerUser: 1,
	}
}

func TestValidateBrandedGameConfig_Valid(t *testing.T) {
	if err := validateBrandedGameConfig(validConfig()); err != nil {
		t.Fatalf("expected valid config to pass, got: %v", err)
	}
}

func TestValidateBrandedGameConfig_MissingSponsorID(t *testing.T) {
	cfg := validConfig()
	cfg.SponsorID = ""
	if err := validateBrandedGameConfig(cfg); err == nil {
		t.Fatal("expected error for empty sponsor_id")
	}
}

func TestValidateBrandedGameConfig_InvalidGameType(t *testing.T) {
	cfg := validConfig()
	cfg.GameType = "unknown_game"
	if err := validateBrandedGameConfig(cfg); err == nil {
		t.Fatal("expected error for invalid game_type")
	}
}

func TestValidateBrandedGameConfig_AllValidGameTypes(t *testing.T) {
	types := []string{"tap_challenge", "memory_flash", "sponsor_rush"}
	for _, gt := range types {
		cfg := validConfig()
		cfg.GameType = gt
		if err := validateBrandedGameConfig(cfg); err != nil {
			t.Fatalf("game_type %q should be valid, got: %v", gt, err)
		}
	}
}

func TestValidateBrandedGameConfig_InvalidRewardType(t *testing.T) {
	cfg := validConfig()
	cfg.RewardType = "bitcoin"
	if err := validateBrandedGameConfig(cfg); err == nil {
		t.Fatal("expected error for invalid reward_type")
	}
}

func TestValidateBrandedGameConfig_NegativeCoins(t *testing.T) {
	cfg := validConfig()
	cfg.RewardType = "coins"
	cfg.RewardCoins = -1
	if err := validateBrandedGameConfig(cfg); err == nil {
		t.Fatal("expected error for negative reward_coins")
	}
}

func TestValidateBrandedGameConfig_ZeroCoinsAllowed(t *testing.T) {
	cfg := validConfig()
	cfg.RewardType = "coins"
	cfg.RewardCoins = 0
	if err := validateBrandedGameConfig(cfg); err != nil {
		t.Fatalf("reward_coins=0 should be allowed, got: %v", err)
	}
}

func TestValidateBrandedGameConfig_MaxPlaysZero(t *testing.T) {
	cfg := validConfig()
	cfg.MaxPlaysPerUser = 0
	if err := validateBrandedGameConfig(cfg); err == nil {
		t.Fatal("expected error for max_plays_per_user=0")
	}
}

func TestValidateBrandedGameConfig_RewardTypeNone(t *testing.T) {
	cfg := validConfig()
	cfg.RewardType = "none"
	cfg.RewardCoins = 0
	if err := validateBrandedGameConfig(cfg); err != nil {
		t.Fatalf("reward_type=none should be valid, got: %v", err)
	}
}

func TestValidateBrandedGameConfig_RewardTypeCoupon(t *testing.T) {
	cfg := validConfig()
	cfg.RewardType = "coupon"
	cfg.RewardCoins = 0
	if err := validateBrandedGameConfig(cfg); err != nil {
		t.Fatalf("reward_type=coupon should be valid, got: %v", err)
	}
}

// ---------- parseBrandedGameConfig ----------

func TestParseBrandedGameConfig_Valid(t *testing.T) {
	raw := `{
		"sponsor_id": "acme",
		"game_type": "memory_flash",
		"reward_type": "coins",
		"reward_coins": 30,
		"max_plays_per_user": 2
	}`
	cfg, err := parseBrandedGameConfig(raw)
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if cfg.SponsorID != "acme" {
		t.Fatalf("sponsor_id mismatch: %s", cfg.SponsorID)
	}
	if cfg.MaxPlaysPerUser != 2 {
		t.Fatalf("max_plays_per_user mismatch: %d", cfg.MaxPlaysPerUser)
	}
}

func TestParseBrandedGameConfig_EmptyString(t *testing.T) {
	_, err := parseBrandedGameConfig("")
	if err == nil {
		t.Fatal("expected error for empty raw config")
	}
}

func TestParseBrandedGameConfig_InvalidJSON(t *testing.T) {
	_, err := parseBrandedGameConfig("{not valid json}")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

// ---------- play-limit logic ----------

func TestRemainingPlaysCalculation(t *testing.T) {
	cases := []struct {
		maxPlays  int
		playsUsed int
		wantCan   bool
		wantLeft  int
	}{
		{maxPlays: 1, playsUsed: 0, wantCan: true, wantLeft: 1},
		{maxPlays: 1, playsUsed: 1, wantCan: false, wantLeft: 0},
		{maxPlays: 3, playsUsed: 2, wantCan: true, wantLeft: 1},
		{maxPlays: 3, playsUsed: 3, wantCan: false, wantLeft: 0},
		{maxPlays: 3, playsUsed: 5, wantCan: false, wantLeft: 0}, // over-limit guard
	}

	for _, tc := range cases {
		remaining := tc.maxPlays - tc.playsUsed
		if remaining < 0 {
			remaining = 0
		}
		canPlay := remaining > 0

		if canPlay != tc.wantCan {
			t.Errorf("maxPlays=%d playsUsed=%d: canPlay=%v want %v", tc.maxPlays, tc.playsUsed, canPlay, tc.wantCan)
		}
		if remaining != tc.wantLeft {
			t.Errorf("maxPlays=%d playsUsed=%d: remaining=%d want %d", tc.maxPlays, tc.playsUsed, remaining, tc.wantLeft)
		}
	}
}

// ---------- coin reward logic ----------

func TestCoinRewardOnlyWhenCompleted(t *testing.T) {
	cfg := validConfig() // reward_type=coins, reward_coins=50

	cases := []struct {
		completed     bool
		wantRewarded  int
	}{
		{completed: true, wantRewarded: 50},
		{completed: false, wantRewarded: 0},
	}

	for _, tc := range cases {
		rewarded := 0
		if tc.completed && cfg.RewardType == "coins" && cfg.RewardCoins > 0 {
			rewarded = cfg.RewardCoins
		}
		if rewarded != tc.wantRewarded {
			t.Errorf("completed=%v: rewarded=%d want %d", tc.completed, rewarded, tc.wantRewarded)
		}
	}
}

func TestCoinRewardZeroWhenRewardTypeNone(t *testing.T) {
	cfg := validConfig()
	cfg.RewardType = "none"
	cfg.RewardCoins = 100

	rewarded := 0
	if cfg.RewardType == "coins" && cfg.RewardCoins > 0 {
		rewarded = cfg.RewardCoins
	}
	if rewarded != 0 {
		t.Fatalf("reward_type=none should give 0 coins, got %d", rewarded)
	}
}

// ---------- XSS: cta_url validation ----------

func TestValidateBrandedGameConfig_RejectsJavascriptURL(t *testing.T) {
	cfg := validConfig()
	cfg.CTAURL = "javascript:alert(1)"
	if err := validateBrandedGameConfig(cfg); err == nil {
		t.Fatal("expected error for javascript: URL")
	}
}

func TestValidateBrandedGameConfig_RejectsDataURL(t *testing.T) {
	cfg := validConfig()
	cfg.CTAURL = "data:text/html,<script>alert(1)</script>"
	if err := validateBrandedGameConfig(cfg); err == nil {
		t.Fatal("expected error for data: URL")
	}
}

func TestValidateBrandedGameConfig_AcceptsHTTPSURL(t *testing.T) {
	cfg := validConfig()
	cfg.CTAURL = "https://example.com/promo"
	if err := validateBrandedGameConfig(cfg); err != nil {
		t.Fatalf("https URL should be valid, got: %v", err)
	}
}

func TestValidateBrandedGameConfig_AcceptsEmptyCTAURL(t *testing.T) {
	cfg := validConfig()
	cfg.CTAURL = ""
	if err := validateBrandedGameConfig(cfg); err != nil {
		t.Fatalf("empty cta_url should be valid, got: %v", err)
	}
}

// ---------- JSON round-trip for BrandedGameConfig ----------

func TestBrandedGameConfigJSONRoundTrip(t *testing.T) {
	original := BrandedGameConfig{
		SponsorID:       "test-sponsor",
		SponsorName:     "Test Sponsor",
		SponsorLogoURL:  "https://example.com/logo.svg",
		PrimaryColor:    "#123456",
		SecondaryColor:  "#abcdef",
		GameType:        "sponsor_rush",
		CTALabel:        "Scopri di più",
		CTAURL:          "https://example.com",
		RewardType:      "coins",
		RewardCoins:     100,
		MaxPlaysPerUser: 2,
	}

	b, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var roundTripped BrandedGameConfig
	if err := json.Unmarshal(b, &roundTripped); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if roundTripped.SponsorID != original.SponsorID ||
		roundTripped.GameType != original.GameType ||
		roundTripped.RewardCoins != original.RewardCoins ||
		roundTripped.MaxPlaysPerUser != original.MaxPlaysPerUser {
		t.Fatalf("round-trip mismatch: got %+v", roundTripped)
	}
}
