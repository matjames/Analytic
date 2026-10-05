package main

import (
	"crypto/rand"
	"crypto/subtle"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"
)

// PendingOTP tracks an active verification challenge.
type PendingOTP struct {
	Code        string
	TargetKey   string
	ContactType string
	Contact     string
	ExpiresAt   time.Time
	Attempts    int
}

// OTPManager handles in-memory and durable OTP generation, validation, and lifecycle.
type OTPManager struct {
	mu    sync.RWMutex
	items map[string]*PendingOTP
}

var defaultOTPManager = newOTPManager()

func newOTPManager() *OTPManager {
	m := &OTPManager{
		items: make(map[string]*PendingOTP),
	}
	// Background cleanup of expired OTPs every 2 minutes
	go func() {
		ticker := time.NewTicker(2 * time.Minute)
		for range ticker.C {
			m.cleanupExpired()
		}
	}()
	return m
}

func (m *OTPManager) cleanupExpired() {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	for k, v := range m.items {
		if now.After(v.ExpiresAt) {
			delete(m.items, k)
		}
	}
}

// GenerateOTP produces a cryptographically secure 6-digit numeric code.
func (m *OTPManager) GenerateOTP(targetKey, contactType, contact string) (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(900000))
	if err != nil {
		return "", fmt.Errorf("failed to generate random OTP: %w", err)
	}
	code := fmt.Sprintf("%06d", n.Int64()+100000)

	otp := &PendingOTP{
		Code:        code,
		TargetKey:   targetKey,
		ContactType: contactType,
		Contact:     contact,
		ExpiresAt:   time.Now().Add(5 * time.Minute),
		Attempts:    0,
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.items[targetKey] = otp
	if contact != "" {
		m.items[contactType+":"+strings.ToLower(strings.TrimSpace(contact))] = otp
	}

	return code, nil
}

// VerifyOTP validates the supplied code against the stored OTP for any matching target key.
func (m *OTPManager) VerifyOTP(targetKeys []string, code string, isDevOrTest bool) (bool, string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var matchedKey string
	var otp *PendingOTP

	for _, k := range targetKeys {
		if k == "" {
			continue
		}
		if item, exists := m.items[k]; exists {
			matchedKey = k
			otp = item
			break
		}
	}

	if otp == nil {
		// In dev/test environments only, permit standard dev verification code "123456"
		if isDevOrTest && code == "123456" {
			return true, ""
		}
		return false, "no_pending_otp"
	}

	if time.Now().After(otp.ExpiresAt) {
		delete(m.items, matchedKey)
		return false, "otp_expired"
	}

	if otp.Attempts >= 5 {
		delete(m.items, matchedKey)
		return false, "max_attempts_exceeded"
	}

	// Constant-time comparison to prevent timing attacks
	if subtle.ConstantTimeCompare([]byte(otp.Code), []byte(code)) == 1 {
		// Clean up all related keys for this OTP
		for k, v := range m.items {
			if v == otp {
				delete(m.items, k)
			}
		}
		return true, ""
	}

	otp.Attempts++
	return false, "invalid_otp"
}
