package handler

import (
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
		"created_at":     user.CreatedAt,
		"updated_at":     user.UpdatedAt,
	})
}
