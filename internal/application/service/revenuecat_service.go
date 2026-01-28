package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/reangeline/backend_applywise/internal/core/domain"
	"github.com/reangeline/backend_applywise/internal/core/ports/inbound"
	"github.com/reangeline/backend_applywise/internal/core/ports/outbound"
)

type revenueCatServiceImpl struct {
	subscriptionRepo outbound.SubscriptionRepository
	userRepo         outbound.UserRepository
}

func NewRevenueCatService(
	subscriptionRepo outbound.SubscriptionRepository,
	userRepo outbound.UserRepository,
) inbound.RevenueCatService {
	return &revenueCatServiceImpl{
		subscriptionRepo: subscriptionRepo,
		userRepo:         userRepo,
	}
}

func (s *revenueCatServiceImpl) ProcessWebhookEvent(ctx context.Context, event map[string]interface{}) error {
	eventType, ok := event["type"].(string)
	if !ok {
		return fmt.Errorf("missing event type")
	}

	fmt.Printf("📱 RevenueCat Event: %s\n", eventType)

	switch eventType {
	case "INITIAL_PURCHASE":
		return s.handleInitialPurchase(ctx, event)
	case "RENEWAL":
		return s.handleRenewal(ctx, event)
	case "CANCELLATION":
		return s.handleCancellation(ctx, event)
	case "EXPIRATION":
		return s.handleExpiration(ctx, event)
	case "NON_RENEWING_PURCHASE":
		return s.handleNonRenewingPurchase(ctx, event)
	case "UNCANCELLATION":
		return s.handleUncancellation(ctx, event)
	case "BILLING_ISSUE":
		return s.handleBillingIssue(ctx, event)
	default:
		fmt.Printf("⚠️ Unhandled event type: %s\n", eventType)
		return nil
	}
}

func (s *revenueCatServiceImpl) handleInitialPurchase(ctx context.Context, event map[string]interface{}) error {
	appUserID, _ := event["app_user_id"].(string)
	productID, _ := event["product_id"].(string)
	store, _ := event["store"].(string)
	transactionID, _ := event["original_transaction_id"].(string)
	expiresAtMs, _ := event["expiration_at_ms"].(float64)
	entitlements, _ := event["entitlement_ids"].([]interface{})

	fmt.Printf("✅ Initial Purchase - CognitoID: %s, Product: %s, TransactionID: %s\n", appUserID, productID, transactionID)

	// Verificar entitlement
	hasPremium := false
	for _, ent := range entitlements {
		if entStr, ok := ent.(string); ok && entStr == "ApplyWise Premium" {
			hasPremium = true
			break
		}
	}

	if !hasPremium {
		fmt.Printf("⚠️ No premium entitlement, skipping\n")
		return nil
	}

	// ✅ MUDANÇA CRÍTICA: Buscar user pelo CognitoID
	user, err := s.userRepo.GetByCognitoID(ctx, appUserID)
	if err != nil {
		fmt.Printf("❌ User not found with CognitoID: %s (error: %v)\n", appUserID, err)
		return fmt.Errorf("user not found: %w", err)
	}

	fmt.Printf("✅ Found user: ID=%s, Email=%s\n", user.ID, user.Email)

	// Converter dados
	storeEnum := s.mapStore(store)
	expiresAt := time.Unix(int64(expiresAtMs)/1000, 0)

	// ✅ Buscar subscription do USER (não do CognitoID)
	subscription, err := s.subscriptionRepo.GetByUserID(ctx, user.ID)

	if err != nil {
		// Não tem subscription - criar
		fmt.Printf("ℹ️ User has no subscription, creating new one\n")

		subscription = domain.NewSubscription(user.ID, domain.PlanPremium)
		subscription.ID = uuid.New().String()
		subscription.UpdateFromRevenueCat(appUserID, transactionID, productID, storeEnum, expiresAt, true)

		return s.subscriptionRepo.Create(ctx, subscription)
	}

	// ✅ IDEMPOTÊNCIA: Verificar se já processou essa transação
	if subscription.RevenueCatOriginalTransactionID == transactionID {
		fmt.Printf("⚠️ Transaction %s already processed, skipping (idempotent)\n", transactionID)
		return nil
	}

	// ✅ Atualizar subscription existente
	fmt.Printf("✅ Updating subscription from %s to premium (TransactionID: %s → %s)\n",
		subscription.Plan, subscription.RevenueCatOriginalTransactionID, transactionID)

	subscription.UpdateFromRevenueCat(appUserID, transactionID, productID, storeEnum, expiresAt, true)

	return s.subscriptionRepo.Update(ctx, subscription)
}

func (s *revenueCatServiceImpl) handleRenewal(ctx context.Context, event map[string]interface{}) error {
	appUserID, _ := event["app_user_id"].(string) // CognitoID
	expiresAtMs, _ := event["expiration_at_ms"].(float64)

	fmt.Printf("🔄 Renewal - CognitoID: %s\n", appUserID)

	// Buscar user pelo CognitoID
	user, err := s.userRepo.GetByCognitoID(ctx, appUserID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// Buscar subscription do user
	subscription, err := s.subscriptionRepo.GetByUserID(ctx, user.ID)
	if err != nil {
		return fmt.Errorf("subscription not found: %w", err)
	}

	expiresAt := time.Unix(int64(expiresAtMs)/1000, 0)
	subscription.Renew(expiresAt)

	return s.subscriptionRepo.Update(ctx, subscription)
}

func (s *revenueCatServiceImpl) handleCancellation(ctx context.Context, event map[string]interface{}) error {
	appUserID, _ := event["app_user_id"].(string) // CognitoID

	fmt.Printf("❌ Cancellation - CognitoID: %s\n", appUserID)

	user, err := s.userRepo.GetByCognitoID(ctx, appUserID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	subscription, err := s.subscriptionRepo.GetByUserID(ctx, user.ID)
	if err != nil {
		return fmt.Errorf("subscription not found: %w", err)
	}

	if err := subscription.Cancel(); err != nil {
		return err
	}

	return s.subscriptionRepo.Update(ctx, subscription)
}

func (s *revenueCatServiceImpl) handleExpiration(ctx context.Context, event map[string]interface{}) error {
	appUserID, _ := event["app_user_id"].(string) // CognitoID

	fmt.Printf("⏰ Expiration - CognitoID: %s\n", appUserID)

	user, err := s.userRepo.GetByCognitoID(ctx, appUserID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	subscription, err := s.subscriptionRepo.GetByUserID(ctx, user.ID)
	if err != nil {
		return fmt.Errorf("subscription not found: %w", err)
	}

	subscription.Expire()
	subscription.Plan = domain.PlanFree

	return s.subscriptionRepo.Update(ctx, subscription)
}

func (s *revenueCatServiceImpl) handleNonRenewingPurchase(ctx context.Context, event map[string]interface{}) error {
	fmt.Printf("💰 Non-Renewing Purchase\n")
	return nil
}

func (s *revenueCatServiceImpl) handleUncancellation(ctx context.Context, event map[string]interface{}) error {
	appUserID, _ := event["app_user_id"].(string) // CognitoID
	expiresAtMs, _ := event["expiration_at_ms"].(float64)

	fmt.Printf("✅ Uncancellation - CognitoID: %s\n", appUserID)

	user, err := s.userRepo.GetByCognitoID(ctx, appUserID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	subscription, err := s.subscriptionRepo.GetByUserID(ctx, user.ID)
	if err != nil {
		return fmt.Errorf("subscription not found: %w", err)
	}

	expiresAt := time.Unix(int64(expiresAtMs)/1000, 0)
	subscription.Renew(expiresAt)

	return s.subscriptionRepo.Update(ctx, subscription)
}

func (s *revenueCatServiceImpl) handleBillingIssue(ctx context.Context, event map[string]interface{}) error {
	appUserID, _ := event["app_user_id"].(string) // CognitoID

	fmt.Printf("⚠️ Billing Issue - CognitoID: %s\n", appUserID)

	user, err := s.userRepo.GetByCognitoID(ctx, appUserID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	subscription, err := s.subscriptionRepo.GetByUserID(ctx, user.ID)
	if err != nil {
		return fmt.Errorf("subscription not found: %w", err)
	}

	subscription.Status = domain.SubscriptionStatusPastDue
	subscription.UpdatedAt = time.Now()

	return s.subscriptionRepo.Update(ctx, subscription)
}

func (s *revenueCatServiceImpl) mapStore(store string) domain.Store {
	switch store {
	case "APP_STORE":
		return domain.StoreAppStore
	case "PLAY_STORE":
		return domain.StorePlayStore
	default:
		return domain.StoreNone
	}
}
