package inbound

import "context"

type SignUpRequest struct {
	Email    string
	Password string
	Name     string
}

type SignInRequest struct {
	Email    string
	Password string
}

type AuthResponse struct {
	AccessToken  string
	RefreshToken string
	IDToken      string
	ExpiresIn    int
}

type AuthService interface {
	SignUp(ctx context.Context, req SignUpRequest) (*AuthResponse, error)
	SignIn(ctx context.Context, req SignInRequest) (*AuthResponse, error)
	SignOut(ctx context.Context, accessToken string) error
	RefreshToken(ctx context.Context, refreshToken string) (*AuthResponse, error)
	VerifyToken(ctx context.Context, token string) (string, error) // retorna cognitoID
}
