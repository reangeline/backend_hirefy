package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/reangeline/backend_applywise/internal/core/domain"
	"github.com/reangeline/backend_applywise/internal/core/ports/inbound"
	"github.com/reangeline/backend_applywise/internal/core/ports/outbound"
)

type subscriptionServiceImpl struct {
	subscriptionRepo outbound.SubscriptionRepository
	paymentGateway   outbound.PaymentGateway
	userRepo         outbound.UserRepository
}

func NewSubscriptionService(
	subscriptionRepo outbound.SubscriptionRepository,
	paymentGateway outbound.PaymentGateway,
	userRepo outbound.UserRepository,
) inbound.SubscriptionService {
	return &subscriptionServiceImpl{
		subscriptionRepo: subscriptionRepo,
		paymentGateway:   paymentGateway,
		userRepo:         userRepo,
	}
}

func (s *subscriptionServiceImpl) CreateSubscription(
	ctx context.Context,
	req inbound.CreateSubscriptionRequest,
) (*domain.Subscription, error) {
	// Busca usuário
	user, err := s.userRepo.GetByID(ctx, req.UserID)
	if err != nil {
		return nil, domain.ErrUserNotFound
	}

	// Verifica se já tem assinatura
	existingSub, _ := s.subscriptionRepo.GetByUserID(ctx, req.UserID)
	if existingSub != nil && existingSub.IsActive() {
		return existingSub, nil
	}

	// Cria customer no Stripe se não existir
	stripeCustomerID := user.StripeCustomerID
	if stripeCustomerID == "" {
		stripeCustomerID, err = s.paymentGateway.CreateCustomer(
			ctx,
			user.Email,
			user.Name,
			map[string]string{"user_id": user.ID},
		)
		if err != nil {
			return nil, err
		}
	}

	// Mapeia plan para priceID do Stripe
	priceID := s.getPriceIDForPlan(req.Plan)

	// Cria subscription no Stripe
	stripeSubID, err := s.paymentGateway.CreateSubscription(
		ctx,
		stripeCustomerID,
		priceID,
		req.PaymentMethod,
	)
	if err != nil {
		return nil, domain.ErrPaymentFailed
	}

	// Cria subscription local
	subscription := domain.NewSubscription(req.UserID, req.Plan)
	subscription.ID = uuid.New().String()
	subscription.StripeCustomerID = stripeCustomerID
	subscription.StripeSubID = stripeSubID

	if err := s.subscriptionRepo.Create(ctx, subscription); err != nil {
		// Tenta cancelar no Stripe se falhar localmente
		_ = s.paymentGateway.CancelSubscription(ctx, stripeSubID)
		return nil, err
	}

	return subscription, nil
}

func (s *subscriptionServiceImpl) GetSubscriptionByUserID(ctx context.Context, userID string) (*domain.Subscription, error) {
	return s.subscriptionRepo.GetByUserID(ctx, userID)
}

func (s *subscriptionServiceImpl) CancelSubscription(ctx context.Context, userID string) error {
	subscription, err := s.subscriptionRepo.GetByUserID(ctx, userID)
	if err != nil {
		return domain.ErrSubscriptionNotFound
	}

	// Cancela no Stripe
	if subscription.StripeSubID != "" {
		if err := s.paymentGateway.CancelSubscription(ctx, subscription.StripeSubID); err != nil {
			return err
		}
	}

	// Atualiza localmente
	if err := subscription.Cancel(); err != nil {
		return err
	}

	return s.subscriptionRepo.Update(ctx, subscription)
}

func (s *subscriptionServiceImpl) UpdateSubscription(
	ctx context.Context,
	userID string,
	newPlan domain.SubscriptionPlan,
) (*domain.Subscription, error) {
	// TODO: Implementar upgrade/downgrade de plano
	return nil, nil
}

func (s *subscriptionServiceImpl) CheckSubscriptionStatus(ctx context.Context, userID string) (bool, error) {
	subscription, err := s.subscriptionRepo.GetByUserID(ctx, userID)
	if err != nil {
		return false, nil // Sem assinatura = free tier
	}

	return subscription.IsActive(), nil
}

func (s *subscriptionServiceImpl) getPriceIDForPlan(plan domain.SubscriptionPlan) string {
	// TODO: Buscar de variáveis de ambiente
	priceIDs := map[domain.SubscriptionPlan]string{
		domain.PlanBasic:   "price_basic_monthly",
		domain.PlanPremium: "price_premium_monthly",
	}
	return priceIDs[plan]
}
