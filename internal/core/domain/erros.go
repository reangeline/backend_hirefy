package domain

import "errors"

var (
	// User errors
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user already exists")
	ErrInvalidEmail      = errors.New("invalid email")
	ErrEmailNotVerified  = errors.New("email not verified")

	// Subscription errors
	ErrSubscriptionNotFound        = errors.New("subscription not found")
	ErrSubscriptionAlreadyCanceled = errors.New("subscription already canceled")
	ErrSubscriptionInactive        = errors.New("subscription is not active")
	ErrInvalidPlan                 = errors.New("invalid subscription plan")
	ErrInsufficientCredits         = errors.New("insufficient credits")

	// Resume errors
	ErrResumeNotFound = errors.New("resume not found")
	ErrInvalidResume  = errors.New("invalid resume format")
	ErrResumeTooLarge = errors.New("resume file too large")

	// Job errors
	ErrJobNotFound = errors.New("optimization job not found")

	// Payment errors
	ErrPaymentFailed        = errors.New("payment failed")
	ErrInvalidPaymentMethod = errors.New("invalid payment method")

	// Authorization errors
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
)
