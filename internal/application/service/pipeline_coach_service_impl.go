package service

import (
	"context"
	"log"
	"strings"

	"github.com/reangeline/backend_applywise/internal/core/domain"
	"github.com/reangeline/backend_applywise/internal/core/ports/inbound"
	"github.com/reangeline/backend_applywise/internal/core/ports/outbound"
	"github.com/reangeline/backend_applywise/pkg/security"
)

type pipelineCoachServiceImpl struct {
	pipelineRepo outbound.PipelineRepository
	aiService    outbound.AIService
	subRepo      outbound.SubscriptionRepository
	creditRepo   outbound.CreditTransactionRepository
}

func NewPipelineCoachService(
	pipelineRepo outbound.PipelineRepository,
	aiService outbound.AIService,
	subRepo outbound.SubscriptionRepository,
	creditRepo outbound.CreditTransactionRepository,
) inbound.PipelineCoachService {
	return &pipelineCoachServiceImpl{
		pipelineRepo: pipelineRepo,
		aiService:    aiService,
		subRepo:      subRepo,
		creditRepo:   creditRepo,
	}
}

func (s *pipelineCoachServiceImpl) Coach(ctx context.Context, req inbound.CoachJobRequest) (*inbound.CoachJobResponse, error) {
	// Wishlist has no AI action
	if strings.EqualFold(req.Stage, string(domain.StageWishlist)) {
		return nil, domain.ErrForbidden
	}

	// Fetch the pipeline job to merge persisted data
	job, err := s.pipelineRepo.Get(ctx, req.UserID, req.JobID)
	if err != nil {
		log.Printf("[coach] pipelineRepo.Get failed: jobID=%s userID=%s err=%v", req.JobID, req.UserID, err)
		return nil, err
	}

	// Merge: request body fields take precedence; fall back to persisted data
	mergeString := func(reqVal, jobVal string) string {
		if reqVal != "" {
			return reqVal
		}
		return jobVal
	}
	mergeSlice := func(reqVal, jobVal []string) []string {
		if len(reqVal) > 0 {
			return reqVal
		}
		return jobVal
	}
	mergeInt := func(reqVal, jobVal int) int {
		if reqVal != 0 {
			return reqVal
		}
		return jobVal
	}

	jobTitle := mergeString(req.JobTitle, job.JobTitle)
	companyName := mergeString(req.CompanyName, job.CompanyName)
	location := mergeString(req.Location, job.Location)
	jobDescription := mergeString(req.JobDescription, job.JobDescription)
	jobURL := mergeString(req.JobURL, job.JobURL)
	matchedKeywords := mergeSlice(req.MatchedKeywords, job.MatchedKeywords)
	missingKeywords := mergeSlice(req.MissingKeywords, job.MissingKeywords)
	atsScore := mergeInt(req.AtsScore, job.AtsScore)

	// Validate merged fields against prompt injection before sending to AI
	if jobDescription != "" {
		if err := security.ValidateJobDescription(security.SanitizeForPrompt(jobDescription)); err != nil {
			return nil, err
		}
		jobDescription = security.SanitizeForPrompt(jobDescription)
	}
	for _, field := range []struct{ val, name string }{
		{jobTitle, "job title"},
		{companyName, "company name"},
		{location, "location"},
		{jobURL, "job URL"},
	} {
		if field.val != "" {
			if err := security.ValidateShortField(field.val, field.name); err != nil {
				return nil, err
			}
		}
	}

	// Credit check
	sub, err := s.subRepo.GetByUserID(ctx, req.UserID)
	if err != nil {
		log.Printf("[coach] subRepo.GetByUserID failed: userID=%s err=%v", req.UserID, err)
		return nil, err
	}
	if !sub.IsActive() {
		log.Printf("[coach] subscription inactive: userID=%s plan=%s credits=%d status=%s", req.UserID, sub.Plan, sub.Credits, sub.Status)
		return nil, domain.ErrSubscriptionInactive
	}
	if sub.Plan == domain.PlanFree {
		if sub.Credits <= 0 {
			log.Printf("[coach] insufficient credits: userID=%s credits=%d", req.UserID, sub.Credits)
			return nil, domain.ErrInsufficientCredits
		}
	}

	// Call AI
	result, err := s.aiService.GenerateCoachContent(ctx, &outbound.CoachJobInput{
		Stage:            req.Stage,
		JobTitle:         jobTitle,
		CompanyName:      companyName,
		Location:         location,
		AtsScore:         atsScore,
		JobDescription:   jobDescription,
		JobURL:           jobURL,
		MatchedKeywords:  matchedKeywords,
		MissingKeywords:  missingKeywords,
		DaysSinceApplied: req.DaysSinceApplied,
		Tone:             req.Tone,
	})
	if err != nil {
		log.Printf("[coach] GenerateCoachContent failed: stage=%s userID=%s err=%v", req.Stage, req.UserID, err)
		return nil, err
	}

	// Deduct credit for free-tier users
	if sub.Plan == domain.PlanFree {
		if err := sub.UseCredit(); err != nil {
			return nil, err
		}
		tx := domain.NewCreditTransaction(req.UserID, 1, domain.CreditTransactionTypeUse, "Pipeline coach")
		tx.Metadata["pipeline_job_id"] = req.JobID
		_ = s.creditRepo.Create(ctx, tx)
		if err := s.subRepo.Update(ctx, sub); err != nil {
			return nil, err
		}
	}

	return &inbound.CoachJobResponse{
		Content: result.Content,
		Stage:   req.Stage,
		Type:    result.Type,
	}, nil
}
