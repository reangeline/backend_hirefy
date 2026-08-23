package cognito

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
)

// SocialSignIn is idempotent: it creates the Cognito user if it doesn't exist,
// then sets a fresh random permanent password, authenticates via
// ADMIN_USER_PASSWORD_AUTH, and returns the token triple plus the user's sub.
func (a *authProviderImpl) SocialSignIn(ctx context.Context, email, name string) (cognitoID, accessToken, refreshToken, idToken string, expiresIn int, err error) {
	// 1. Check whether the user already exists in the pool.
	cognitoID, err = a.getCognitoUserID(ctx, email)
	if err != nil {
		var notFound *types.UserNotFoundException
		if !errors.As(err, &notFound) {
			return "", "", "", "", 0, fmt.Errorf("failed to check Cognito user: %w", err)
		}
		// User does not exist – create it.
		cognitoID, err = a.adminCreateSocialUser(ctx, email, name)
		if err != nil {
			// Race condition: another request created the user between the check
			// and the create call. Retrieve the ID and continue.
			var usernameExists *types.UsernameExistsException
			if errors.As(err, &usernameExists) {
				cognitoID, err = a.getCognitoUserID(ctx, email)
				if err != nil {
					return "", "", "", "", 0, fmt.Errorf("failed to retrieve user after conflict: %w", err)
				}
			} else {
				return "", "", "", "", 0, fmt.Errorf("failed to create social user: %w", err)
			}
		}
	}

	// 2. Set a fresh random permanent password so we can authenticate in step 3.
	// Generating a new password on every login is intentional – the password is
	// never exposed to the user and exists solely to satisfy Cognito's auth flow.
	password, err := generateSocialPassword()
	if err != nil {
		return "", "", "", "", 0, fmt.Errorf("failed to generate password: %w", err)
	}

	setPassInput := &cognitoidentityprovider.AdminSetUserPasswordInput{
		UserPoolId: aws.String(a.userPoolID),
		Username:   aws.String(email),
		Password:   aws.String(password),
		Permanent:  true,
	}
	if _, err = a.client.AdminSetUserPassword(ctx, setPassInput); err != nil {
		return "", "", "", "", 0, fmt.Errorf("failed to set social user password: %w", err)
	}

	// 3. Authenticate using the admin flow.
	authInput := &cognitoidentityprovider.AdminInitiateAuthInput{
		UserPoolId: aws.String(a.userPoolID),
		ClientId:   aws.String(a.clientID),
		AuthFlow:   types.AuthFlowTypeAdminUserPasswordAuth,
		AuthParameters: map[string]string{
			"USERNAME": email,
			"PASSWORD": password,
		},
	}
	authResult, err := a.client.AdminInitiateAuth(ctx, authInput)
	if err != nil {
		return "", "", "", "", 0, fmt.Errorf("failed to authenticate social user: %w", err)
	}
	if authResult.AuthenticationResult == nil {
		return "", "", "", "", 0, fmt.Errorf("social authentication returned no result")
	}

	return cognitoID,
		aws.ToString(authResult.AuthenticationResult.AccessToken),
		aws.ToString(authResult.AuthenticationResult.RefreshToken),
		aws.ToString(authResult.AuthenticationResult.IdToken),
		int(authResult.AuthenticationResult.ExpiresIn),
		nil
}

// getCognitoUserID retrieves the Cognito sub attribute for an existing user.
// Returns UserNotFoundException when the user does not exist.
func (a *authProviderImpl) getCognitoUserID(ctx context.Context, email string) (string, error) {
	input := &cognitoidentityprovider.AdminGetUserInput{
		UserPoolId: aws.String(a.userPoolID),
		Username:   aws.String(email),
	}
	result, err := a.client.AdminGetUser(ctx, input)
	if err != nil {
		return "", err
	}
	for _, attr := range result.UserAttributes {
		if aws.ToString(attr.Name) == "sub" {
			return aws.ToString(attr.Value), nil
		}
	}
	return "", fmt.Errorf("sub attribute not found for user %s", email)
}

// adminCreateSocialUser creates a pre-confirmed Cognito user with email_verified=true
// and MessageAction=SUPPRESS so no welcome email is sent.
func (a *authProviderImpl) adminCreateSocialUser(ctx context.Context, email, name string) (string, error) {
	if name == "" {
		name = email // fallback so the attribute is never empty
	}
	input := &cognitoidentityprovider.AdminCreateUserInput{
		UserPoolId:    aws.String(a.userPoolID),
		Username:      aws.String(email),
		MessageAction: types.MessageActionTypeSuppress,
		UserAttributes: []types.AttributeType{
			{Name: aws.String("email"), Value: aws.String(email)},
			{Name: aws.String("name"), Value: aws.String(name)},
			{Name: aws.String("email_verified"), Value: aws.String("true")},
		},
	}
	result, err := a.client.AdminCreateUser(ctx, input)
	if err != nil {
		return "", err
	}
	for _, attr := range result.User.Attributes {
		if aws.ToString(attr.Name) == "sub" {
			return aws.ToString(attr.Value), nil
		}
	}
	return "", fmt.Errorf("sub attribute not found in created user")
}

// generateSocialPassword returns a cryptographically random password that
// satisfies typical Cognito pool policies (upper, lower, digit, special char,
// min-length 8). The password is never stored or shown to the user.
func generateSocialPassword() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	// base64url gives A-Z a-z 0-9 - _; prepend fixed chars to satisfy every
	// character-class requirement regardless of pool policy configuration.
	return "Aa1!" + base64.RawURLEncoding.EncodeToString(b), nil
}
