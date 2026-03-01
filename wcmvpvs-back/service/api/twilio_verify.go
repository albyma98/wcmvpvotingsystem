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

type twilioVerifyConfig struct {
	AccountSID string
	AuthToken  string
	ServiceSID string
}

type twilioVerifyClient struct {
	httpClient *http.Client
	cfg        twilioVerifyConfig
}

type twilioError struct {
	StatusCode int
	Code       int    `json:"code"`
	Message    string `json:"message"`
}

func (e twilioError) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("twilio status %d", e.StatusCode)
	}
	return fmt.Sprintf("twilio error %d: %s", e.StatusCode, e.Message)
}

func newTwilioVerifyClient(cfg twilioVerifyConfig) *twilioVerifyClient {
	return &twilioVerifyClient{httpClient: &http.Client{Timeout: 10 * time.Second}, cfg: cfg}
}

func (c *twilioVerifyClient) enabled() bool {
	return strings.TrimSpace(c.cfg.AccountSID) != "" && strings.TrimSpace(c.cfg.AuthToken) != "" && strings.TrimSpace(c.cfg.ServiceSID) != ""
}

func (c *twilioVerifyClient) StartSMSVerification(phone string) error {
	if !c.enabled() {
		return errors.New("twilio not configured")
	}
	values := url.Values{}
	values.Set("To", phone)
	values.Set("Channel", "sms")
	_, err := c.postForm("Verifications", values)
	return err
}

func (c *twilioVerifyClient) CheckSMSVerification(phone string, code string) (bool, error) {
	if !c.enabled() {
		return false, errors.New("twilio not configured")
	}
	values := url.Values{}
	values.Set("To", phone)
	values.Set("Code", code)
	respBody, err := c.postForm("VerificationCheck", values)
	if err != nil {
		return false, err
	}
	var payload struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(respBody, &payload); err != nil {
		return false, err
	}
	return strings.EqualFold(payload.Status, "approved"), nil
}

func (c *twilioVerifyClient) postForm(path string, values url.Values) ([]byte, error) {
	u := fmt.Sprintf("https://verify.twilio.com/v2/Services/%s/%s", c.cfg.ServiceSID, path)
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
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var twErr twilioError
		twErr.StatusCode = resp.StatusCode
		if err := json.Unmarshal(body, &twErr); err != nil {
			twErr.Message = string(body)
		}
		return nil, twErr
	}
	return body, nil
}

func twilioHTTPStatus(err error) int {
	var twErr twilioError
	if errors.As(err, &twErr) {
		switch twErr.StatusCode {
		case http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden:
			return http.StatusBadGateway
		case http.StatusTooManyRequests:
			return http.StatusTooManyRequests
		default:
			return http.StatusBadGateway
		}
	}
	return http.StatusBadGateway
}
