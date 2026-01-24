package stripe

import (
	"context"
	"fmt"

	"github.com/reangeline/backend_applywise/internal/core/ports/outbound"
	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/checkout/session"
	"github.com/stripe/stripe-go/v81/customer"
	"github.com/stripe/stripe-go/v81/subscription"
	"github.com/stripe/stripe-go/v81/webhook"
)

type paymentGatewayImpl struct {
	apiKey string
}

// NewPaymentGateway cria nova instância do gateway de pagamento
func NewPaymentGateway(apiKey string) outbound.PaymentGateway {
	stripe.Key = apiKey
	return &paymentGatewayImpl{
		apiKey: apiKey,
	}
}

func (p *paymentGatewayImpl) CreateCustomer(ctx context.Context, email, name string) (string, error) {
	params := &stripe.CustomerParams{
		Email: stripe.String(email),
		Name:  stripe.String(name),
	}

	cust, err := customer.New(params)
	if err != nil {
		return "", fmt.Errorf("failed to create customer: %w", err)
	}

	return cust.ID, nil
}

func (p *paymentGatewayImpl) CreateSubscription(ctx context.Context, customerID, priceID string) (string, error) {
	params := &stripe.SubscriptionParams{
		Customer: stripe.String(customerID),
		Items: []*stripe.SubscriptionItemsParams{
			{
				Price: stripe.String(priceID),
			},
		},
		PaymentBehavior: stripe.String("default_incomplete"),
		Expand: []*string{
			stripe.String("latest_invoice.payment_intent"),
		},
	}

	sub, err := subscription.New(params)
	if err != nil {
		return "", fmt.Errorf("failed to create subscription: %w", err)
	}

	return sub.ID, nil
}

func (p *paymentGatewayImpl) CancelSubscription(ctx context.Context, subscriptionID string) error {
	params := &stripe.SubscriptionCancelParams{}

	_, err := subscription.Cancel(subscriptionID, params)
	if err != nil {
		return fmt.Errorf("failed to cancel subscription: %w", err)
	}

	return nil
}

func (p *paymentGatewayImpl) CreateCheckoutSession(ctx context.Context, userID, email, priceID string) (string, error) {
	params := &stripe.CheckoutSessionParams{
		Mode: stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(priceID),
				Quantity: stripe.Int64(1),
			},
		},
		CustomerEmail:     stripe.String(email),
		ClientReferenceID: stripe.String(userID),
		SuccessURL:        stripe.String("https://app.applywise.com/success?session_id={CHECKOUT_SESSION_ID}"),
		CancelURL:         stripe.String("https://app.applywise.com/pricing"),
		Metadata: map[string]string{
			"user_id": userID,
		},
	}

	sess, err := session.New(params)
	if err != nil {
		return "", fmt.Errorf("failed to create checkout session: %w", err)
	}

	return sess.URL, nil
}

func (p *paymentGatewayImpl) VerifyWebhookSignature(payload []byte, signature, secret string) error {
	_, err := webhook.ConstructEventWithOptions(
		payload,
		signature,
		secret,
		webhook.ConstructEventOptions{
			IgnoreAPIVersionMismatch: true,
		},
	)
	if err != nil {
		return fmt.Errorf("webhook signature verification failed: %w", err)
	}

	return nil
}
