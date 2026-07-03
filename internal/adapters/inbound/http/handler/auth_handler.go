package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/reangeline/backend_applywise/internal/core/ports/inbound"
)

type AuthHandler struct {
	authService inbound.AuthService
	userService inbound.UserService
}

func NewAuthHandler(authService inbound.AuthService, userService inbound.UserService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		userService: userService,
	}
}

type SignUpRequestDTO struct {
	Email           string                 `json:"email" validate:"required,email"`
	Password        string                 `json:"password" validate:"required,min=8"`
	Name            string                 `json:"name" validate:"required"`
	TermsAcceptedAt string                 `json:"terms_accepted_at"`
	TermsVersion    string                 `json:"terms_version"`
	ParsedResume    map[string]interface{} `json:"parsed_resume,omitempty"`
}

type SignInRequestDTO struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

func (h *AuthHandler) SignUp(w http.ResponseWriter, r *http.Request) {
	var req SignUpRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// O authService.SignUp já cria o usuário no Cognito E no DynamoDB
	authResp, err := h.authService.SignUp(r.Context(), inbound.SignUpRequest{
		Email:           req.Email,
		Password:        req.Password,
		Name:            req.Name,
		TermsAcceptedAt: req.TermsAcceptedAt,
		TermsVersion:    req.TermsVersion,
		ParsedResume:    req.ParsedResume,
	})
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Retorna apenas a resposta de autenticação
	respondJSON(w, http.StatusCreated, authResp)
}

func (h *AuthHandler) SignIn(w http.ResponseWriter, r *http.Request) {
	var req SignInRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	authResp, err := h.authService.SignIn(r.Context(), inbound.SignInRequest{
		Email:    req.Email,
		Password: req.Password,
	})

	if err != nil {
		respondError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	respondJSON(w, http.StatusOK, authResp)
}

func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	authResp, err := h.authService.RefreshToken(r.Context(), req.RefreshToken)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "invalid refresh token")
		return
	}

	respondJSON(w, http.StatusOK, authResp)
}

func (h *AuthHandler) ConfirmSignUp(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
		Code  string `json:"code"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	confirmReq := inbound.ConfirmSignUpRequest{
		Email: req.Email,
		Code:  req.Code,
	}

	if err := h.authService.ConfirmSignUp(r.Context(), confirmReq); err != nil {
		// Parsear tipo de erro do Cognito
		statusCode, errorType, message := parseAuthError(err)
		respondJSON(w, statusCode, map[string]string{
			"error":   errorType,
			"message": message,
		})
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "Email verified successfully",
	})
}

// ✅ ADICIONAR RESEND CODE HANDLER
func (h *AuthHandler) ResendCode(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resendReq := inbound.ResendCodeRequest{
		Email: req.Email,
	}

	if err := h.authService.ResendConfirmationCode(r.Context(), resendReq); err != nil {
		statusCode, errorType, message := parseAuthError(err)
		respondJSON(w, statusCode, map[string]string{
			"error":   errorType,
			"message": message,
		})
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "Verification code sent successfully",
	})
}

// ✅ HELPER PARA PARSEAR ERROS DO COGNITO
func parseAuthError(err error) (statusCode int, errorType string, message string) {
	errMsg := err.Error()

	// Detectar tipo de erro pela mensagem
	switch {
	case strings.Contains(errMsg, "CodeMismatchException"):
		return http.StatusBadRequest, "CodeMismatchException", "Invalid verification code provided"

	case strings.Contains(errMsg, "ExpiredCodeException"):
		return http.StatusBadRequest, "ExpiredCodeException", "Verification code has expired, please request a new one"

	case strings.Contains(errMsg, "UserNotFoundException"):
		return http.StatusNotFound, "UserNotFoundException", "User does not exist"

	case strings.Contains(errMsg, "NotAuthorizedException"):
		return http.StatusForbidden, "NotAuthorizedException", "User cannot be confirmed. User may already be confirmed"

	case strings.Contains(errMsg, "TooManyRequestsException"):
		return http.StatusTooManyRequests, "TooManyRequestsException", "Too many requests, please try again later"

	case strings.Contains(errMsg, "email is required"), strings.Contains(errMsg, "confirmation code is required"):
		return http.StatusBadRequest, "InvalidRequest", errMsg

	case strings.Contains(errMsg, "failed to send email"):
		return http.StatusInternalServerError, "EmailDeliveryError", "Failed to send verification email. Please try again later."

	case strings.Contains(errMsg, "user not found"):
		return http.StatusNotFound, "UserNotFound", "No account found with that email address"

	case strings.Contains(errMsg, "email already verified"):
		return http.StatusBadRequest, "AlreadyVerified", "This email is already verified"

	case strings.Contains(errMsg, "failed to create verification code"):
		return http.StatusInternalServerError, "InternalError", "Failed to generate verification code. Please try again."

	default:
		return http.StatusInternalServerError, "InternalError", "An unexpected error occurred"
	}
}

func (h *AuthHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.authService.ForgotPassword(r.Context(), inbound.ForgotPasswordRequest{
		Email: req.Email,
	}); err != nil {
		statusCode, errorType, message := parseAuthError(err)
		respondJSON(w, statusCode, map[string]string{
			"error":   errorType,
			"message": message,
		})
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "Password reset code sent to your email",
	})
}

func (h *AuthHandler) ConfirmForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email       string `json:"email"`
		Code        string `json:"code"`
		NewPassword string `json:"new_password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.authService.ConfirmForgotPassword(r.Context(), inbound.ConfirmForgotPasswordRequest{
		Email:       req.Email,
		Code:        req.Code,
		NewPassword: req.NewPassword,
	}); err != nil {
		statusCode, errorType, message := parseAuthError(err)
		respondJSON(w, statusCode, map[string]string{
			"error":   errorType,
			"message": message,
		})
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "Password reset successfully",
	})
}

type SocialSignInRequestDTO struct {
	Provider string `json:"provider"`
	IDToken  string `json:"id_token"`
	Name     string `json:"name"`
}

func (h *AuthHandler) SocialSignIn(w http.ResponseWriter, r *http.Request) {
	var req SocialSignInRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Provider == "" || req.IDToken == "" {
		respondError(w, http.StatusBadRequest, "provider and id_token are required")
		return
	}
	if req.Provider != "apple" && req.Provider != "google" {
		respondError(w, http.StatusBadRequest, "provider must be \"apple\" or \"google\"")
		return
	}

	authResp, err := h.authService.SocialSignIn(r.Context(), inbound.SocialSignInRequest{
		Provider: req.Provider,
		IDToken:  req.IDToken,
		Name:     req.Name,
	})
	if err != nil {
		errMsg := err.Error()
		switch {
		case strings.Contains(errMsg, "token validation failed"),
			strings.Contains(errMsg, "invalid Apple token"),
			strings.Contains(errMsg, "invalid Google token"):
			respondError(w, http.StatusUnauthorized, "invalid or expired social token")
		case strings.Contains(errMsg, "email not available in token"):
			respondError(w, http.StatusUnauthorized, errMsg)
		case strings.Contains(errMsg, "provider") && strings.Contains(errMsg, "must be"):
			respondError(w, http.StatusBadRequest, errMsg)
		default:
			respondError(w, http.StatusInternalServerError, "social sign-in failed")
		}
		return
	}

	respondJSON(w, http.StatusOK, authResp)
}
