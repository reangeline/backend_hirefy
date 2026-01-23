package openai

import (
	"context"
	"net/http"
	"time"
)

// Client é um wrapper para facilitar testes e reutilização
type Client struct {
	apiKey     string
	httpClient *http.Client
	baseURL    string
}

// NewClient cria um novo cliente OpenAI
func NewClient(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		baseURL: openaiAPIURL,
	}
}

// ChatCompletion faz uma chamada de chat completion
func (c *Client) ChatCompletion(ctx context.Context, req OpenAIRequest) (*OpenAIResponse, error) {
	// Implementação similar ao callOpenAI
	// Útil para testes e reutilização
	return nil, nil
}
