package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/reangeline/backend_applywise/internal/core/domain"
	"github.com/reangeline/backend_applywise/internal/core/ports/inbound"
	"github.com/reangeline/backend_applywise/internal/core/ports/outbound"
	"github.com/reangeline/backend_applywise/pkg/security"
)

type linkedInPostServiceImpl struct {
	resumeRepo    outbound.ResumeRepository
	postIdeasRepo outbound.LinkedInPostIdeasRepository
	aiService     outbound.AIService
}

func NewLinkedInPostService(
	resumeRepo outbound.ResumeRepository,
	postIdeasRepo outbound.LinkedInPostIdeasRepository,
	aiService outbound.AIService,
) inbound.LinkedInPostService {
	return &linkedInPostServiceImpl{
		resumeRepo:    resumeRepo,
		postIdeasRepo: postIdeasRepo,
		aiService:     aiService,
	}
}

func (s *linkedInPostServiceImpl) resumeAnalysisFor(ctx context.Context, userID, resumeID string) (*outbound.ResumeAnalysis, error) {
	resume, err := s.resumeRepo.GetResume(ctx, userID, resumeID)
	if err != nil {
		return nil, err
	}

	return &outbound.ResumeAnalysis{
		Skills:         extractSkillsFromParsedData(resume.ParsedData),
		Experience:     extractExperiencesFromParsedData(resume.ParsedData),
		Education:      extractEducationFromParsedData(resume.ParsedData),
		Keywords:       convertToStringSlice(resume.ParsedData["keywords"]),
		StructuredData: resume.ParsedData,
	}, nil
}

func (s *linkedInPostServiceImpl) GenerateTopics(ctx context.Context, req inbound.GenerateTopicsRequest) (*domain.LinkedInPostIdeas, error) {
	resumeAnalysis, err := s.resumeAnalysisFor(ctx, req.UserID, req.ResumeID)
	if err != nil {
		return nil, err
	}

	targetRole := security.SanitizeForPrompt(req.TargetRole)
	if targetRole != "" {
		if err := security.ValidateShortField(targetRole, "target role"); err != nil {
			return nil, err
		}
	}

	result, err := s.aiService.GenerateLinkedInPostTopics(ctx, &outbound.PostTopicsInput{
		Resume:     resumeAnalysis,
		TargetRole: targetRole,
	})
	if err != nil {
		return nil, fmt.Errorf("AI post topics generation failed: %w", err)
	}

	now := time.Now().UTC()
	ideas := &domain.LinkedInPostIdeas{
		ID:        uuid.New().String(),
		UserID:    req.UserID,
		Topics:    result.Topics,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.postIdeasRepo.Upsert(ctx, ideas); err != nil {
		return nil, fmt.Errorf("failed to save post topics: %w", err)
	}

	return ideas, nil
}

func (s *linkedInPostServiceImpl) GetLatestTopics(ctx context.Context, userID string) (*domain.LinkedInPostIdeas, error) {
	return s.postIdeasRepo.GetByUserID(ctx, userID)
}

func (s *linkedInPostServiceImpl) DraftPost(ctx context.Context, req inbound.DraftPostRequest) (*inbound.DraftPostResult, error) {
	resumeAnalysis, err := s.resumeAnalysisFor(ctx, req.UserID, req.ResumeID)
	if err != nil {
		return nil, err
	}

	title := security.SanitizeForPrompt(req.TopicTitle)
	if err := security.ValidateShortField(title, "topic title"); err != nil {
		return nil, err
	}

	angle := security.SanitizeForPrompt(req.TopicAngle)
	if err := security.ValidateJobDescription(angle); err != nil {
		return nil, err
	}

	result, err := s.aiService.DraftLinkedInPost(ctx, &outbound.PostDraftInput{
		Resume:     resumeAnalysis,
		TopicTitle: title,
		TopicAngle: angle,
	})
	if err != nil {
		return nil, fmt.Errorf("AI post draft failed: %w", err)
	}

	// Salva o rascunho no tema correspondente — próxima visita/troca de tema lê daqui em
	// vez de gerar de novo. Falha silenciosa (loga, não bloqueia a resposta): o usuário já
	// tem o texto gerado na tela mesmo se a persistência falhar.
	if ideas, err := s.postIdeasRepo.GetByUserID(ctx, req.UserID); err == nil &&
		req.TopicIndex >= 0 && req.TopicIndex < len(ideas.Topics) &&
		ideas.Topics[req.TopicIndex].Title == req.TopicTitle {
		ideas.Topics[req.TopicIndex].Draft = result.PostText
		ideas.UpdatedAt = time.Now().UTC()
		if err := s.postIdeasRepo.Upsert(ctx, ideas); err != nil {
			log.Printf("[linkedin-post] failed to persist draft: userID=%s topicIndex=%d err=%v", req.UserID, req.TopicIndex, err)
		}
	}

	return &inbound.DraftPostResult{PostText: result.PostText}, nil
}
