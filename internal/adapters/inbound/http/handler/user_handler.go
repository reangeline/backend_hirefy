package handler

import (
	"encoding/json"
	"net/http"

	"github.com/reangeline/backend_applywise/internal/core/ports/inbound"
)

type UserHandler struct {
	userService inbound.UserService
}

func NewUserHandler(userService inbound.UserService) *UserHandler {
	return &UserHandler{
		userService: userService,
	}
}

func (h *UserHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	// Pegar userID do contexto (adicionado pelo middleware)
	userID, ok := r.Context().Value("user_id").(string)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not found in context")
		return
	}

	// Buscar user no DynamoDB
	user, err := h.userService.GetUserByID(r.Context(), userID)
	if err != nil {
		respondError(w, http.StatusNotFound, "user not found")
		return
	}

	// Retornar dados do user
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"id":             user.ID,
		"email":          user.Email,
		"name":           user.Name,
		"email_verified": user.EmailVerified,
		"fcm_token":      user.FCMToken,
		"created_at":     user.CreatedAt,
		"updated_at":     user.UpdatedAt,
	})
}

type updateFCMTokenRequest struct {
	FCMToken string `json:"fcm_token"`
}

func (h *UserHandler) UpdateFCMToken(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(string)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not found in context")
		return
	}

	var req updateFCMTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.FCMToken == "" {
		respondError(w, http.StatusBadRequest, "fcm_token is required")
		return
	}

	if err := h.userService.UpdateFCMToken(r.Context(), userID, req.FCMToken); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *UserHandler) DeleteMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(string)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not found in context")
		return
	}

	if err := h.userService.DeleteUser(r.Context(), userID); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to delete account")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
