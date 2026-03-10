package domain

import (
	"time"

	"github.com/google/uuid"
)

type CreditTransactionType string

const (
	CreditTransactionTypeAdd CreditTransactionType = "add"
	CreditTransactionTypeUse CreditTransactionType = "use"
)

type CreditTransaction struct {
	ID        string                `json:"id"`
	UserID    string                `json:"user_id"`
	Amount    int                   `json:"amount"`
	Type      CreditTransactionType `json:"type"`
	Reason    string                `json:"reason"`
	Metadata  map[string]string     `json:"metadata,omitempty"` // ex: {"resume_id": "...", "product_id": "credits_10"}
	CreatedAt time.Time             `json:"created_at"`
}

func NewCreditTransaction(userID string, amount int, txType CreditTransactionType, reason string) *CreditTransaction {
	return &CreditTransaction{
		ID:        uuid.New().String(),
		UserID:    userID,
		Amount:    amount,
		Type:      txType,
		Reason:    reason,
		Metadata:  make(map[string]string),
		CreatedAt: time.Now(),
	}
}
