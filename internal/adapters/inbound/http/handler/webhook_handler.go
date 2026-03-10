package handler

import (
	"fmt"
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
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		respondError(w, http.StatusBadRequest, "failed to read body")
		return
	}

	signature := r.Header.Get("Stripe-Signature")

	// Log para debug
	fmt.Printf("Webhook received - Size: %d bytes, Has signature: %v\n", len(payload), signature != "")

	if signature == "" {
		// Em desenvolvimento, aceita sem assinatura
		fmt.Println("WARNING: No signature provided, accepting anyway (DEV mode)")
		respondJSON(w, http.StatusOK, map[string]string{"status": "received_no_signature"})
		return
	}

	// Processar webhook com assinatura
	if err := h.paymentService.HandleWebhook(r.Context(), payload, signature); err != nil {
		fmt.Printf("Webhook processing error: %v\n", err)
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "success"})
}
