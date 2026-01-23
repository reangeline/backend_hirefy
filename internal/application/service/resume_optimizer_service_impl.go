package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/reangeline/backend_applywise/internal/core/domain"
	"github.com/reangeline/backend_applywise/internal/core/ports/inbound"
	"github.com/reangeline/backend_applywise/internal/core/ports/outbound"
)

type resumeOptimizerServiceImpl struct {
	resumeRepo       outbound.ResumeRepository
	aiService        outbound.AIService
	subscriptionRepo outbound.SubscriptionRepository
}

func NewResumeOptimizerService(
	resumeRepo outbound.ResumeRepository,
	aiService outbound.AIService,
	subscriptionRepo outbound.SubscriptionRepository,
) inbound.ResumeOptimizerService {
	return &resumeOptimizerServiceImpl{
		resumeRepo:       resumeRepo,
		aiService:        aiService,
		subscriptionRepo: subscriptionRepo,
	}
}

func (s *resumeOptimizerServiceImpl) UploadResume(
	ctx context.Context,
	req inbound.UploadResumeRequest,
) (*domain.Resume, error) {
	// Parse do currículo usando IA
	analysis, err := s.aiService.ParseResume(ctx, req.Content)
	if err != nil {
		return nil, domain.ErrInvalidResume
	}

	resume := &domain.Resume{
		ID:              uuid.New().String(),
		UserID:          req.UserID,
		OriginalContent: req.Content,
		OriginalS3Key:   req.FileKey,
		ParsedData: map[string]interface{}{
			"skills":     analysis.Skills,
			"experience": analysis.Experience,
			"education":  analysis.Education,
			"keywords":   analysis.Keywords,
		},
	}

	if err := s.resumeRepo.CreateResume(ctx, resume); err != nil {
		return nil, err
	}

	return resume, nil
}

func (s *resumeOptimizerServiceImpl) OptimizeResume(
	ctx context.Context,
	req inbound.OptimizeResumeRequest,
) (*domain.OptimizedResume, error) {
	// Busca subscription mas permite free tier
	subscription, err := s.subscriptionRepo.GetByUserID(ctx, req.UserID)
	if err == nil && subscription != nil && !subscription.IsActive() {
		return nil, domain.ErrSubscriptionInactive
	}

	// Busca todos os currículos do usuário
	resumes, err := s.resumeRepo.ListResumesByUserID(ctx, req.UserID)
	if err != nil {
		return nil, err
	}

	// Encontra o currículo específico
	var resume *domain.Resume
	for _, r := range resumes {
		if r.ID == req.ResumeID {
			resume = r
			break
		}
	}

	if resume == nil {
		return nil, domain.ErrResumeNotFound
	}

	// Parse da job description
	jobAnalysis, err := s.aiService.ParseJobDescription(ctx, req.JobDescription)
	if err != nil {
		return nil, err
	}

	// Reconstrói análise do currículo
	resumeAnalysis := &outbound.ResumeAnalysis{
		Skills:     convertToStringSlice(resume.ParsedData["skills"]),
		Experience: convertToStringSlice(resume.ParsedData["experience"]),
		Education:  convertToStringSlice(resume.ParsedData["education"]),
		Keywords:   convertToStringSlice(resume.ParsedData["keywords"]),
	}

	// Otimiza usando IA
	result, err := s.aiService.OptimizeResume(ctx, resumeAnalysis, jobAnalysis, resume.OriginalContent)
	if err != nil {
		return nil, err
	}

	// Salva resultado
	optimized := &domain.OptimizedResume{
		ID:               uuid.New().String(),
		UserID:           req.UserID,
		ResumeID:         req.ResumeID,
		OptimizedContent: result.OptimizedContent,
		MatchScore:       result.MatchScore,
		Suggestions:      result.Suggestions,
		CreatedAt:        time.Now(),
	}

	if err := s.resumeRepo.CreateOptimizedResume(ctx, optimized); err != nil {
		return nil, err
	}

	return optimized, nil
}

func (s *resumeOptimizerServiceImpl) GetResume(ctx context.Context, userID, resumeID string) (*domain.Resume, error) {
	resume, err := s.resumeRepo.GetResume(ctx, resumeID)
	if err != nil {
		return nil, err
	}

	// Verifica ownership
	if resume.UserID != userID {
		return nil, domain.ErrForbidden
	}

	return resume, nil
}

func (s *resumeOptimizerServiceImpl) ListResumes(ctx context.Context, userID string) ([]*domain.Resume, error) {
	return s.resumeRepo.ListResumesByUserID(ctx, userID)
}

func (s *resumeOptimizerServiceImpl) GetOptimizedResume(ctx context.Context, userID, optimizedResumeID string) (*domain.OptimizedResume, error) {
	optimized, err := s.resumeRepo.GetOptimizedResume(ctx, optimizedResumeID)
	if err != nil {
		return nil, err
	}

	if optimized.UserID != userID {
		return nil, domain.ErrForbidden
	}

	return optimized, nil
}

func (s *resumeOptimizerServiceImpl) ListOptimizedResumes(ctx context.Context, userID string) ([]*domain.OptimizedResume, error) {
	return s.resumeRepo.ListOptimizedResumesByUserID(ctx, userID)
}

func convertToStringSlice(data interface{}) []string {
	if data == nil {
		return []string{}
	}

	// Se já é []string, retorna direto
	if strSlice, ok := data.([]string); ok {
		return strSlice
	}

	// Se é []interface{}, converte cada item
	if interfaceSlice, ok := data.([]interface{}); ok {
		result := make([]string, len(interfaceSlice))
		for i, v := range interfaceSlice {
			if str, ok := v.(string); ok {
				result[i] = str
			}
		}
		return result
	}

	return []string{}
}
