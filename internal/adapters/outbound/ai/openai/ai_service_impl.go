package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/reangeline/backend_applywise/internal/core/ports/outbound"
)

const (
	openaiAPIURL = "https://api.openai.com/v1/chat/completions"
	defaultModel = "gpt-4o-mini" // Mais barato para MVP
)

type aiServiceImpl struct {
	apiKey     string
	httpClient *http.Client
}

// NewAIService cria nova instância do serviço de IA
func NewAIService(apiKey string) outbound.AIService {
	return &aiServiceImpl{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func (s *aiServiceImpl) ParseResume(ctx context.Context, content string) (*outbound.ResumeAnalysis, error) {
	prompt := fmt.Sprintf(`Analyze the following resume and extract structured information.
						Return ONLY a JSON object with this exact structure (no markdown, no explanation):
						{
						"skills": ["skill1", "skill2"],
						"experience": ["experience description 1", "experience description 2"],
						"education": ["education 1", "education 2"],
						"keywords": ["keyword1", "keyword2"]
						}

						Resume:
						%s`, content)

	response, err := s.callOpenAI(ctx, prompt, 0.3)
	if err != nil {
		return nil, err
	}

	var analysis outbound.ResumeAnalysis
	if err := json.Unmarshal([]byte(response), &analysis); err != nil {
		return nil, fmt.Errorf("failed to parse resume analysis: %w", err)
	}

	return &analysis, nil
}

func (s *aiServiceImpl) ParseJobDescription(ctx context.Context, content string) (*outbound.JobAnalysis, error) {
	prompt := fmt.Sprintf(`Analyze the following job description and extract key requirements.
Return ONLY a JSON object with this exact structure (no markdown, no explanation):
{
  "required_skills": ["skill1", "skill2"],
  "keywords": ["keyword1", "keyword2"],
  "experience": "experience level description"
}

Job Description:
%s`, content)

	response, err := s.callOpenAI(ctx, prompt, 0.3)
	if err != nil {
		return nil, err
	}

	var analysis outbound.JobAnalysis
	if err := json.Unmarshal([]byte(response), &analysis); err != nil {
		return nil, fmt.Errorf("failed to parse job analysis: %w", err)
	}

	return &analysis, nil
}

func (s *aiServiceImpl) OptimizeResume(ctx context.Context, resume *outbound.ResumeAnalysis, job *outbound.JobAnalysis, originalResume string) (*outbound.OptimizationResult, error) {
	prompt := fmt.Sprintf(`You are an expert resume optimizer. Your task is to optimize a resume to better match a job description and improve ATS (Applicant Tracking System) rankings.

ORIGINAL RESUME:
%s

CANDIDATE'S PROFILE:
- Skills: %s
- Experience: %s
- Education: %s

JOB REQUIREMENTS:
- Required Skills: %s
- Keywords: %s
- Experience Level: %s

INSTRUCTIONS:
1. Rewrite the resume to better highlight relevant skills and experience
2. Incorporate job keywords naturally (don't keyword stuff)
3. Maintain truthfulness - don't add fake experience
4. Optimize formatting for ATS systems
5. Calculate a match score (0-100) based on skill alignment

Return ONLY a JSON object with this structure (no markdown):
{
  "optimized_content": "the optimized resume text",
  "match_score": 85.5,
  "suggestions": ["suggestion 1", "suggestion 2", "suggestion 3"]
}`,
		originalResume,
		strings.Join(resume.Skills, ", "),
		strings.Join(resume.Experience, "; "),
		strings.Join(resume.Education, "; "),
		strings.Join(job.RequiredSkills, ", "),
		strings.Join(job.Keywords, ", "),
		job.Experience,
	)

	response, err := s.callOpenAI(ctx, prompt, 0.7)
	if err != nil {
		return nil, err
	}

	fmt.Printf("Full OpenAI optimization response:\n%s\n", response)

	var result outbound.OptimizationResult
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		// Log do erro para debug
		fmt.Printf("Failed to parse OpenAI response: %v\nResponse: %s\n", err, response)
		return nil, fmt.Errorf("failed to parse optimization result: %w", err)
	}

	fmt.Printf("Parsed result - Content length: %d, Score: %.2f, Suggestions: %d\n",
		len(result.OptimizedContent), result.MatchScore, len(result.Suggestions))

	// Validação: se campos importantes estão vazios, retorna erro
	if result.OptimizedContent == "" && len(result.Suggestions) == 0 {
		return nil, fmt.Errorf("OpenAI returned empty optimization result")
	}

	return &result, nil
}

// callOpenAI faz chamada para a API do OpenAI
func (s *aiServiceImpl) callOpenAI(ctx context.Context, prompt string, temperature float64) (string, error) {
	requestBody := OpenAIRequest{
		Model: defaultModel,
		Messages: []Message{
			{
				Role:    "system",
				Content: "You are a professional resume optimization assistant. Always return valid JSON responses without markdown formatting.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Temperature: temperature,
		MaxTokens:   4000,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", openaiAPIURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.apiKey))

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to call OpenAI API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("OpenAI API error (status %d): %s", resp.StatusCode, string(body))
	}

	var openAIResp OpenAIResponse
	if err := json.Unmarshal(body, &openAIResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(openAIResp.Choices) == 0 {
		return "", fmt.Errorf("no response from OpenAI")
	}

	content := openAIResp.Choices[0].Message.Content

	// Limpeza agressiva de markdown e formatação
	content = strings.TrimSpace(content)

	// Remove ```json ou ``` do início
	if after, ok := strings.CutPrefix(content, "```json"); ok {
		content = after
	}
	if after, ok := strings.CutPrefix(content, "```"); ok {
		content = after
	}

	// Remove ``` do final
	content = strings.TrimSuffix(content, "```")

	content = strings.TrimSpace(content)

	// Log para debug (remover depois em produção)
	fmt.Printf("OpenAI response (first 200 chars): %s\n", content[:min(200, len(content))])

	return content, nil
}

// Helper function
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
