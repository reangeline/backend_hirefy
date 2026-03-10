package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/reangeline/backend_applywise/internal/core/ports/outbound"
)

const (
	openaiAPIURL              = "https://api.openai.com/v1/chat/completions"
	defaultModel              = "gpt-4o-mini" // Parse, salary: cheap and fast
	optimizationModel         = "gpt-4o"      // Resume & LinkedIn optimization: higher quality
	parseMaxTokens            = 800           // Sufficient for parse responses
	optimizeMaxTokens         = 1800          // Large enough for full optimization JSON
	linkedInMaxTokens         = 3200          // LinkedIn needs more tokens (longer about + suggestions)
	defaultHTTPTimeout        = 90 * time.Second
	perAttemptTimeout         = 25 * time.Second // regular calls
	linkedInPerAttemptTimeout = 45 * time.Second // LinkedIn is heavier; needs more time per attempt
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
			Timeout: defaultHTTPTimeout,
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

	response, err := s.callOpenAI(ctx, defaultModel, prompt, 0.3, parseMaxTokens)
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

	response, err := s.callOpenAI(ctx, defaultModel, prompt, 0.3, parseMaxTokens)
	if err != nil {
		return nil, err
	}

	var analysis outbound.JobAnalysis
	if err := json.Unmarshal([]byte(response), &analysis); err != nil {
		clean := sanitizeJSON(response)
		if clean == "" {
			return nil, fmt.Errorf("failed to parse job analysis: %w", err)
		}
		if err2 := json.Unmarshal([]byte(clean), &analysis); err2 != nil {
			return nil, fmt.Errorf("failed to parse job analysis (cleaned): %w", err2)
		}
	}

	return &analysis, nil
}

func (s *aiServiceImpl) OptimizeResume(ctx context.Context, resume *outbound.ResumeAnalysis, job *outbound.JobAnalysis, originalResume string) (*outbound.OptimizationResult, error) {
	// Para currículos manuais, originalResume pode estar vazio - usamos dados estruturados para contexto
	resumeData := map[string]interface{}{
		"skills":          resume.Skills,
		"experience":      resume.Experience,
		"education":       resume.Education,
		"keywords":        resume.Keywords,
		"structured_data": resume.StructuredData,
	}

	resumeDataJSON, _ := json.Marshal(resumeData)

	// Only ask the model to populate projects if the candidate actually has them.
	hasProjects := false
	if projects, ok := resume.StructuredData["projects"]; ok {
		switch v := projects.(type) {
		case []interface{}:
			hasProjects = len(v) > 0
		case []map[string]interface{}:
			hasProjects = len(v) > 0
		}
	}

	projectsBlock := `"projects": []`
	if hasProjects {
		projectsBlock = `"projects": [
      {
        "name": "keep original project name",
        "url": "keep original",
        "description": "rewrite to highlight relevant technologies and outcomes"
      }
    ]`
	}

	prompt := fmt.Sprintf(`You are an expert resume optimizer. Optimize the candidate's resume to better match the job requirements.

CRITICAL RULES (be concise):
1. NEVER invent or add experiences, skills, or education that the candidate doesn't have
2. ONLY enhance and rewrite existing information to better highlight relevant aspects
3. Use job keywords naturally where they genuinely apply to candidate's experience
4. Maintain 100%% truthfulness - optimize wording, not facts
5. Keep responses compact and non-repetitive

CANDIDATE'S CURRENT DATA:
%s

JOB REQUIREMENTS:
- Required Skills: %s
- Keywords: %s
- Experience Level: %s

TASK:
Analyze the candidate's data and optimize it for this job by:
- Rewriting descriptions to emphasize relevant skills and achievements
- Incorporating job keywords where they naturally fit the candidate's real experience
- Highlighting transferable skills
- Calculating match score (0-100) based on actual skill alignment
- Clearly list which required skills/experiences are missing or weak
- If required skills are largely absent, keep match_score low (<=20), avoid fabricating content, and focus on listing gaps and small truthful improvements only
- Evaluate experience depth: if there are few roles or descriptions are sparse, explicitly say experience is limited and propose concrete improvements (add metrics, outcomes, scope, stack); if there are many roles, suggest consolidating or prioritizing the most relevant
- For professional summary also use work experience and skills to rewrite it in a way that better highlights the candidate's fit for THIS job, without adding any new information

Return ONLY a JSON object with this EXACT structure (no markdown, no extra text):
{
  "optimized_content": "optimized resume as formatted text",
  "parsed_data": {
    "personal": {
      "full_name": "keep original if provided",
      "email": "keep original",
      "phone": "keep original",
      "current_role": "optimized to match job if relevant",
      "country": "keep original",
      "state": "keep original",
      "city": "keep original",
      "linkedin_url": "keep original",
      "website_url": "keep original",
      "github_url": "keep original",
      "summary": "rewritten professional summary highlighting relevant experience for THIS job"
    },
		"experiences": [
			{
				"role": "original role title",
				"company": "original company",
				"start_date": "keep original format",
				"end_date": "keep original or null",
				"is_current": true/false,
				"description": "rewrite as bullet list highlighting achievements and skills relevant to target job, using job keywords naturally"
			}
		],
    "education": [
      {
        "institution": "keep original",
        "degree": "keep original",
        "start_date": "keep original",
        "end_date": "keep original",
        "is_current": false
      }
    ],
    "projects": %s
  },
  "match_score": 85.5,
  "suggestions": [
    "specific suggestion 1 for improvement",
    "specific suggestion 2",
    "specific suggestion 3"
	],
	"missing_requirements": [
		"required skill or experience the candidate does not cover",
		"another missing or weak requirement",
		"techonology or keyword that should be better highlighted"
	]
}
Ensure optimized_content and descriptions stay succinct (avoid exceeding ~900 words total).
`,
		string(resumeDataJSON),
		strings.Join(job.RequiredSkills, ", "),
		strings.Join(job.Keywords, ", "),
		job.Experience,
		projectsBlock,
	)

	response, err := s.callOpenAI(ctx, optimizationModel, prompt, 0.9, optimizeMaxTokens)
	if err != nil {
		return nil, err
	}

	var result outbound.OptimizationResult
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		clean := sanitizeJSON(response)
		if clean == "" {
			return nil, fmt.Errorf("failed to parse optimization result: %w", err)
		}

		if err2 := json.Unmarshal([]byte(clean), &result); err2 != nil {
			return nil, fmt.Errorf("failed to parse optimization result: %w", err2)
		}
	}

	// Validação: se campos importantes estão vazios, retorna erro
	if result.OptimizedContent == "" && len(result.Suggestions) == 0 {
		return nil, fmt.Errorf("OpenAI returned empty optimization result")
	}

	return &result, nil
}

const salaryMaxTokens = 400

func (s *aiServiceImpl) OptimizeForLinkedIn(ctx context.Context, resume *outbound.ResumeAnalysis) (*outbound.LinkedInOptimizationResult, error) {
	resumeJSON, _ := json.Marshal(map[string]interface{}{
		"skills":          resume.Skills,
		"experience":      resume.Experience,
		"education":       resume.Education,
		"keywords":        resume.Keywords,
		"structured_data": resume.StructuredData,
	})

	prompt := fmt.Sprintf(`You are a LinkedIn profile optimization expert. Your goal is to maximize the candidate's chances of being found by recruiters and getting interviews.

════════════════════════════════
ABSOLUTE RULES
════════════════════════════════
1. NEVER invent skills, certifications, companies, or experiences the candidate does not have.
2. ONLY rephrase and restructure EXISTING information.
3. Respond ONLY with the JSON structure below — no markdown, no explanation, no extra text.
4. ALL fields are MANDATORY. Never omit "experiences", "languages", or any other field. If the source data has entries, you MUST include them in the output.

════════════════════════════════
HEADLINE — recruiter search optimization
════════════════════════════════
- LinkedIn's algorithm uses the headline as the primary keyword index for recruiter searches.
- Pack it with the most in-demand job titles and core technologies from the candidate's stack.
- Format: "[Primary Role] | [Secondary Role if applicable] | [Tech Stack keywords separated by ·] | [Domain/Industry]"
- Example: "Senior Backend Engineer | Golang · Node.js · Python | Microservices · gRPC · Kafka | Fintech"
- Max 220 characters. Prioritize searchability over creativity.

════════════════════════════════
ABOUT — first-person, 800–1800 characters
════════════════════════════════
- Written in first person, professional and compelling.
- Start with current role + years of experience + main domain.
- Mention core technologies, methodologies, and industries.
- Close with what the candidate is looking for / open to.

════════════════════════════════
EXPERIENCES — MANDATORY, use ALL entries from structured_data.experiences
════════════════════════════════
- You MUST output every experience found in the candidate data. Do not skip any.
- Each description must be an array of 2–3 bullet points, starting with a strong action verb.
- Include at least one metric or quantified outcome per bullet when the data allows it.
- Use keywords from the candidate's tech stack naturally inside each bullet.

════════════════════════════════
LANGUAGES — MANDATORY, use ALL entries from structured_data.languages
════════════════════════════════
- Output every language found in the candidate data. Do not skip any.
- Normalize the level to one of: Native, Fluent, Advanced, Intermediate, Basic.

════════════════════════════════
PROFILE STRENGTH SCORE — calculate honestly
════════════════════════════════
Apply this rubric to calculate a score from 0 to 100. Do NOT default to 85. Be precise:
- Headline has 4+ keywords relevant to the role (+20 pts)
- About section is 800–1800 characters and covers role, tech, and industry (+15 pts)  
- All experiences have 2+ quantified bullet points (+20 pts; deduct 5 per experience missing metrics)
- Skills section has 15+ relevant skills (+10 pts)
- Languages section has at least one entry (+5 pts)
- Profile has a clear professional narrative across sections (+10 pts)
- Missing key ecosystem tools that are commonly required for the candidate's role (-5 per major gap, min 0)
- Weak or generic headline with fewer than 3 searchable keywords (-10 pts)
Sum all points. The result is the profile_strength_score.

════════════════════════════════
ECOSYSTEM SUGGESTIONS + SSI TIPS
════════════════════════════════
Produce 6–10 suggestions total, mixing:

A) ECOSYSTEM suggestions (4–6): For each technology the candidate already uses, identify the most frequently requested complementary skills in that ecosystem from real job market demand:
   AWS → EKS, Lambda, CloudFront, RDS, IAM, CDK
   Docker → Kubernetes, Helm, containerd, Docker Compose
   React → Next.js, Redux, React Query, Storybook
   Python → FastAPI, Pandas, SQLAlchemy, Celery
   Terraform → Atlantis, Terragrunt, Pulumi
   Node.js → NestJS, Fastify, Bull, Prisma
   Go → gRPC, chi, sqlx, wire
   PostgreSQL → pgvector, TimescaleDB, pgBouncer
   Kafka → Schema Registry, ksqlDB, Kafka Connect
   Azure → AKS, Azure Functions, Cosmos DB, Azure DevOps
   Jenkins → ArgoCD, Tekton, GitHub Actions
   Format: "You use [X] — recruiters frequently pair it with [Y, Z]. Adding these to your profile can significantly increase visibility for [role type] searches."

B) SSI (Social Selling Index) tips (2–4): The LinkedIn SSI score directly impacts how often your profile appears in recruiter searches. Include this tip ALWAYS as one of the suggestions:
   "Check your LinkedIn SSI score at https://www.linkedin.com/sales/ssi (must be logged in). A score above 70 significantly increases your profile's visibility to recruiters."
   Then add 1–3 specific SSI improvement tips based on the candidate's profile gaps, chosen from:
   - "Establish your professional brand: publish or share at least 2 posts per month about your area of expertise to increase your 'Professional Brand' SSI pillar."
   - "Engage with insights: comment on posts from industry leaders and companies in your target sector to boost the 'Engage with Insights' SSI pillar."
   - "Build relationships: connect with recruiters and professionals in your target companies. Aim for 500+ connections in your field."
   - "Find the right people: follow companies you target and engage with their content to appear in their employees' feeds."

════════════════════════════════
CANDIDATE DATA
════════════════════════════════
%s

════════════════════════════════
OUTPUT — return ONLY this JSON, no other text
════════════════════════════════
{
  "headline": "keyword-rich headline for recruiter search (max 220 chars)",
  "about": "first-person About section 800–1800 characters",
  "experiences": [
    {
      "role": "exact role title from source data",
      "company": "exact company name from source data",
      "start_date": "keep original value",
      "end_date": "keep original value or null",
      "is_current": true,
      "description": [
        "Strong action verb + achievement or responsibility + metric or outcome",
        "Strong action verb + technology used + business impact"
      ]
    }
  ],
  "skills": ["top 20 skills from candidate data"],
  "languages": [
    {"name": "language from source data", "level": "Native|Fluent|Advanced|Intermediate|Basic"}
  ],
  "suggestions": [
    "You use [X] — recruiters frequently pair it with [Y, Z]. Adding these to your profile can significantly increase visibility for [role type] searches.",
    "Check your LinkedIn SSI score at https://www.linkedin.com/sales/ssi (must be logged in). A score above 70 significantly increases your profile visibility to recruiters.",
    "more ecosystem and SSI tips based on candidate data"
  ],
  "profile_strength_score": 0.0
}
`, string(resumeJSON))

	response, err := s.callOpenAI(ctx, optimizationModel, prompt, 0.7, linkedInMaxTokens, linkedInPerAttemptTimeout)
	if err != nil {
		return nil, err
	}

	var result outbound.LinkedInOptimizationResult
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		clean := sanitizeJSON(response)
		if clean == "" {
			return nil, fmt.Errorf("failed to parse LinkedIn optimization result: %w", err)
		}
		if err2 := json.Unmarshal([]byte(clean), &result); err2 != nil {
			return nil, fmt.Errorf("failed to parse LinkedIn optimization result (cleaned): %w", err2)
		}
	}

	if result.Headline == "" && result.About == "" {
		return nil, fmt.Errorf("OpenAI returned empty LinkedIn optimization result")
	}

	return &result, nil
}

func (s *aiServiceImpl) EstimateSalary(ctx context.Context, targetRole, targetCompany string) (*outbound.SalaryEstimate, error) {
	companyCtx := ""
	if targetCompany != "" {
		companyCtx = fmt.Sprintf(" at the company \"%s\"", targetCompany)
	}

	prompt := fmt.Sprintf(`You are a compensation data specialist. Based on publicly known market data, estimate the salary range for the role "%s"%s.

STRICT RULES:
1. Only use real, publicly known market compensation data (Glassdoor, LinkedIn Salary, Levels.fyi, etc.).
2. If you do NOT have reliable data for this specific role and/or company, set "found" to false and leave all numeric fields as 0.
3. NEVER invent or extrapolate numbers. If uncertain, set found=false.
4. When found=true, provide min, max, and midpoint values in the local currency of the company's primary market.
5. Specify whether the values are monthly or yearly in the "period" field.
6. Include a brief "disclaimer" reminding this is an estimate based on public data.

Return ONLY a JSON object with this EXACT structure (no markdown, no explanation):
{
  "found": true,
  "currency": "USD",
  "min_salary": 8000,
  "max_salary": 15000,
  "midpoint": 11500,
  "period": "monthly",
  "location": "USA",
  "seniority": "mid-level",
  "notes": "Range based on market data for similar roles in USA",
  "disclaimer": "Estimate based on publicly available market data. Actual compensation may vary."
}

If no reliable data is found return:
{"found": false, "currency": "", "min_salary": 0, "max_salary": 0, "midpoint": 0, "period": "", "location": "", "seniority": "", "notes": "", "disclaimer": ""}
`, targetRole, companyCtx)

	response, err := s.callOpenAI(ctx, defaultModel, prompt, 0.1, salaryMaxTokens)
	if err != nil {
		return nil, err
	}

	var estimate outbound.SalaryEstimate
	if err := json.Unmarshal([]byte(response), &estimate); err != nil {
		clean := sanitizeJSON(response)
		if clean == "" {
			return &outbound.SalaryEstimate{Found: false}, nil
		}
		if err2 := json.Unmarshal([]byte(clean), &estimate); err2 != nil {
			return &outbound.SalaryEstimate{Found: false}, nil
		}
	}

	return &estimate, nil
}

// callOpenAI faz chamada para a API do OpenAI.
// model: modelo a usar (defaultModel ou optimizationModel).
// attemptTimeout define o deadline por tentativa; passe 0 para usar o padrão (perAttemptTimeout).
func (s *aiServiceImpl) callOpenAI(ctx context.Context, model string, prompt string, temperature float64, maxTokens int, attemptTimeout ...time.Duration) (string, error) {
	perAttempt := perAttemptTimeout
	if len(attemptTimeout) > 0 && attemptTimeout[0] > 0 {
		perAttempt = attemptTimeout[0]
	}
	requestBody := OpenAIRequest{
		Model: model,
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
		Temperature:    temperature,
		MaxTokens:      maxTokens,
		ResponseFormat: &ResponseFormat{Type: "json_object"},
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	const maxAttempts = 2
	baseBackoff := 500 * time.Millisecond

	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		// Give each attempt its own independent deadline so a stalled
		// first attempt does not starve subsequent retries.
		attemptTimeout := computeRequestTimeout(ctx, perAttempt)
		if attemptTimeout <= 0 {
			break // parent context already expired; no point retrying
		}
		attemptCtx, cancel := context.WithTimeout(ctx, attemptTimeout)

		req, err := http.NewRequestWithContext(attemptCtx, "POST", openaiAPIURL, bytes.NewReader(jsonBody))
		if err != nil {
			cancel()
			return "", fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.apiKey))

		resp, err := s.httpClient.Do(req)
		if err != nil {
			cancel()
			lastErr = fmt.Errorf("failed to call OpenAI API: %w", err)
			// Do not retry if the parent context is already done.
			if ctx.Err() != nil {
				return "", lastErr
			}
			if attempt < maxAttempts {
				time.Sleep(time.Duration(attempt) * baseBackoff)
				continue
			}
			return "", lastErr
		}

		// Read body BEFORE canceling the context — the response stream
		// depends on the context remaining active until the read completes.
		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		cancel() // safe to release now that the body is fully consumed
		if readErr != nil {
			return "", fmt.Errorf("failed to read response: %w", readErr)
		}

		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("OpenAI API error (status %d): %s", resp.StatusCode, string(body))
			if attempt < maxAttempts {
				time.Sleep(time.Duration(1<<(attempt-1)) * baseBackoff)
				continue
			}
			return "", lastErr
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
		if openAIResp.Choices[0].FinishReason == "length" {
			return "", fmt.Errorf("OpenAI response truncated (finish_reason=length)")
		}

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

		return content, nil
	}

	return "", lastErr
}

// computeRequestTimeout picks the smaller of desiredTimeout and the time
// remaining in ctx (minus a safety margin so the Lambda can persist results).
func computeRequestTimeout(ctx context.Context, desiredTimeout time.Duration) time.Duration {
	allowed := desiredTimeout
	if deadline, ok := ctx.Deadline(); ok {
		const safetyMargin = 8 * time.Second
		remaining := time.Until(deadline) - safetyMargin
		if remaining < allowed {
			allowed = remaining
		}
	}
	if allowed < 2*time.Second {
		return 0 // signal: not enough time left
	}
	return allowed
}

// Helper function
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// sanitizeJSON tries to extract the first '{' through the last '}' to mitigate stray text or code fences.
func sanitizeJSON(s string) string {
	start := strings.Index(s, "{")
	if start == -1 {
		return ""
	}
	end := strings.LastIndex(s, "}")
	if end == -1 || end <= start {
		return ""
	}

	clean := strings.TrimSpace(s[start : end+1])
	clean = strings.TrimPrefix(clean, "json")

	// Fix common model mistakes that break JSON
	trailingCommaObj := regexp.MustCompile(`,\s*}`)
	trailingCommaArr := regexp.MustCompile(`,\s*]`)
	emptyArrayQuote := regexp.MustCompile(`\[\]\s*"\s*,`)

	clean = trailingCommaObj.ReplaceAllString(clean, "}")
	clean = trailingCommaArr.ReplaceAllString(clean, "]")
	clean = emptyArrayQuote.ReplaceAllString(clean, "[],")

	return strings.TrimSpace(clean)
}
