package outbound

import (
	"context"
)

type AuthProvider interface {
	SignUp(ctx context.Context, email, password, name string) (string, error)
	SignIn(ctx context.Context, email, password string) (accessToken, refreshToken, idToken string, expiresIn int, err error)
	SignOut(ctx context.Context, accessToken string) error
	RefreshToken(ctx context.Context, refreshToken string) (accessToken, idToken string, expiresIn int, err error)
	VerifyToken(ctx context.Context, token string) (string, error)
	ConfirmSignUp(ctx context.Context, email, confirmationCode string) error
	ResendConfirmationCode(ctx context.Context, email string) error
	MarkEmailAsVerified(ctx context.Context, email string) error
	ForgotPassword(ctx context.Context, email string) error
	ConfirmForgotPassword(ctx context.Context, email, code, newPassword string) error
	DeleteUser(ctx context.Context, email string) error
	// UserExistsInCognito returns true when a Cognito user with the given email
	// exists in the user pool. Used to detect orphaned DynamoDB records after a
	// partial account deletion.
	UserExistsInCognito(ctx context.Context, email string) (bool, error)
	// SocialSignIn creates the Cognito user if needed, sets a random permanent
	// password, and authenticates via ADMIN_USER_PASSWORD_AUTH. Returns the
	// Cognito sub (cognitoID) plus the standard token triple.
	SocialSignIn(ctx context.Context, email, name string) (cognitoID, accessToken, refreshToken, idToken string, expiresIn int, err error)
}
