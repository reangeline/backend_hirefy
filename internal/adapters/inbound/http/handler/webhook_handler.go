package handler

import (
	"io"
	"net/http"

	"github.com/reangeline/backend_applywise/internal/core/ports/inbound"
)

type WebhookHandler struct {
	paymentService inbound.PaymentService
}

func NewWebhookHandler(paymentService inbound.PaymentService) *WebhookHandler {
	return &WebhookHandler{
		paymentService: paymentService,
	}
}

func (h *WebhookHandler) HandleStripeWebhook(w http.ResponseWriter, r *http.Request) {
	// Lê o body do request
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		respondError(w, http.StatusBadRequest, "error reading request body")
		return
	}

	// Pega a assinatura do header
	signature := r.Header.Get("Stripe-Signature")
	if signature == "" {
		respondError(w, http.StatusBadRequest, "missing stripe signature")
		return
	}

	// Processa o webhook
	if err := h.paymentService.HandleWebhook(r.Context(), payload, signature); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Retorna 200 para o Stripe saber que recebemos
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"received": true}`))
}
