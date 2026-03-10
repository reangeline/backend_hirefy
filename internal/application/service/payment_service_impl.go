package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
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
	return s.paymentGateway.CreateCustomer(ctx, email, name)

}

func (s *paymentServiceImpl) CreatePaymentMethod(ctx context.Context, req inbound.CreatePaymentMethodRequest) (string, error) {
	// O payment method já vem criado do frontend (Stripe.js)
	// Aqui só retornamos o token que foi passado
	return req.PaymentToken, nil
}

func (s *paymentServiceImpl) CreateCheckoutSession(ctx context.Context, req inbound.CreateCheckoutSessionRequest) (string, error) {
	// Mapeia plan para priceID
	priceID := s.getPriceIDForPlan(req.Plan)

	// CreateCheckoutSession agora aceita apenas userID, email, priceID
	// As URLs são fixas no adapter
	return s.paymentGateway.CreateCheckoutSession(
		ctx,
		req.UserID, // Assumindo que existe no request
		req.Email,  // Assumindo que existe no request
		priceID,
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

	// Verifica se a subscription já existe
	existingSub, err := s.subscriptionRepo.GetByStripeSubscriptionID(ctx, subscriptionID)
	if err == nil && existingSub != nil {
		// Já existe, apenas ativa
		existingSub.Status = domain.SubscriptionStatusActive
		existingSub.UpdatedAt = time.Now()
		return s.subscriptionRepo.Update(ctx, existingSub)
	}

	// Cria nova subscription
	subscription := &domain.Subscription{
		ID:               uuid.New().String(),
		UserID:           userID,
		Plan:             s.getPlanFromPriceID(data), // Detecta o plano automaticamente
		Status:           domain.SubscriptionStatusActive,
		StripeCustomerID: customerID,
		StripeSubID:      subscriptionID,
		CurrentPeriodEnd: time.Now().AddDate(0, 1, 0), // 1 mês
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	return s.subscriptionRepo.Create(ctx, subscription)
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

func (s *paymentServiceImpl) getPlanFromPriceID(data map[string]interface{}) domain.SubscriptionPlan {
	// Extrai o priceID do line_items ou usa default
	// Por enquanto, retorna o plano premium/básico
	// TODO: Mapear corretamente do priceID

	// Se você tiver PlanBasic ou PlanPremium, use um deles
	// Caso contrário, veja qual constante existe em domain.SubscriptionPlan
	return domain.PlanPremium // ou domain.PlanBasic, dependendo do que existir
}
