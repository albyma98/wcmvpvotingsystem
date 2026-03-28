package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type twilioMessagingConfig struct {
	AccountSID                  string
	AuthToken                   string
	MessagingServiceSID         string
	WhatsAppMessagingServiceSID string
}

type twilioMessagingClient struct {
	httpClient *http.Client
	cfg        twilioMessagingConfig
}

type twilioMessageResult struct {
	SID    string `json:"sid"`
	Status string `json:"status"`
}

func newTwilioMessagingClient(cfg twilioMessagingConfig) *twilioMessagingClient {
	return &twilioMessagingClient{httpClient: &http.Client{Timeout: 10 * time.Second}, cfg: cfg}
}

func (c *twilioMessagingClient) enabled() bool {
	return strings.TrimSpace(c.cfg.AccountSID) != "" && strings.TrimSpace(c.cfg.AuthToken) != "" && strings.TrimSpace(c.cfg.MessagingServiceSID) != ""
}

func (c *twilioMessagingClient) enabledWhatsApp() bool {
	return strings.TrimSpace(c.cfg.AccountSID) != "" &&
		strings.TrimSpace(c.cfg.AuthToken) != "" &&
		strings.TrimSpace(c.cfg.WhatsAppMessagingServiceSID) != ""
}

func (c *twilioMessagingClient) SendSMS(phone string, body string) (twilioMessageResult, error) {
	if !c.enabled() {
		return twilioMessageResult{}, errors.New("twilio messaging not configured")
	}
	values, err := buildTwilioMessageValues(phone, body, c.cfg.MessagingServiceSID)
	if err != nil {
		return twilioMessageResult{}, err
	}
	resp, err := c.postForm(values)
	if err != nil {
		return twilioMessageResult{}, err
	}
	var parsed twilioMessageResult
	if err := json.Unmarshal(resp, &parsed); err != nil {
		return twilioMessageResult{}, err
	}
	return parsed, nil
}

func (c *twilioMessagingClient) SendWhatsApp(phone string, body string) (twilioMessageResult, error) {
	if !c.enabledWhatsApp() {
		return twilioMessageResult{}, errors.New("twilio whatsapp messaging not configured")
	}
	to := strings.TrimSpace(phone)
	if to == "" {
		return twilioMessageResult{}, errors.New("whatsapp destination is empty")
	}
	if !strings.HasPrefix(strings.ToLower(to), "whatsapp:") {
		to = "whatsapp:" + to
	}
	values, err := buildTwilioMessageValues(to, body, c.cfg.WhatsAppMessagingServiceSID)
	if err != nil {
		return twilioMessageResult{}, err
	}
	resp, err := c.postForm(values)
	if err != nil {
		return twilioMessageResult{}, err
	}
	var parsed twilioMessageResult
	if err := json.Unmarshal(resp, &parsed); err != nil {
		return twilioMessageResult{}, err
	}
	return parsed, nil
}

func buildTwilioMessageValues(to, body, messagingServiceSID string) (url.Values, error) {
	trimmedBody := strings.TrimSpace(body)
	if trimmedBody == "" {
		return nil, errors.New("sms body is empty")
	}
	values := url.Values{}
	values.Set("To", strings.TrimSpace(to))
	values.Set("Body", trimmedBody)
	values.Set("MessagingServiceSid", strings.TrimSpace(messagingServiceSID))
	return values, nil
}

func (c *twilioMessagingClient) postForm(values url.Values) ([]byte, error) {
	u := fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Messages.json", c.cfg.AccountSID)
	req, err := http.NewRequest(http.MethodPost, u, strings.NewReader(values.Encode()))
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(c.cfg.AccountSID, c.cfg.AuthToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body := []byte{}
	body, _ = io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, twilioError{StatusCode: resp.StatusCode, Message: string(body)}
	}
	return body, nil
}
