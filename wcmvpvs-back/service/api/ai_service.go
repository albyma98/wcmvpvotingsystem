package api

import (
	"bytes"
	"context"
	"crypto/sha1"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
	"github.com/sirupsen/logrus"
)

type aiServiceConfig struct {
	Enabled          bool
	ProviderBaseURL  string
	APIKey           string
	Model            string
	RequestTimeout   time.Duration
	CacheTTL         time.Duration
	MaxPopupsSession int
	PopupCooldown    time.Duration
	Logger           logrus.FieldLogger
}

type aiService struct {
	cfg    aiServiceConfig
	client *http.Client
	mu     sync.Mutex
	cache  map[string]cachedAIResult
}

type cachedAIResult struct {
	ExpiresAt time.Time
	Payload   []byte
}

type aiUpsellRequest struct {
	UserID                int              `json:"user_id,omitempty"`
	SessionID             string           `json:"session_id,omitempty"`
	Trigger               string           `json:"trigger"`
	EventID               int              `json:"event_id,omitempty"`
	EventPhase            string           `json:"event_phase,omitempty"`
	AdminPriorityProducts []string         `json:"admin_priority_products,omitempty"`
	Cart                  []aiCartItem     `json:"cart"`
	PurchaseHistory       []string         `json:"purchase_history,omitempty"`
	AvailableProducts     []aiProductInput `json:"available_products"`
}

type aiCartItem struct {
	ProductID   string `json:"product_id,omitempty"`
	ProductName string `json:"product_name"`
	Category    string `json:"category,omitempty"`
	Quantity    int    `json:"quantity"`
}

type aiProductInput struct {
	ProductID  string `json:"product_id,omitempty"`
	Name       string `json:"name"`
	Category   string `json:"category,omitempty"`
	Available  bool   `json:"available"`
	Visible    bool   `json:"visible"`
	Priority   bool   `json:"priority,omitempty"`
	PriceCents int    `json:"price_cents,omitempty"`
}

type aiUpsellResponse struct {
	Suggestions []aiUpsellSuggestion `json:"suggestions"`
	Source      string               `json:"source"`
	Interaction int                  `json:"interaction_id,omitempty"`
}

type aiUpsellSuggestion struct {
	ProductName   string `json:"product_name"`
	ProductID     string `json:"product_id,omitempty"`
	Reason        string `json:"reason"`
	MarketingText string `json:"marketing_text"`
	Priority      int    `json:"priority"`
}

type aiPopupRequest struct {
	UserID              int                      `json:"user_id,omitempty"`
	SessionID           string                   `json:"session_id,omitempty"`
	TriggerType         string                   `json:"trigger_type"`
	EventID             int                      `json:"event_id,omitempty"`
	EventPhase          string                   `json:"event_phase,omitempty"`
	Objective           string                   `json:"objective"`
	SessionsCount       int                      `json:"sessions_count,omitempty"`
	GamesPlayed         int                      `json:"games_played,omitempty"`
	Coins               int                      `json:"coins,omitempty"`
	LastGame            string                   `json:"last_game,omitempty"`
	LastPurchase        string                   `json:"last_purchase,omitempty"`
	InactiveSeconds     int                      `json:"inactive_seconds,omitempty"`
	CartItemsCount      int                      `json:"cart_items_count,omitempty"`
	CartTotalCents      int                      `json:"cart_total_cents,omitempty"`
	PopupHistorySession []map[string]interface{} `json:"popup_history_session,omitempty"`
	Extra               map[string]interface{}   `json:"extra,omitempty"`
}

type aiPopupResponse struct {
	PopupTitle    string `json:"popup_title"`
	PopupBody     string `json:"popup_body"`
	CTA           string `json:"cta_text"`
	Tone          string `json:"tone"`
	UrgencyLevel  string `json:"urgency_level"`
	Source        string `json:"source"`
	InteractionID int    `json:"interaction_id,omitempty"`
	CooldownSecs  int    `json:"cooldown_seconds,omitempty"`
}

type aiInteractionTrackRequest struct {
	Outcome    string `json:"outcome"`
	Trigger    string `json:"trigger,omitempty"`
	SessionID  string `json:"session_id,omitempty"`
	OccurredAt string `json:"occurred_at,omitempty"`
}

type aiProviderResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func newAIService(cfg aiServiceConfig) *aiService {
	if cfg.RequestTimeout <= 0 {
		cfg.RequestTimeout = 4 * time.Second
	}
	if cfg.CacheTTL <= 0 {
		cfg.CacheTTL = 90 * time.Second
	}
	if cfg.MaxPopupsSession <= 0 {
		cfg.MaxPopupsSession = 3
	}
	if cfg.PopupCooldown <= 0 {
		cfg.PopupCooldown = 8 * time.Minute
	}
	if cfg.Logger == nil {
		cfg.Logger = logrus.New()
	}
	return &aiService{cfg: cfg, client: &http.Client{Timeout: cfg.RequestTimeout}, cache: map[string]cachedAIResult{}}
}

func (svc *aiService) enabled() bool {
	return svc != nil && svc.cfg.Enabled
}

func (svc *aiService) GenerateUpsellSuggestions(ctx context.Context, req aiUpsellRequest) aiUpsellResponse {
	fallback := svc.fallbackUpsells(req)
	if !svc.enabled() || strings.TrimSpace(svc.cfg.APIKey) == "" {
		fallback.Source = "fallback"
		return fallback
	}
	cacheKey := svc.cacheKey("upsell", req)
	if payload, ok := svc.getCache(cacheKey); ok {
		var resp aiUpsellResponse
		if json.Unmarshal(payload, &resp) == nil {
			resp.Source = "llm_cache"
			return resp
		}
	}

	prompt := svc.buildUpsellPrompt(req)
	content, err := svc.callProvider(ctx, prompt)
	if err != nil {
		svc.cfg.Logger.WithError(err).Warn("ai upsell provider failed")
		fallback.Source = "fallback"
		return fallback
	}

	var parsed struct {
		Suggestions []aiUpsellSuggestion `json:"suggestions"`
	}
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		svc.cfg.Logger.WithError(err).Warn("ai upsell invalid json")
		fallback.Source = "fallback"
		return fallback
	}

	validated := svc.validateUpsells(req, parsed.Suggestions)
	if len(validated) == 0 {
		fallback.Source = "fallback"
		return fallback
	}
	resp := aiUpsellResponse{Suggestions: validated, Source: "llm"}
	if payload, err := json.Marshal(resp); err == nil {
		svc.setCache(cacheKey, payload)
	}
	return resp
}

func (svc *aiService) GeneratePopupMessage(ctx context.Context, req aiPopupRequest) aiPopupResponse {
	fallback := svc.fallbackPopup(req)
	if !svc.enabled() || strings.TrimSpace(svc.cfg.APIKey) == "" {
		fallback.Source = "fallback"
		fallback.CooldownSecs = int(svc.cfg.PopupCooldown.Seconds())
		return fallback
	}
	cacheKey := svc.cacheKey("popup", req)
	if payload, ok := svc.getCache(cacheKey); ok {
		var resp aiPopupResponse
		if json.Unmarshal(payload, &resp) == nil {
			resp.Source = "llm_cache"
			resp.CooldownSecs = int(svc.cfg.PopupCooldown.Seconds())
			return resp
		}
	}
	content, err := svc.callProvider(ctx, svc.buildPopupPrompt(req))
	if err != nil {
		svc.cfg.Logger.WithError(err).Warn("ai popup provider failed")
		fallback.Source = "fallback"
		fallback.CooldownSecs = int(svc.cfg.PopupCooldown.Seconds())
		return fallback
	}
	var parsed struct {
		PopupTitle   string `json:"popup_title"`
		PopupBody    string `json:"popup_body"`
		CTAText      string `json:"cta_text"`
		Tone         string `json:"tone"`
		UrgencyLevel string `json:"urgency_level"`
	}
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		svc.cfg.Logger.WithError(err).Warn("ai popup invalid json")
		fallback.Source = "fallback"
		fallback.CooldownSecs = int(svc.cfg.PopupCooldown.Seconds())
		return fallback
	}
	resp := aiPopupResponse{
		PopupTitle:   sanitizeAIText(parsed.PopupTitle, 50),
		PopupBody:    sanitizeAIText(parsed.PopupBody, 140),
		CTA:          sanitizeAIText(parsed.CTAText, 28),
		Tone:         sanitizeAIText(parsed.Tone, 16),
		UrgencyLevel: sanitizeAIText(parsed.UrgencyLevel, 16),
		Source:       "llm",
		CooldownSecs: int(svc.cfg.PopupCooldown.Seconds()),
	}
	if resp.PopupTitle == "" || resp.PopupBody == "" || resp.CTA == "" {
		fallback.Source = "fallback"
		fallback.CooldownSecs = int(svc.cfg.PopupCooldown.Seconds())
		return fallback
	}
	if payload, err := json.Marshal(resp); err == nil {
		svc.setCache(cacheKey, payload)
	}
	return resp
}

func (svc *aiService) buildUpsellPrompt(req aiUpsellRequest) string {
	payload, _ := json.Marshal(req)
	return "Sei un assistente copywriter per upsell in-app durante un evento sportivo. Rispondi SOLO con JSON valido nel formato {\"suggestions\":[{\"product_name\":string,\"reason\":string,\"marketing_text\":string,\"priority\":number}]}. Regole: massimo 3 suggerimenti, non inventare prodotti, niente prezzi, tono breve e commerciale naturale, italiano. Input: " + string(payload)
}

func (svc *aiService) buildPopupPrompt(req aiPopupRequest) string {
	payload, _ := json.Marshal(req)
	return "Sei un assistente copywriter per popup in-app di fan engagement sportivo. Rispondi SOLO con JSON valido nel formato {\"popup_title\":string,\"popup_body\":string,\"cta_text\":string,\"tone\":string,\"urgency_level\":string}. Regole: copy breve, sportivo, chiaro, non invasivo, italiano. Input: " + string(payload)
}

func (svc *aiService) callProvider(ctx context.Context, prompt string) (string, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(svc.cfg.ProviderBaseURL), "/")
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	body := map[string]interface{}{
		"model": svc.cfg.Model,
		"messages": []map[string]string{
			{"role": "system", "content": "Return only valid JSON."},
			{"role": "user", "content": prompt},
		},
		"temperature": 0.4,
	}
	encoded, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/chat/completions", bytes.NewReader(encoded))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+svc.cfg.APIKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := svc.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("provider status %d", resp.StatusCode)
	}
	var parsed aiProviderResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", err
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("no provider choices")
	}
	return strings.TrimSpace(parsed.Choices[0].Message.Content), nil
}

func (svc *aiService) fallbackUpsells(req aiUpsellRequest) aiUpsellResponse {
	available := map[string]aiProductInput{}
	categories := map[string]bool{}
	for _, item := range req.Cart {
		if strings.TrimSpace(item.Category) != "" {
			categories[strings.ToLower(strings.TrimSpace(item.Category))] = true
		}
	}
	for _, p := range req.AvailableProducts {
		if p.Available && p.Visible {
			available[strings.ToLower(strings.TrimSpace(p.Name))] = p
		}
	}
	out := make([]aiUpsellSuggestion, 0, 3)
	for _, p := range req.AvailableProducts {
		if !p.Available || !p.Visible {
			continue
		}
		if p.Priority && !containsCart(req.Cart, p.Name) {
			out = append(out, aiUpsellSuggestion{ProductName: p.Name, ProductID: p.ProductID, Reason: "priorità commerciale attiva", MarketingText: "Aggiungilo al volo e completa l'ordine in un attimo.", Priority: 1})
		}
		if len(out) >= 3 {
			break
		}
	}
	for _, p := range req.AvailableProducts {
		if len(out) >= 3 {
			break
		}
		if !p.Available || !p.Visible || containsCart(req.Cart, p.Name) {
			continue
		}
		if categories[strings.ToLower(strings.TrimSpace(p.Category))] {
			out = append(out, aiUpsellSuggestion{ProductName: p.Name, ProductID: p.ProductID, Reason: "coerente con il carrello", MarketingText: "Ci sta perfettamente con quello che hai già scelto.", Priority: len(out) + 1})
		}
	}
	for _, p := range req.AvailableProducts {
		if len(out) >= 3 {
			break
		}
		if !p.Available || !p.Visible || containsCart(req.Cart, p.Name) || alreadySuggested(out, p.Name) {
			continue
		}
		out = append(out, aiUpsellSuggestion{ProductName: p.Name, ProductID: p.ProductID, Reason: "scelta rapida", MarketingText: "Un'aggiunta smart per chiudere l'ordine senza pensarci troppo.", Priority: len(out) + 1})
	}
	return aiUpsellResponse{Suggestions: out, Source: "fallback"}
}

func (svc *aiService) validateUpsells(req aiUpsellRequest, suggestions []aiUpsellSuggestion) []aiUpsellSuggestion {
	products := map[string]aiProductInput{}
	for _, p := range req.AvailableProducts {
		key := strings.ToLower(strings.TrimSpace(p.Name))
		if p.Available && p.Visible && key != "" {
			products[key] = p
		}
	}
	out := make([]aiUpsellSuggestion, 0, 3)
	for _, suggestion := range suggestions {
		if len(out) >= 3 {
			break
		}
		key := strings.ToLower(strings.TrimSpace(suggestion.ProductName))
		product, ok := products[key]
		if !ok || containsCart(req.Cart, suggestion.ProductName) || suggestion.Priority <= 0 || alreadySuggested(out, suggestion.ProductName) {
			continue
		}
		out = append(out, aiUpsellSuggestion{
			ProductName:   product.Name,
			ProductID:     product.ProductID,
			Reason:        sanitizeAIText(suggestion.Reason, 70),
			MarketingText: sanitizeAIText(suggestion.MarketingText, 90),
			Priority:      suggestion.Priority,
		})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Priority < out[j].Priority })
	return out
}

func (svc *aiService) fallbackPopup(req aiPopupRequest) aiPopupResponse {
	popup := aiPopupResponse{Tone: "sportivo", UrgencyLevel: "medium", Source: "fallback"}
	switch strings.TrimSpace(req.TriggerType) {
	case "coins_near_reward":
		popup.PopupTitle = "Sei vicino al premio"
		popup.PopupBody = "Ti manca poco: fai un'altra azione e prova a riscattare subito le tue monete."
		popup.CTA = "Guarda i premi"
	case "inactive_user":
		popup.PopupTitle = "La partita continua"
		popup.PopupBody = "Rientra nel live: c'è ancora tempo per giocare, votare o fare un ordine veloce."
		popup.CTA = "Torna ora"
	case "cart_abandon_risk":
		popup.PopupTitle = "Ordine quasi pronto"
		popup.PopupBody = "Hai già scelto il meglio: chiudi adesso l'ordine dal BAR in pochi tap."
		popup.CTA = "Completa ordine"
	case "game_reengagement":
		popup.PopupTitle = "Rientra in gioco"
		popup.PopupBody = "Hai già giocato bene in passato: torna ora e prova a guadagnare altre monete."
		popup.CTA = "Gioca adesso"
	default:
		popup.PopupTitle = "Scopri cosa puoi fare"
		popup.PopupBody = "Apri una feature live e sfrutta al massimo l'esperienza durante l'evento."
		popup.CTA = "Esplora"
	}
	return popup
}

func sanitizeAIText(value string, maxLen int) string {
	clean := strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
	clean = strings.ReplaceAll(clean, "\"", "")
	if maxLen > 0 && len(clean) > maxLen {
		clean = strings.TrimSpace(clean[:maxLen])
	}
	return clean
}

func containsCart(cart []aiCartItem, productName string) bool {
	target := strings.ToLower(strings.TrimSpace(productName))
	for _, item := range cart {
		if strings.ToLower(strings.TrimSpace(item.ProductName)) == target {
			return true
		}
	}
	return false
}

func alreadySuggested(items []aiUpsellSuggestion, productName string) bool {
	target := strings.ToLower(strings.TrimSpace(productName))
	for _, item := range items {
		if strings.ToLower(strings.TrimSpace(item.ProductName)) == target {
			return true
		}
	}
	return false
}

func (svc *aiService) cacheKey(prefix string, payload interface{}) string {
	encoded, _ := json.Marshal(payload)
	sum := sha1.Sum(encoded)
	return prefix + ":" + hex.EncodeToString(sum[:])
}

func (svc *aiService) getCache(key string) ([]byte, bool) {
	svc.mu.Lock()
	defer svc.mu.Unlock()
	entry, ok := svc.cache[key]
	if !ok || time.Now().After(entry.ExpiresAt) {
		delete(svc.cache, key)
		return nil, false
	}
	return entry.Payload, true
}

func (svc *aiService) setCache(key string, payload []byte) {
	svc.mu.Lock()
	defer svc.mu.Unlock()
	svc.cache[key] = cachedAIResult{ExpiresAt: time.Now().Add(svc.cfg.CacheTTL), Payload: payload}
}

func getAIIdentity(r *http.Request, ctxOrgID int, db database.AppDatabase) (int, string) {
	sessionToken := strings.TrimSpace(r.Header.Get("Authorization"))
	sessionToken = strings.TrimPrefix(sessionToken, "Bearer ")
	if sessionToken != "" && db != nil {
		if fan, err := db.GetFanBySessionToken(sessionToken); err == nil {
			return fan.Profile.ID, sessionToken
		}
	}
	deviceID := strings.TrimSpace(r.Header.Get("X-Device-ID"))
	if deviceID != "" && db != nil {
		if fan, err := db.GetFanByDevice(0, ctxOrgID, deviceID); err == nil {
			return fan.Profile.ID, deviceID
		}
		return 0, deviceID
	}
	return 0, ""
}

func getPopupSessionState(db database.AppDatabase, sessionID, trigger string, maxPopups int, cooldown time.Duration) (database.AIPopupSessionState, error) {
	if db == nil {
		return database.AIPopupSessionState{}, sql.ErrConnDone
	}
	return db.GetAIPopupSessionState(sessionID, trigger, maxPopups, cooldown)
}
