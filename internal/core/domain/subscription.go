package domain

import (
	"time"
)

type SubscriptionStatus string
type SubscriptionPlan string

const (
	SubscriptionStatusActive   SubscriptionStatus = "active"
	SubscriptionStatusCanceled SubscriptionStatus = "canceled"
	SubscriptionStatusExpired  SubscriptionStatus = "expired"
	SubscriptionStatusPastDue  SubscriptionStatus = "past_due"
)

const (
	PlanFree    SubscriptionPlan = "free"
	PlanBasic   SubscriptionPlan = "basic"
	PlanPremium SubscriptionPlan = "premium"
)

type Subscription struct {
	ID               string             `json:"id"`
	UserID           string             `json:"user_id"`
	Plan             SubscriptionPlan   `json:"plan"`
	Status           SubscriptionStatus `json:"status"`
	StripeCustomerID string             `json:"stripe_customer_id,omitempty"`
	StripeSubID      string             `json:"stripe_subscription_id,omitempty"`
	CurrentPeriodEnd time.Time          `json:"current_period_end"`
	CreatedAt        time.Time          `json:"created_at"`
	UpdatedAt        time.Time          `json:"updated_at"`
	CanceledAt       *time.Time         `json:"canceled_at,omitempty"`
}

func NewSubscription(userID string, plan SubscriptionPlan) *Subscription {
	now := time.Now()

	// Free tier por padrão
	periodEnd := now.AddDate(100, 0, 0) // "infinito" para free
	if plan != PlanFree {
		periodEnd = now.AddDate(0, 1, 0) // 1 mês
	}

	return &Subscription{
		UserID:           userID,
		Plan:             plan,
		Status:           SubscriptionStatusActive,
		CurrentPeriodEnd: periodEnd,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}

func (s *Subscription) IsActive() bool {
	return s.Status == SubscriptionStatusActive &&
		time.Now().Before(s.CurrentPeriodEnd)
}

func (s *Subscription) Cancel() error {
	if s.Status == SubscriptionStatusCanceled {
		return ErrSubscriptionAlreadyCanceled
	}

	now := time.Now()
	s.Status = SubscriptionStatusCanceled
	s.CanceledAt = &now
	s.UpdatedAt = now
	return nil
}

func (s *Subscription) Renew(periodEnd time.Time) {
	s.CurrentPeriodEnd = periodEnd
	s.Status = SubscriptionStatusActive
	s.UpdatedAt = time.Now()
}
