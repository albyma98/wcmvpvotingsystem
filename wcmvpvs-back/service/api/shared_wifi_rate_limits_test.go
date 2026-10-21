package api

import (
	"fmt"
	"testing"
	"time"
)

func TestSharedWiFiVoteRateLimits(t *testing.T) {
	router := &_router{
		voteRateByDevice: map[string][]time.Time{},
		voteRateByIP:     map[string][]time.Time{},
	}
	now := time.Now()
	const sharedIP = "192.0.2.10"

	// Two thousand different devices can vote in the same instant on one network.
	for index := 0; index < 2000; index++ {
		if limited, message := router.shouldThrottleVoteAttempt(fmt.Sprintf("fan-%d", index), sharedIP, now); limited {
			t.Fatalf("shared Wi-Fi blocked voter %d: %s", index+1, message)
		}
	}
	if limited, message := router.shouldThrottleVoteAttempt("overflow", sharedIP, now); !limited || message != voteThrottleIPMessage {
		t.Fatalf("IP burst limit not enforced: limited=%v, message=%q", limited, message)
	}
	if limited, _ := router.shouldThrottleVoteAttempt("other-network", "192.0.2.11", now); limited {
		t.Fatal("one network's burst affected another network")
	}
	if limited, _ := router.shouldThrottleVoteAttempt("after-window", sharedIP, now.Add(10*time.Second+time.Nanosecond)); limited {
		t.Fatal("shared IP did not recover after the vote window")
	}

	// Switching networks must not bypass the existing per-device protection.
	for attempt := 0; attempt < 5; attempt++ {
		if limited, _ := router.shouldThrottleVoteAttempt("repeat-device", "192.0.2.12", now); limited {
			t.Fatalf("device blocked too early on attempt %d", attempt+1)
		}
	}
	if limited, message := router.shouldThrottleVoteAttempt("repeat-device", "192.0.2.13", now); !limited || message != voteThrottleMessage {
		t.Fatalf("device limit not enforced across networks: limited=%v, message=%q", limited, message)
	}
	if limited, _ := router.shouldThrottleVoteAttempt("repeat-device", "192.0.2.12", now.Add(30*time.Second+time.Nanosecond)); limited {
		t.Fatal("device did not recover after its vote window")
	}
}

func TestSharedWiFiSMSRateLimits(t *testing.T) {
	router := &_router{
		authRateByPhone: map[string][]time.Time{},
		authRateByIP:    map[string][]time.Time{},
	}
	const sharedIP = "192.0.2.20"

	// Simulate two thousand phones requesting an SMS and two resends each.
	for index := 0; index < 2000; index++ {
		phone := fmt.Sprintf("+39333%07d", index)
		for send := 0; send < 3; send++ {
			if !router.allowAuthRequest(phone, sharedIP) {
				t.Fatalf("shared Wi-Fi blocked phone %d on send %d", index+1, send+1)
			}
		}
	}
	if router.allowAuthRequest("+393339999990", sharedIP) {
		t.Fatal("SMS IP burst limit not enforced")
	}
	if !router.allowAuthRequest("+393339999991", "192.0.2.21") {
		t.Fatal("one network's SMS burst affected another network")
	}
	if router.allowAuthRequest("+393330000000", "192.0.2.22") {
		t.Fatal("switching networks bypassed the three-SMS phone limit")
	}

	// Seed an expired window without waiting or sending actual SMS messages.
	expired := time.Now().Add(-10*time.Minute - time.Second)
	for index := range router.authRateByIP[sharedIP] {
		router.authRateByIP[sharedIP][index] = expired
	}
	if !router.allowAuthRequest("+393339999992", sharedIP) {
		t.Fatal("shared IP did not recover after the SMS window")
	}
	phone := "+393330000000"
	for index := range router.authRateByPhone[phone] {
		router.authRateByPhone[phone][index] = expired
	}
	if !router.allowAuthRequest(phone, "192.0.2.22") {
		t.Fatal("phone did not recover after its SMS window")
	}
}
