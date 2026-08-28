package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/reangeline/backend_applywise/internal/core/domain"
	"github.com/reangeline/backend_applywise/internal/core/ports/inbound"
	"github.com/reangeline/backend_applywise/internal/core/ports/outbound"
)

type paymentServiceImpl struct {
	paymentGateway   outbound.PaymentGateway
	subscriptionRepo outbound.SubscriptionRepository
	webhookSecret    string
	premiumPriceID   string
}

func NewPaymentService(
	paymentGateway outbound.PaymentGateway,
	subscriptionRepo outbound.SubscriptionRepository,
	webhookSecret string,
	premiumPriceID string,
) inbound.PaymentService {
	return &paymentServiceImpl{
		paymentGateway:   paymentGateway,
		subscriptionRepo: subscriptionRepo,
		webhookSecret:    webhookSecret,
		premiumPriceID:   premiumPriceID,
	}
}

func (s *paymentServiceImpl) CreateCustomer(ctx context.Context, userID, email, name string) (string, error) {
	return s.paymentGateway.CreateCustomer(ctx, email, name)

}

func (s *paymentServiceImpl) CreatePaymentMethod(ctx context.Context, req inbound.CreatePaymentMethodRequest) (string, error) {
	// O payment method já vem criado do frontend (Stripe.js)
	// Aqui só retornamos o token que foi passado
	return req.PaymentToken, nil
}

func (s *paymentServiceImpl) HandleWebhook(ctx context.Context, payload []byte, signature string) error {
	// Verifica a assinatura do webhook
	if err := s.paymentGateway.VerifyWebhookSignature(payload, signature, s.webhookSecret); err != nil {
		return fmt.Errorf("invalid webhook signature: %w", err)
	}

	// Parse o evento
	var event StripeEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return fmt.Errorf("failed to parse webhook: %w", err)
	}

	// Processa baseado no tipo de evento
	switch event.Type {
	case "checkout.session.completed":
		return s.handleCheckoutCompleted(ctx, event.Data.Object)

	case "customer.subscription.updated":
		return s.handleSubscriptionUpdated(ctx, event.Data.Object)

	case "customer.subscription.deleted":
		return s.handleSubscriptionDeleted(ctx, event.Data.Object)

	case "invoice.payment_succeeded":
		return s.handlePaymentSucceeded(ctx, event.Data.Object)

	case "invoice.payment_failed":
		return s.handlePaymentFailed(ctx, event.Data.Object)

	default:
		// Evento não tratado, mas não é erro
		return nil
	}
}

// Estruturas para parsing dos eventos do Stripe
type StripeEvent struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Data struct {
		Object map[string]interface{} `json:"object"`
	} `json:"data"`
}

func (s *paymentServiceImpl) handleCheckoutCompleted(ctx context.Context, data map[string]interface{}) error {
	// Extrair dados do checkout
	customerID, _ := data["customer"].(string)
	subscriptionID, _ := data["subscription"].(string)
	metadata, _ := data["metadata"].(map[string]interface{})
	userID, _ := metadata["user_id"].(string)

	if userID == "" {
		return fmt.Errorf("missing user_id in metadata")
	}

	if subscriptionID == "" {
		// Checkout sem subscription (payment único), ignora
		return nil
	}

	// Confirma qual price_id foi realmente comprado direto no Stripe, em vez de confiar no
	// payload do evento — só ativamos Premium se bater com o price configurado no servidor
	// (achado de segurança, spec 007: nunca assumir o plano sem checar).
	priceID, err := s.paymentGateway.GetSubscriptionPriceID(ctx, subscriptionID)
	if err != nil {
		return fmt.Errorf("failed to verify subscription price: %w", err)
	}
	if priceID != s.premiumPriceID {
		return fmt.Errorf("checkout completed with unexpected price_id %q (expected premium %q)", priceID, s.premiumPriceID)
	}

	// Atualiza a subscription que o usuário já tem (toda conta ganha uma linha Free no
	// signup — ver NewSubscription) em vez de criar uma segunda linha pro mesmo usuário.
	// Achado spec 007: criar uma nova linha aqui deixava 2 registros SUB# por usuário, e
	// GetByUserID (Query com Limit:1, sem ordenação por data) podia retornar a linha Free
	// antiga em vez da Premium nova, fazendo o upgrade parecer que não tinha funcionado.
	existingSub, err := s.subscriptionRepo.GetByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to load existing subscription for user %s: %w", userID, err)
	}

	existingSub.Plan = domain.PlanPremium
	existingSub.Status = domain.SubscriptionStatusActive
	existingSub.StripeCustomerID = customerID
	existingSub.StripeSubID = subscriptionID
	existingSub.CurrentPeriodEnd = time.Now().AddDate(0, 1, 0) // 1 mês
	existingSub.UpdatedAt = time.Now()

	return s.subscriptionRepo.Update(ctx, existingSub)
}

func (s *paymentServiceImpl) handleSubscriptionUpdated(ctx context.Context, data map[string]interface{}) error {
	subscriptionID, ok := data["id"].(string)
	if !ok {
		return fmt.Errorf("missing subscription id")
	}

	subscription, err := s.subscriptionRepo.GetByStripeSubscriptionID(ctx, subscriptionID)
	if err != nil {
		return err
	}

	// Atualiza status
	status, _ := data["status"].(string)
	switch status {
	case "active":
		subscription.Status = domain.SubscriptionStatusActive
	case "past_due":
		subscription.Status = domain.SubscriptionStatusPastDue
	case "canceled":
		subscription.Status = domain.SubscriptionStatusCanceled
		now := time.Now()
		subscription.CanceledAt = &now
	}

	// Atualiza current_period_end
	if periodEnd, ok := data["current_period_end"].(float64); ok {
		subscription.CurrentPeriodEnd = time.Unix(int64(periodEnd), 0)
	}

	subscription.UpdatedAt = time.Now()

	return s.subscriptionRepo.Update(ctx, subscription)
}

func (s *paymentServiceImpl) handleSubscriptionDeleted(ctx context.Context, data map[string]interface{}) error {
	subscriptionID, ok := data["id"].(string)
	if !ok {
		return fmt.Errorf("missing subscription id")
	}

	subscription, err := s.subscriptionRepo.GetByStripeSubscriptionID(ctx, subscriptionID)
	if err != nil {
		return err
	}

	subscription.Status = domain.SubscriptionStatusCanceled
	now := time.Now()
	subscription.CanceledAt = &now
	subscription.UpdatedAt = now

	return s.subscriptionRepo.Update(ctx, subscription)
}

func (s *paymentServiceImpl) handlePaymentSucceeded(ctx context.Context, data map[string]interface{}) error {
	subscriptionID, ok := data["subscription"].(string)
	if !ok {
		return nil // Pode ser um pagamento one-time
	}

	subscription, err := s.subscriptionRepo.GetByStripeSubscriptionID(ctx, subscriptionID)
	if err != nil {
		return err
	}

	// Renova a subscription
	if periodEnd, ok := data["period_end"].(float64); ok {
		subscription.Renew(time.Unix(int64(periodEnd), 0))
	}

	return s.subscriptionRepo.Update(ctx, subscription)
}

func (s *paymentServiceImpl) handlePaymentFailed(ctx context.Context, data map[string]interface{}) error {
	subscriptionID, ok := data["subscription"].(string)
	if !ok {
		return nil
	}

	subscription, err := s.subscriptionRepo.GetByStripeSubscriptionID(ctx, subscriptionID)
	if err != nil {
		return err
	}

	subscription.Status = domain.SubscriptionStatusPastDue
	subscription.UpdatedAt = time.Now()

	return s.subscriptionRepo.Update(ctx, subscription)
}
