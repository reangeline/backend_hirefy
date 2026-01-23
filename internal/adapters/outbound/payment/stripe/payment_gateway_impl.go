package stripe

import (
	"context"
	"fmt"

	"github.com/reangeline/backend_applywise/internal/core/ports/outbound"
	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/checkout/session"
	"github.com/stripe/stripe-go/v76/customer"
	"github.com/stripe/stripe-go/v76/subscription"
	"github.com/stripe/stripe-go/v76/webhook"
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

func (p *paymentGatewayImpl) CreateCustomer(ctx context.Context, email, name string, metadata map[string]string) (string, error) {
	params := &stripe.CustomerParams{
		Email: stripe.String(email),
		Name:  stripe.String(name),
	}

	// Adiciona metadata
	if metadata != nil {
		params.Metadata = metadata
	}

	cust, err := customer.New(params)
	if err != nil {
		return "", fmt.Errorf("failed to create customer: %w", err)
	}

	return cust.ID, nil
}

func (p *paymentGatewayImpl) CreateSubscription(ctx context.Context, customerID, priceID, paymentMethodID string) (string, error) {
	// Primeiro, anexa o payment method ao customer
	if paymentMethodID != "" {
		params := &stripe.CustomerParams{
			InvoiceSettings: &stripe.CustomerInvoiceSettingsParams{
				DefaultPaymentMethod: stripe.String(paymentMethodID),
			},
		}
		if _, err := customer.Update(customerID, params); err != nil {
			return "", fmt.Errorf("failed to attach payment method: %w", err)
		}
	}

	// Cria a subscription
	params := &stripe.SubscriptionParams{
		Customer: stripe.String(customerID),
		Items: []*stripe.SubscriptionItemsParams{
			{
				Price: stripe.String(priceID),
			},
		},
		PaymentBehavior: stripe.String("default_incomplete"),
		PaymentSettings: &stripe.SubscriptionPaymentSettingsParams{
			SaveDefaultPaymentMethod: stripe.String("on_subscription"),
		},
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

func (p *paymentGatewayImpl) CreateCheckoutSession(ctx context.Context, customerID, priceID, successURL, cancelURL string) (string, error) {
	params := &stripe.CheckoutSessionParams{
		Customer: stripe.String(customerID),
		Mode:     stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(priceID),
				Quantity: stripe.Int64(1),
			},
		},
		SuccessURL: stripe.String(successURL),
		CancelURL:  stripe.String(cancelURL),
	}

	sess, err := session.New(params)
	if err != nil {
		return "", fmt.Errorf("failed to create checkout session: %w", err)
	}

	return sess.URL, nil
}

func (p *paymentGatewayImpl) VerifyWebhookSignature(payload []byte, signature, secret string) error {
	_, err := webhook.ConstructEvent(payload, signature, secret)
	if err != nil {
		return fmt.Errorf("webhook signature verification failed: %w", err)
	}

	return nil
}
