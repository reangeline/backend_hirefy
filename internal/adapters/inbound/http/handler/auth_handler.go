package handler

import (
	"encoding/json"
	"net/http"

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
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
	Name     string `json:"name" validate:"required"`
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
		Email:    req.Email,
		Password: req.Password,
		Name:     req.Name,
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
