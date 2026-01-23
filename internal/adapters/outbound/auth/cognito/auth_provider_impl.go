package cognito

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
	"github.com/golang-jwt/jwt/v5"
	"github.com/reangeline/backend_applywise/internal/core/ports/outbound"
)

type authProviderImpl struct {
	client       *cognitoidentityprovider.Client
	userPoolID   string
	clientID     string
	clientSecret string
}

// NewAuthProvider cria nova instância do provider de autenticação
func NewAuthProvider(cfg aws.Config, userPoolID, clientID string) outbound.AuthProvider {
	return &authProviderImpl{
		client:     cognitoidentityprovider.NewFromConfig(cfg),
		userPoolID: userPoolID,
		clientID:   clientID,
	}
}

func (a *authProviderImpl) SignUp(ctx context.Context, email, password, name string) (string, error) {
	input := &cognitoidentityprovider.SignUpInput{
		ClientId: aws.String(a.clientID),
		Username: aws.String(email),
		Password: aws.String(password),
		UserAttributes: []types.AttributeType{
			{
				Name:  aws.String("email"),
				Value: aws.String(email),
			},
			{
				Name:  aws.String("name"),
				Value: aws.String(name),
			},
		},
	}

	result, err := a.client.SignUp(ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to sign up: %w", err)
	}

	// Auto-confirma o usuário (apenas para dev - em prod use email verification)
	if err := a.confirmUser(ctx, email); err != nil {
		return "", fmt.Errorf("failed to confirm user: %w", err)
	}

	return *result.UserSub, nil
}

func (a *authProviderImpl) SignIn(ctx context.Context, email, password string) (accessToken, refreshToken, idToken string, expiresIn int, err error) {
	input := &cognitoidentityprovider.InitiateAuthInput{
		AuthFlow: types.AuthFlowTypeUserPasswordAuth,
		ClientId: aws.String(a.clientID),
		AuthParameters: map[string]string{
			"USERNAME": email,
			"PASSWORD": password,
		},
	}

	result, err := a.client.InitiateAuth(ctx, input)
	if err != nil {
		return "", "", "", 0, fmt.Errorf("failed to sign in: %w", err)
	}

	if result.AuthenticationResult == nil {
		return "", "", "", 0, fmt.Errorf("authentication failed")
	}

	return *result.AuthenticationResult.AccessToken,
		*result.AuthenticationResult.RefreshToken,
		*result.AuthenticationResult.IdToken,
		int(result.AuthenticationResult.ExpiresIn),
		nil
}

func (a *authProviderImpl) SignOut(ctx context.Context, accessToken string) error {
	input := &cognitoidentityprovider.GlobalSignOutInput{
		AccessToken: aws.String(accessToken),
	}

	_, err := a.client.GlobalSignOut(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to sign out: %w", err)
	}

	return nil
}

func (a *authProviderImpl) RefreshToken(ctx context.Context, refreshToken string) (accessToken, idToken string, expiresIn int, err error) {
	input := &cognitoidentityprovider.InitiateAuthInput{
		AuthFlow: types.AuthFlowTypeRefreshTokenAuth,
		ClientId: aws.String(a.clientID),
		AuthParameters: map[string]string{
			"REFRESH_TOKEN": refreshToken,
		},
	}

	result, err := a.client.InitiateAuth(ctx, input)
	if err != nil {
		return "", "", 0, fmt.Errorf("failed to refresh token: %w", err)
	}

	if result.AuthenticationResult == nil {
		return "", "", 0, fmt.Errorf("refresh failed")
	}

	return *result.AuthenticationResult.AccessToken,
		*result.AuthenticationResult.IdToken,
		int(result.AuthenticationResult.ExpiresIn),
		nil
}

func (a *authProviderImpl) VerifyToken(ctx context.Context, token string) (string, error) {
	// Parse o JWT sem verificar assinatura (apenas para MVP/desenvolvimento)
	parser := jwt.NewParser()
	unverified, _, err := parser.ParseUnverified(token, jwt.MapClaims{})
	if err != nil {
		return "", fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := unverified.Claims.(jwt.MapClaims)
	if !ok {
		return "", fmt.Errorf("invalid token claims")
	}

	// Extrai o sub (Cognito User ID)
	sub, ok := claims["sub"].(string)
	if !ok {
		return "", fmt.Errorf("missing sub claim")
	}

	// Verifica se o token não expirou
	if exp, ok := claims["exp"].(float64); ok {
		if time.Now().Unix() > int64(exp) {
			return "", fmt.Errorf("token expired")
		}
	}

	return sub, nil
}

// confirmUser auto-confirma o usuário (apenas para desenvolvimento)
func (a *authProviderImpl) confirmUser(ctx context.Context, username string) error {
	input := &cognitoidentityprovider.AdminConfirmSignUpInput{
		UserPoolId: aws.String(a.userPoolID),
		Username:   aws.String(username),
	}

	_, err := a.client.AdminConfirmSignUp(ctx, input)
	return err
}
