package stripe

import (
	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/client"
)

// Client wrapper para o Stripe client
type Client struct {
	api *client.API
}

// NewClient cria novo cliente Stripe
func NewClient(apiKey string) *Client {
	stripe.Key = apiKey
	return &Client{
		api: &client.API{},
	}
}

// GetAPI retorna o cliente da API
func (c *Client) GetAPI() *client.API {
	return c.api
}
