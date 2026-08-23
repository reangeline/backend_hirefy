package cognito

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
	"github.com/golang-jwt/jwt/v5"
	"github.com/reangeline/backend_applywise/internal/core/ports/outbound"
)

type authProviderImpl struct {
	client     *cognitoidentityprovider.Client
	userPoolID string
	clientID   string
}

// NewAuthProvider cria nova instância do provider de autenticação
func NewAuthProvider(cfg aws.Config, userPoolID, clientID string) outbound.AuthProvider {
	return &authProviderImpl{
		client:     cognitoidentityprovider.NewFromConfig(cfg),
		userPoolID: userPoolID,
		clientID:   clientID,
	}
}

// SignUp cria um novo usuário no Cognito
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

	// Auto-confirma o usuário
	if err := a.confirmUser(ctx, email); err != nil {
		return "", fmt.Errorf("failed to confirm user: %w", err)
	}

	// Marca email como verificado no Cognito (necessário para ForgotPassword funcionar)
	if err := a.MarkEmailAsVerified(ctx, email); err != nil {
		fmt.Printf("⚠️ Warning: failed to mark email as verified during signup: %v\n", err)
	}

	return *result.UserSub, nil
}

// SignIn autentica um usuário
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

// SignOut faz logout global do usuário
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

// RefreshToken renova o access token
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

// VerifyToken verifica e valida um JWT token
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

// ConfirmSignUp confirma o email do usuário com código
func (a *authProviderImpl) ConfirmSignUp(ctx context.Context, email, confirmationCode string) error {
	input := &cognitoidentityprovider.ConfirmSignUpInput{
		ClientId:         aws.String(a.clientID),
		Username:         aws.String(email),
		ConfirmationCode: aws.String(confirmationCode),
	}

	_, err := a.client.ConfirmSignUp(ctx, input)
	if err != nil {
		return a.handleCognitoError(err)
	}

	return nil
}

// ResendConfirmationCode reenvia o código de confirmação
func (a *authProviderImpl) ResendConfirmationCode(ctx context.Context, email string) error {
	input := &cognitoidentityprovider.ResendConfirmationCodeInput{
		ClientId: aws.String(a.clientID),
		Username: aws.String(email),
	}

	_, err := a.client.ResendConfirmationCode(ctx, input)
	if err != nil {
		return a.handleCognitoError(err)
	}

	return nil
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

func (a *authProviderImpl) handleCognitoError(err error) error {
	// Mapeia erros específicos do Cognito para mensagens amigáveis
	errMsg := err.Error()

	switch e := err.(type) {
	case *types.CodeMismatchException:
		return fmt.Errorf("CodeMismatchException: Invalid verification code provided")
	case *types.ExpiredCodeException:
		return fmt.Errorf("ExpiredCodeException: Verification code has expired, please request a new one")
	case *types.UserNotFoundException:
		return fmt.Errorf("UserNotFoundException: User does not exist")
	case *types.NotAuthorizedException:
		return fmt.Errorf("NotAuthorizedException: User cannot be confirmed. User may already be confirmed")
	case *types.TooManyRequestsException:
		return fmt.Errorf("TooManyRequestsException: Too many requests, please try again later")
	case *types.InvalidParameterException:
		// Se for erro de reenvio para usuário já confirmado
		if strings.Contains(errMsg, "already confirmed") || strings.Contains(errMsg, "cannot reset") {
			return fmt.Errorf("NotAuthorizedException: User is already confirmed")
		}
		return fmt.Errorf("InvalidParameterException: Invalid parameter provided")
	default:
		// Log do erro original para debug
		fmt.Printf("Cognito error: %v\n", err)
		return fmt.Errorf("CognitoError: %v", e)
	}
}

// ForgotPassword inicia o fluxo de reset de senha via Cognito
func (a *authProviderImpl) ForgotPassword(ctx context.Context, email string) error {
	input := &cognitoidentityprovider.ForgotPasswordInput{
		ClientId: aws.String(a.clientID),
		Username: aws.String(email),
	}

	_, err := a.client.ForgotPassword(ctx, input)
	if err != nil {
		return a.handleCognitoError(err)
	}

	return nil
}

// ConfirmForgotPassword confirma o novo password com o código recebido
func (a *authProviderImpl) ConfirmForgotPassword(ctx context.Context, email, code, newPassword string) error {
	input := &cognitoidentityprovider.ConfirmForgotPasswordInput{
		ClientId:         aws.String(a.clientID),
		Username:         aws.String(email),
		ConfirmationCode: aws.String(code),
		Password:         aws.String(newPassword),
	}

	_, err := a.client.ConfirmForgotPassword(ctx, input)
	if err != nil {
		return a.handleCognitoError(err)
	}

	return nil
}

func (a *authProviderImpl) MarkEmailAsVerified(ctx context.Context, email string) error {
	input := &cognitoidentityprovider.AdminUpdateUserAttributesInput{
		UserPoolId: aws.String(a.userPoolID),
		Username:   aws.String(email),
		UserAttributes: []types.AttributeType{
			{
				Name:  aws.String("email_verified"),
				Value: aws.String("true"),
			},
		},
	}

	_, err := a.client.AdminUpdateUserAttributes(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to mark email as verified in Cognito: %w", err)
	}

	fmt.Printf("✅ Cognito: email_verified set to true for %s\n", email)
	return nil
}

// UserExistsInCognito returns true when the user exists in the Cognito user pool.
// It uses AdminGetUser internally; a UserNotFoundException means the user is absent.
func (a *authProviderImpl) UserExistsInCognito(ctx context.Context, email string) (bool, error) {
	_, err := a.getCognitoUserID(ctx, email)
	if err != nil {
		var notFound *types.UserNotFoundException
		if errors.As(err, &notFound) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check Cognito user existence: %w", err)
	}
	return true, nil
}

// DeleteUser permanently removes a user from the Cognito user pool.
// It is idempotent: if the user no longer exists the call is treated as a success.
func (a *authProviderImpl) DeleteUser(ctx context.Context, email string) error {
	input := &cognitoidentityprovider.AdminDeleteUserInput{
		UserPoolId: aws.String(a.userPoolID),
		Username:   aws.String(email),
	}

	_, err := a.client.AdminDeleteUser(ctx, input)
	if err != nil {
		var notFound *types.UserNotFoundException
		if errors.As(err, &notFound) {
			// User is already gone – goal achieved.
			return nil
		}
		return fmt.Errorf("failed to delete user from Cognito: %w", err)
	}

	return nil
}
