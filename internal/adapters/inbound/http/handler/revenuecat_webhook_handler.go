package handler

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/reangeline/backend_applywise/internal/core/ports/inbound"
)

type RevenueCatWebhookHandler struct {
	revenueCatService inbound.RevenueCatService
	webhookSecret     string
}

func NewRevenueCatWebhookHandler(revenueCatService inbound.RevenueCatService) *RevenueCatWebhookHandler {
	return &RevenueCatWebhookHandler{
		revenueCatService: revenueCatService,
		webhookSecret:     os.Getenv("REVENUECAT_WEBHOOK_SECRET"),
	}
}

func (h *RevenueCatWebhookHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	// Ler body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		fmt.Printf("❌ Failed to read webhook body: %v\n", err)
		respondError(w, http.StatusBadRequest, "failed to read body")
		return
	}

	// Log para debug
	fmt.Printf("%s", "\n"+strings.Repeat("=", 60)+"\n")
	fmt.Printf("📱 REVENUECAT WEBHOOK RECEIVED\n")
	fmt.Printf("%s", strings.Repeat("=", 60)+"\n")
	fmt.Printf("Body size: %d bytes\n", len(body))

	// Validar signature (se configurado)
	signature := r.Header.Get("X-Revenuecat-Signature")
	if h.webhookSecret != "" && signature != "" {
		if !h.validateSignature(body, signature) {
			fmt.Printf("❌ Invalid signature\n")
			respondError(w, http.StatusUnauthorized, "invalid signature")
			return
		}
		fmt.Printf("✅ Signature validated\n")
	} else {
		fmt.Printf("⚠️ No signature validation (webhook secret not configured)\n")
	}

	// Parse JSON
	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		fmt.Printf("❌ Failed to parse JSON: %v\n", err)
		respondError(w, http.StatusBadRequest, "invalid json")
		return
	}

	// Log payload
	prettyJSON, _ := json.MarshalIndent(payload, "", "  ")
	fmt.Printf("Payload:\n%s\n", string(prettyJSON))

	// Extrair evento
	event, ok := payload["event"].(map[string]interface{})
	if !ok {
		fmt.Printf("❌ Missing 'event' field\n")
		respondError(w, http.StatusBadRequest, "missing event field")
		return
	}

	if err := h.revenueCatService.ProcessWebhookEvent(r.Context(), event); err != nil {
		fmt.Printf("❌ Failed to process event: %v\n", err)
		respondError(w, http.StatusInternalServerError, "failed to process event")
		return
	}

	fmt.Printf("✅ Webhook processed successfully\n")
	fmt.Printf("%s", strings.Repeat("=", 60)+"\n\n")

	respondJSON(w, http.StatusOK, map[string]string{
		"status": "success",
	})
}

// validateSignature valida a assinatura do webhook usando HMAC-SHA256
func (h *RevenueCatWebhookHandler) validateSignature(body []byte, signature string) bool {
	if h.webhookSecret == "" {
		return true // Aceita sem validação se não tiver secret configurado
	}

	// Calcular HMAC
	mac := hmac.New(sha256.New, []byte(h.webhookSecret))
	mac.Write(body)
	expectedMAC := hex.EncodeToString(mac.Sum(nil))

	// Comparar
	return hmac.Equal([]byte(signature), []byte(expectedMAC))
}
