package domain

import (
	"time"
)

type SubscriptionStatus string
type SubscriptionPlan string
type Store string

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

const (
	StoreAppStore  Store = "app_store"
	StorePlayStore Store = "play_store"
	StoreStripe    Store = "stripe"
	StoreNone      Store = ""
)

type Subscription struct {
	ID     string             `json:"id"`
	UserID string             `json:"user_id"`
	Plan   SubscriptionPlan   `json:"plan"`
	Status SubscriptionStatus `json:"status"`

	// Stripe (web payments)
	StripeCustomerID string `json:"stripe_customer_id,omitempty"`
	StripeSubID      string `json:"stripe_subscription_id,omitempty"`

	// RevenueCat (mobile payments)
	RevenueCatCustomerID            string `json:"revenuecat_customer_id,omitempty"`
	RevenueCatOriginalTransactionID string `json:"revenuecat_transaction_id,omitempty"`
	Store                           Store  `json:"store,omitempty"`
	ProductIdentifier               string `json:"product_identifier,omitempty"`

	CurrentPeriodEnd time.Time  `json:"current_period_end"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	CanceledAt       *time.Time `json:"canceled_at,omitempty"`
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
		Store:            StoreNone,
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

func (s *Subscription) Expire() {
	s.Status = SubscriptionStatusExpired
	s.UpdatedAt = time.Now()
}

// UpdateFromRevenueCat atualiza subscription com dados do RevenueCat
func (s *Subscription) UpdateFromRevenueCat(
	customerID string,
	transactionID string,
	productID string,
	store Store,
	expiresAt time.Time,
	isActive bool,
) {
	s.RevenueCatCustomerID = customerID
	s.RevenueCatOriginalTransactionID = transactionID
	s.ProductIdentifier = productID
	s.Store = store
	s.CurrentPeriodEnd = expiresAt
	s.UpdatedAt = time.Now()

	if isActive {
		s.Status = SubscriptionStatusActive
		s.Plan = PlanPremium
	} else {
		s.Status = SubscriptionStatusExpired
		s.Plan = PlanFree
	}
}
