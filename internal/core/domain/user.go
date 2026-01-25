package domain

import (
	"time"
)

type UserStatus string

const (
	UserStatusActive    UserStatus = "active"
	UserStatusInactive  UserStatus = "inactive"
	UserStatusSuspended UserStatus = "suspended"
)

type User struct {
	ID               string     `json:"id"`
	Email            string     `json:"email"`
	Name             string     `json:"name"`
	Status           UserStatus `json:"status"`
	CognitoID        string     `json:"cognito_id"`
	StripeCustomerID string     `json:"stripe_customer_id,omitempty"`
	EmailVerified    bool       `json:"email_verified"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

func NewUser(email, name, cognitoID string) *User {
	now := time.Now()
	return &User{
		Email:         email,
		Name:          name,
		Status:        UserStatusActive,
		CognitoID:     cognitoID,
		EmailVerified: false,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

func (u *User) Activate() {
	u.Status = UserStatusActive
	u.UpdatedAt = time.Now()
}

func (u *User) Suspend() {
	u.Status = UserStatusSuspended
	u.UpdatedAt = time.Now()
}
