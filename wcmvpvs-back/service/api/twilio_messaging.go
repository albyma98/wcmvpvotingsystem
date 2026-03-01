package api

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type twilioMessagingConfig struct {
	AccountSID          string
	AuthToken           string
	MessagingServiceSID string
}

type twilioMessagingClient struct {
	httpClient *http.Client
	cfg        twilioMessagingConfig
}

func newTwilioMessagingClient(cfg twilioMessagingConfig) *twilioMessagingClient {
	return &twilioMessagingClient{httpClient: &http.Client{Timeout: 10 * time.Second}, cfg: cfg}
}

func (c *twilioMessagingClient) enabled() bool {
	return strings.TrimSpace(c.cfg.AccountSID) != "" && strings.TrimSpace(c.cfg.AuthToken) != "" && strings.TrimSpace(c.cfg.MessagingServiceSID) != ""
}

func (c *twilioMessagingClient) SendSMS(phone string, body string) error {
	if !c.enabled() {
		return errors.New("twilio messaging not configured")
	}
	trimmedBody := strings.TrimSpace(body)
	if trimmedBody == "" {
		return errors.New("sms body is empty")
	}
	values := url.Values{}
	values.Set("To", phone)
	values.Set("Body", trimmedBody)
	values.Set("MessagingServiceSid", c.cfg.MessagingServiceSID)
	_, err := c.postForm(values)
	return err
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
