package domain

import "time"

// InterviewQuestionKind classifica o tipo de pergunta de entrevista.
type InterviewQuestionKind string

const (
	InterviewKindBehavioral  InterviewQuestionKind = "behavioral"
	InterviewKindTechnical   InterviewQuestionKind = "technical"
	InterviewKindSituational InterviewQuestionKind = "situational"
	InterviewKindScreening   InterviewQuestionKind = "screening"
)

// InterviewQuestion representa uma pergunta de prática de entrevista gerada pra uma vaga do
// pipeline, com a resposta do usuário e a avaliação da IA depois de respondida. Ver
// .spec/010-interview-practice/spec.md.
type InterviewQuestion struct {
	ID           string
	JobID        string
	UserID       string
	Kind         InterviewQuestionKind
	Question     string
	WhatTheyWant string
	MethodHint   string

	// Preenchidos só depois de responder (SubmitAnswer)
	Answer        string
	ContentScore  int
	STARSituation int
	STARTask      int
	STARAction    int
	STARResult    int
	Strengths     []string
	Gaps          []string
	ModelAnswer   string
	FollowUp      string

	CreatedAt  time.Time
	AnsweredAt *time.Time
}

// IsAnswered indica se a pergunta já foi respondida e avaliada.
func (q *InterviewQuestion) IsAnswered() bool {
	return q.AnsweredAt != nil
}
