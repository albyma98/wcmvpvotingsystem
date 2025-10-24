package api

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
)

func (rt *_router) antiDoubleVoteMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hashedIP, err := rt.computeHashedIP(r)
		if err != nil {
			rt.baseLogger.WithError(err).Warn("unable to compute client hashed ip")
			writeJSONError(w, http.StatusBadRequest, "IP non valido")
			return
		}

		ctx := withHashedIP(r.Context(), hashedIP)

		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			rt.baseLogger.WithField("hashed_ip", hashedIP).WithError(err).Error("unable to read vote payload")
			writeJSONError(w, http.StatusInternalServerError, "Errore interno")
			return
		}
		_ = r.Body.Close()

		var payload struct {
			EventID int    `json:"event_id"`
			Code    string `json:"code"`
		}
		if len(bodyBytes) > 0 {
			if err := json.Unmarshal(bodyBytes, &payload); err != nil {
				// Restore body and continue, handler will return an error
				r = r.WithContext(ctx)
				r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
				next.ServeHTTP(w, r)
				return
			}
		}

		ctx = withHashedIP(ctx, hashedIP)

		trimmedCode := strings.TrimSpace(payload.Code)
		if trimmedCode != "" {
			bypassCode, err := rt.validateBypassCode(trimmedCode)
			if err != nil {
				rt.baseLogger.WithField("hashed_ip", hashedIP).WithError(err).Warn("invalid bypass code provided")
				writeJSONError(w, http.StatusBadRequest, "Codice non valido")
				return
			}

			used, err := rt.db.IsBypassCodeUsed(bypassCode)
			if err != nil {
				rt.baseLogger.WithField("hashed_ip", hashedIP).WithError(err).Error("unable to verify bypass code usage")
				writeJSONError(w, http.StatusInternalServerError, "Errore interno")
				return
			}
			if used {
				writeJSONError(w, http.StatusConflict, "Codice già utilizzato")
				return
			}

			ctx = withBypassCode(ctx, bypassCode)
		} else if payload.EventID != 0 {
			already, err := rt.db.HasVoteFromHashedIP(payload.EventID, hashedIP)
			if err != nil {
				rt.baseLogger.WithField("hashed_ip", hashedIP).WithError(err).Error("unable to verify existing vote")
				writeJSONError(w, http.StatusInternalServerError, "Errore interno")
				return
			}
			if already {
				writeJSONError(w, http.StatusConflict, "Hai già votato")
				return
			}
		}

		r = r.WithContext(ctx)
		r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		next.ServeHTTP(w, r)
	})
}

func (rt *_router) computeHashedIP(r *http.Request) (string, error) {
	rawIP := strings.TrimSpace(r.Header.Get("X-Forwarded-For"))
	if rawIP != "" {
		parts := strings.Split(rawIP, ",")
		if len(parts) > 0 {
			rawIP = strings.TrimSpace(parts[0])
		}
	}
	if rawIP == "" {
		host := r.RemoteAddr
		if h, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
			host = h
		}
		rawIP = strings.TrimSpace(host)
	}
	if rawIP == "" {
		return "", errors.New("empty ip address")
	}

	parsedIP := net.ParseIP(rawIP)
	if parsedIP == nil {
		return "", fmt.Errorf("invalid ip address: %s", rawIP)
	}

	truncatedIP, prefix, err := truncateIP(parsedIP)
	if err != nil {
		return "", err
	}

	canonical := fmt.Sprintf("%s/%d", truncatedIP, prefix)
	mac := hmac.New(sha256.New, rt.hmacIPKey)
	_, _ = mac.Write([]byte(canonical))
	return hex.EncodeToString(mac.Sum(nil)), nil
}

func truncateIP(ip net.IP) (net.IP, int, error) {
	if v4 := ip.To4(); v4 != nil {
		mask := net.CIDRMask(24, 32)
		return v4.Mask(mask), 24, nil
	}
	v6 := ip.To16()
	if v6 == nil {
		return nil, 0, errors.New("invalid ip length")
	}
	mask := net.CIDRMask(64, 128)
	return v6.Mask(mask), 64, nil
}

func (rt *_router) validateBypassCode(raw string) (string, error) {
	parts := strings.Split(raw, ".")
	if len(parts) != 2 {
		return "", errors.New("invalid bypass code format")
	}
	code := strings.TrimSpace(parts[0])
	signatureHex := strings.TrimSpace(parts[1])
	if code == "" || signatureHex == "" {
		return "", errors.New("invalid bypass code value")
	}

	providedSignature, err := hex.DecodeString(signatureHex)
	if err != nil {
		return "", fmt.Errorf("invalid bypass code signature: %w", err)
	}

	mac := hmac.New(sha256.New, rt.hmacCodeKey)
	_, _ = mac.Write([]byte(code))
	expectedSignature := mac.Sum(nil)

	if !hmac.Equal(providedSignature, expectedSignature) {
		return "", errors.New("bypass code signature mismatch")
	}

	return code, nil
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
