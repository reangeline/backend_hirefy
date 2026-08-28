package handler

import (
	"encoding/json"
	"net/http"

	"github.com/reangeline/backend_applywise/internal/core/domain"
	"github.com/reangeline/backend_applywise/internal/core/ports/inbound"
)

type SubscriptionHandler struct {
	subscriptionService inbound.SubscriptionService
}

func NewSubscriptionHandler(subscriptionService inbound.SubscriptionService) *SubscriptionHandler {
	return &SubscriptionHandler{
		subscriptionService: subscriptionService,
	}
}

type CreateSubscriptionRequestDTO struct {
	Plan          string `json:"plan" validate:"required,oneof=basic premium"`
	PaymentMethod string `json:"payment_method" validate:"required"`
}

func (h *SubscriptionHandler) GetSubscription(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(string)

	subscription, err := h.subscriptionService.GetSubscriptionByUserID(r.Context(), userID)
	if err != nil {
		// Se não encontrar, retorna free tier
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"plan":   "free",
			"status": "active",
		})
		return
	}

	respondJSON(w, http.StatusOK, subscription)
}

func (h *SubscriptionHandler) CreateSubscription(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(string)

	var req CreateSubscriptionRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Converte string para SubscriptionPlan
	var plan domain.SubscriptionPlan
	switch req.Plan {
	case "basic":
		plan = domain.PlanBasic
	case "premium":
		plan = domain.PlanPremium
	default:
		respondError(w, http.StatusBadRequest, "invalid plan")
		return
	}

	subscription, err := h.subscriptionService.CreateSubscription(r.Context(), inbound.CreateSubscriptionRequest{
		UserID:        userID,
		Plan:          plan,
		PaymentMethod: req.PaymentMethod,
	})
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, subscription)
}

func (h *SubscriptionHandler) CancelSubscription(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(string)

	if err := h.subscriptionService.CancelSubscription(r.Context(), userID); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "subscription cancelled successfully",
	})
}

func (h *SubscriptionHandler) CheckStatus(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(string)

	isActive, err := h.subscriptionService.CheckSubscriptionStatus(r.Context(), userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]bool{
		"is_active": isActive,
	})
}

func (h *SubscriptionHandler) CreateCheckout(w http.ResponseWriter, r *http.Request) {
	// Não lê price_id do corpo — o cliente não escolhe o preço, o backend sempre usa o
	// Price ID do Premium configurado no servidor (achado de segurança, spec 007).
	userID := r.Context().Value("user_id").(string)
	user := r.Context().Value("user").(*domain.User)

	// Criar checkout session
	checkoutURL, err := h.subscriptionService.CreateCheckoutSession(r.Context(), inbound.CreateCheckoutRequest{
		UserID: userID,
		Email:  user.Email,
	})

	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to create checkout session")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"checkout_url": checkoutURL,
	})
}

func (h *SubscriptionHandler) GetCredits(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(string)

	// Buscar subscription para obter saldo de créditos
	subscription, err := h.subscriptionService.GetSubscriptionByUserID(r.Context(), userID)
	if err != nil {
		respondError(w, http.StatusNotFound, "subscription not found")
		return
	}

	// Buscar histórico de transações (últimas 50)
	transactions, err := h.subscriptionService.GetCreditHistory(r.Context(), userID, 50)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to get credit history")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"credits":      subscription.Credits,
		"plan":         subscription.Plan,
		"transactions": transactions,
	})
}
