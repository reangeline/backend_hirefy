package inbound

import "context"

type SignUpRequest struct {
	Email           string
	Password        string
	Name            string
	TermsAcceptedAt string
	TermsVersion    string
}

type SignInRequest struct {
	Email    string
	Password string
}

type RefreshTokenRequest struct {
	RefreshToken string
}

type ConfirmSignUpRequest struct {
	Email string
	Code  string
}

type ResendCodeRequest struct {
	Email string
}

type ForgotPasswordRequest struct {
	Email string
}

type ConfirmForgotPasswordRequest struct {
	Email       string
	Code        string
	NewPassword string
}

// ✅ ADICIONAR AuthResponse
type AuthResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token"`
	ExpiresIn    int    `json:"expires_in"`
	Message      string `json:"message,omitempty"` // ✅ CERTIFIQUE-SE QUE ESTÁ AQUI

}

type AuthService interface {
	SignUp(ctx context.Context, req SignUpRequest) (*AuthResponse, error)
	SignIn(ctx context.Context, req SignInRequest) (*AuthResponse, error)
	SignOut(ctx context.Context, accessToken string) error
	RefreshToken(ctx context.Context, refreshToken string) (*AuthResponse, error)
	VerifyToken(ctx context.Context, token string) (string, error) // ✅ ADICIONAR
	ConfirmSignUp(ctx context.Context, req ConfirmSignUpRequest) error
	ResendConfirmationCode(ctx context.Context, req ResendCodeRequest) error
	ForgotPassword(ctx context.Context, req ForgotPasswordRequest) error
	ConfirmForgotPassword(ctx context.Context, req ConfirmForgotPasswordRequest) error
}
