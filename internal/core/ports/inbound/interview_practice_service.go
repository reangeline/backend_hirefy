package inbound

import "context"

// NextQuestionRequest pede a próxima pergunta de prática pra uma vaga do pipeline.
type NextQuestionRequest struct {
	UserID string
	JobID  string
	Kind   string // "behavioral" | "technical" | "situational" | "screening"
}

// InterviewQuestionDTO é uma pergunta de prática (com ou sem resposta ainda).
type InterviewQuestionDTO struct {
	ID            string   `json:"id"`
	Kind          string   `json:"kind"`
	Question      string   `json:"question"`
	WhatTheyWant  string   `json:"what_they_want,omitempty"`
	MethodHint    string   `json:"method_hint,omitempty"`
	Answer        string   `json:"answer,omitempty"`
	ContentScore  int      `json:"content_score,omitempty"`
	StarSituation int      `json:"star_situation,omitempty"`
	StarTask      int      `json:"star_task,omitempty"`
	StarAction    int      `json:"star_action,omitempty"`
	StarResult    int      `json:"star_result,omitempty"`
	Strengths     []string `json:"strengths,omitempty"`
	Gaps          []string `json:"gaps,omitempty"`
	ModelAnswer   string   `json:"model_answer,omitempty"`
	FollowUp      string   `json:"follow_up,omitempty"`
	CreatedAt     string   `json:"created_at"`
	Answered      bool     `json:"answered"`
}

// SubmitAnswerRequest envia a resposta do candidato pra uma pergunta gerada.
type SubmitAnswerRequest struct {
	UserID     string
	JobID      string
	QuestionID string
	Answer     string
}

// InterviewPracticeService conduz a prática de entrevista (perguntas + avaliação de
// resposta) por vaga do pipeline. Gerar pergunta é grátis; avaliar resposta consome 1
// crédito no plano Free (mesmo padrão de PipelineCoachService).
type InterviewPracticeService interface {
	NextQuestion(ctx context.Context, req NextQuestionRequest) (*InterviewQuestionDTO, error)
	SubmitAnswer(ctx context.Context, req SubmitAnswerRequest) (*InterviewQuestionDTO, error)
	ListHistory(ctx context.Context, userID, jobID string) ([]InterviewQuestionDTO, error)
}
