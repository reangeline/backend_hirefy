package domain

import (
	"crypto/rand"
	"fmt"
	"time"
)

type VerificationCode struct {
	Email     string
	Code      string
	ExpiresAt time.Time
	CreatedAt time.Time
}

// GenerateVerificationCode gera código de 6 dígitos
func GenerateVerificationCode() string {
	// Gerar 6 dígitos aleatórios
	b := make([]byte, 3)
	_, _ = rand.Read(b)
	code := fmt.Sprintf("%06d", int(b[0])<<16|int(b[1])<<8|int(b[2]))
	return code[:6]
}

func NewVerificationCode(email string) *VerificationCode {
	return &VerificationCode{
		Email:     email,
		Code:      GenerateVerificationCode(),
		ExpiresAt: time.Now().Add(15 * time.Minute), // Expira em 15 min
		CreatedAt: time.Now(),
	}
}

func (v *VerificationCode) IsExpired() bool {
	return time.Now().After(v.ExpiresAt)
}
