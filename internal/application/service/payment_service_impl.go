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
}

func NewPaymentService(
	paymentGateway outbound.PaymentGateway,
	subscriptionRepo outbound.SubscriptionRepository,
	webhookSecret string,
) inbound.PaymentService {
	return &paymentServiceImpl{
		paymentGateway:   paymentGateway,
		subscriptionRepo: subscriptionRepo,
		webhookSecret:    webhookSecret,
	}
}

func (s *paymentServiceImpl) CreateCustomer(ctx context.Context, userID, email, name string) (string, error) {
	return s.paymentGateway.CreateCustomer(ctx, email, name, map[string]string{
		"user_id": userID,
	})
}

func (s *paymentServiceImpl) CreatePaymentMethod(ctx context.Context, req inbound.CreatePaymentMethodRequest) (string, error) {
	// O payment method já vem criado do frontend (Stripe.js)
	// Aqui só retornamos o token que foi passado
	return req.PaymentToken, nil
}

func (s *paymentServiceImpl) CreateCheckoutSession(ctx context.Context, req inbound.CreateCheckoutSessionRequest) (string, error) {
	// Busca ou cria customer
	// Nota: Você precisaria buscar o user primeiro para pegar o StripeCustomerID
	// Por simplicidade, vou assumir que já existe

	// Mapeia plan para priceID
	priceID := s.getPriceIDForPlan(req.Plan)

	return s.paymentGateway.CreateCheckoutSession(
		ctx,
		"", // customerID - deveria ser passado ou buscado
		priceID,
		req.SuccessURL,
		req.CancelURL,
	)
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
	subscriptionID, ok := data["subscription"].(string)
	if !ok {
		return fmt.Errorf("missing subscription id")
	}

	// Busca a subscription
	subscription, err := s.subscriptionRepo.GetByStripeSubscriptionID(ctx, subscriptionID)
	if err != nil {
		return fmt.Errorf("subscription not found: %w", err)
	}

	// Ativa a subscription
	subscription.Status = domain.SubscriptionStatusActive
	subscription.UpdatedAt = time.Now()

	return s.subscriptionRepo.Update(ctx, subscription)
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

func (s *paymentServiceImpl) getPriceIDForPlan(plan string) string {
	// TODO: Buscar de variáveis de ambiente
	priceIDs := map[string]string{
		"basic":   "price_basic_monthly",
		"premium": "price_premium_monthly",
	}
	return priceIDs[plan]
}
