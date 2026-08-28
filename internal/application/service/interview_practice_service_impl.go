package service

import (
	"context"
	"strings"
	"time"

	"github.com/reangeline/backend_applywise/internal/core/domain"
	"github.com/reangeline/backend_applywise/internal/core/ports/inbound"
	"github.com/reangeline/backend_applywise/internal/core/ports/outbound"
	"github.com/reangeline/backend_applywise/pkg/security"
)

type interviewPracticeServiceImpl struct {
	pipelineRepo  outbound.PipelineRepository
	interviewRepo outbound.InterviewRepository
	resumeRepo    outbound.ResumeRepository
	aiService     outbound.AIService
	subRepo       outbound.SubscriptionRepository
	creditRepo    outbound.CreditTransactionRepository
}

func NewInterviewPracticeService(
	pipelineRepo outbound.PipelineRepository,
	interviewRepo outbound.InterviewRepository,
	resumeRepo outbound.ResumeRepository,
	aiService outbound.AIService,
	subRepo outbound.SubscriptionRepository,
	creditRepo outbound.CreditTransactionRepository,
) inbound.InterviewPracticeService {
	return &interviewPracticeServiceImpl{
		pipelineRepo:  pipelineRepo,
		interviewRepo: interviewRepo,
		resumeRepo:    resumeRepo,
		aiService:     aiService,
		subRepo:       subRepo,
		creditRepo:    creditRepo,
	}
}

// resumeDataFor busca os dados estruturados do currículo ligado à vaga, se houver. Best
// effort: uma vaga adicionada rápido (sem otimização) pode não ter currículo vinculado —
// nesse caso a pergunta/avaliação seguem sem esses dados, só menos personalizadas.
func (s *interviewPracticeServiceImpl) resumeDataFor(ctx context.Context, userID string, job *domain.PipelineJob) map[string]interface{} {
	resumeID := job.OptimizedResumeID
	if resumeID == "" {
		resumeID = job.ResumeID
	}
	if resumeID == "" {
		return nil
	}
	resume, err := s.resumeRepo.GetResume(ctx, userID, resumeID)
	if err != nil || resume == nil {
		return nil
	}
	return resume.ParsedData
}

func (s *interviewPracticeServiceImpl) NextQuestion(ctx context.Context, req inbound.NextQuestionRequest) (*inbound.InterviewQuestionDTO, error) {
	job, err := s.pipelineRepo.Get(ctx, req.UserID, req.JobID)
	if err != nil {
		return nil, err
	}
	if strings.EqualFold(string(job.Stage), string(domain.StageWishlist)) {
		return nil, domain.ErrForbidden
	}

	kind := strings.ToLower(strings.TrimSpace(req.Kind))
	if kind == "" {
		kind = "behavioral"
	}

	jobDescription := job.JobDescription
	if jobDescription != "" {
		if err := security.ValidateJobDescription(security.SanitizeForPrompt(jobDescription)); err != nil {
			return nil, err
		}
		jobDescription = security.SanitizeForPrompt(jobDescription)
	}

	history, err := s.interviewRepo.List(ctx, req.UserID, req.JobID)
	if err != nil {
		return nil, err
	}
	previousQuestions := make([]string, 0, len(history))
	pastGaps := make([]string, 0)
	for _, h := range history {
		previousQuestions = append(previousQuestions, h.Question)
		pastGaps = append(pastGaps, h.Gaps...)
	}

	result, err := s.aiService.GenerateInterviewQuestion(ctx, &outbound.InterviewQuestionInput{
		Kind:              kind,
		JobTitle:          job.JobTitle,
		CompanyName:       job.CompanyName,
		JobDescription:    jobDescription,
		MatchedKeywords:   job.MatchedKeywords,
		MissingKeywords:   job.MissingKeywords,
		ResumeData:        s.resumeDataFor(ctx, req.UserID, job),
		PreviousQuestions: previousQuestions,
		PastGaps:          pastGaps,
	})
	if err != nil {
		return nil, err
	}

	question := &domain.InterviewQuestion{
		JobID:        req.JobID,
		UserID:       req.UserID,
		Kind:         domain.InterviewQuestionKind(kind),
		Question:     result.Question,
		WhatTheyWant: result.WhatTheyWant,
		MethodHint:   result.MethodHint,
		CreatedAt:    time.Now(),
	}
	if err := s.interviewRepo.Create(ctx, question); err != nil {
		return nil, err
	}

	return toInterviewQuestionDTO(question), nil
}

func (s *interviewPracticeServiceImpl) SubmitAnswer(ctx context.Context, req inbound.SubmitAnswerRequest) (*inbound.InterviewQuestionDTO, error) {
	question, err := s.interviewRepo.Get(ctx, req.UserID, req.JobID, req.QuestionID)
	if err != nil {
		return nil, err
	}

	answer := req.Answer
	if err := security.ValidateResumeContent(security.SanitizeForPrompt(answer)); err != nil {
		return nil, err
	}
	answer = security.SanitizeForPrompt(answer)

	job, err := s.pipelineRepo.Get(ctx, req.UserID, req.JobID)
	if err != nil {
		return nil, err
	}

	jobDescription := job.JobDescription
	if jobDescription != "" {
		jobDescription = security.SanitizeForPrompt(jobDescription)
	}

	// Checagem de crédito — avaliar resposta é a entrega de valor real (nota, gaps,
	// resposta-modelo), gerar pergunta é grátis. Mesmo padrão de PipelineCoachService.
	sub, err := s.subRepo.GetByUserID(ctx, req.UserID)
	if err != nil {
		return nil, err
	}
	if !sub.IsActive() {
		return nil, domain.ErrSubscriptionInactive
	}
	if sub.Plan == domain.PlanFree {
		if sub.Credits <= 0 {
			return nil, domain.ErrInsufficientCredits
		}
	}

	eval, err := s.aiService.EvaluateInterviewAnswer(ctx, &outbound.InterviewAnswerInput{
		Kind:            string(question.Kind),
		Question:        question.Question,
		JobTitle:        job.JobTitle,
		CompanyName:     job.CompanyName,
		JobDescription:  jobDescription,
		ResumeData:      s.resumeDataFor(ctx, req.UserID, job),
		CandidateAnswer: answer,
	})
	if err != nil {
		return nil, err
	}

	now := time.Now()
	question.Answer = answer
	question.ContentScore = eval.ContentScore
	question.Strengths = eval.Strengths
	question.Gaps = eval.Gaps
	question.ModelAnswer = eval.ModelAnswer
	question.FollowUp = eval.FollowUp
	question.AnsweredAt = &now
	if eval.Star != nil {
		question.STARSituation = eval.Star.Situation
		question.STARTask = eval.Star.Task
		question.STARAction = eval.Star.Action
		question.STARResult = eval.Star.Result
	}

	if err := s.interviewRepo.Update(ctx, question); err != nil {
		return nil, err
	}

	if sub.Plan == domain.PlanFree {
		if err := sub.UseCredit(); err != nil {
			return nil, err
		}
		tx := domain.NewCreditTransaction(req.UserID, 1, domain.CreditTransactionTypeUse, "Interview practice")
		tx.Metadata["pipeline_job_id"] = req.JobID
		tx.Metadata["interview_question_id"] = req.QuestionID
		_ = s.creditRepo.Create(ctx, tx)
		if err := s.subRepo.Update(ctx, sub); err != nil {
			return nil, err
		}
	}

	return toInterviewQuestionDTO(question), nil
}

func (s *interviewPracticeServiceImpl) ListHistory(ctx context.Context, userID, jobID string) ([]inbound.InterviewQuestionDTO, error) {
	history, err := s.interviewRepo.List(ctx, userID, jobID)
	if err != nil {
		return nil, err
	}
	dtos := make([]inbound.InterviewQuestionDTO, 0, len(history))
	for i := range history {
		dtos = append(dtos, *toInterviewQuestionDTO(&history[i]))
	}
	return dtos, nil
}

func toInterviewQuestionDTO(q *domain.InterviewQuestion) *inbound.InterviewQuestionDTO {
	return &inbound.InterviewQuestionDTO{
		ID:            q.ID,
		Kind:          string(q.Kind),
		Question:      q.Question,
		WhatTheyWant:  q.WhatTheyWant,
		MethodHint:    q.MethodHint,
		Answer:        q.Answer,
		ContentScore:  q.ContentScore,
		StarSituation: q.STARSituation,
		StarTask:      q.STARTask,
		StarAction:    q.STARAction,
		StarResult:    q.STARResult,
		Strengths:     q.Strengths,
		Gaps:          q.Gaps,
		ModelAnswer:   q.ModelAnswer,
		FollowUp:      q.FollowUp,
		CreatedAt:     q.CreatedAt.Format(time.RFC3339),
		Answered:      q.IsAnswered(),
	}
}
