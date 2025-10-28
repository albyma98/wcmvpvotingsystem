package api

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
)

const (
	ticketCodeDigits          = 4
	maxCodeGenerationAttempts = 100
)

func generateNumericCode() (string, error) {
	min := big.NewInt(1)
	if ticketCodeDigits > 1 {
		// 10^(digits-1)
		min.Exp(big.NewInt(10), big.NewInt(ticketCodeDigits-1), nil)
	}
	rangeMax := big.NewInt(9)
	if ticketCodeDigits == 1 {
		rangeMax = big.NewInt(10)
	}
	if ticketCodeDigits > 1 {
		rangeMax.Mul(rangeMax, min) // 9 * 10^(digits-1)
	}
	n, err := rand.Int(rand.Reader, rangeMax)
	if err != nil {
		return "", err
	}
	if ticketCodeDigits > 1 {
		n.Add(n, min)
	}
	return fmt.Sprintf("%0*d", ticketCodeDigits, n.Int64()), nil
}

func signCode(secret, code string) string {
	h := hmac.New(sha256.New, []byte(secret))
	_, _ = h.Write([]byte(code))
	return hex.EncodeToString(h.Sum(nil))
}

func isUniqueConstraintError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "UNIQUE constraint failed")
}

func isVoteCodeCollision(err error) bool {
	return err != nil && strings.Contains(err.Error(), "votes.event_id, votes.ticket_code")
}

func isVoteFingerprintCollision(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "votes.event_id, votes.fingerprint_hash") ||
		strings.Contains(msg, "votes.fingerprint_hash") ||
		strings.Contains(strings.ToLower(msg), "unique_vote_per_event_fingerprint")
}
