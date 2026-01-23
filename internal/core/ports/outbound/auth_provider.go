package outbound

import "context"

// AuthProvider define integração com provedor de autenticação (Cognito)
type AuthProvider interface {
	SignUp(ctx context.Context, email, password, name string) (string, error) // retorna cognitoID
	SignIn(ctx context.Context, email, password string) (accessToken, refreshToken, idToken string, expiresIn int, err error)
	SignOut(ctx context.Context, accessToken string) error
	RefreshToken(ctx context.Context, refreshToken string) (accessToken, idToken string, expiresIn int, err error)
	VerifyToken(ctx context.Context, token string) (string, error) // retorna cognitoID
}
