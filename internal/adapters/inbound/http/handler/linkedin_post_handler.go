package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/reangeline/backend_applywise/internal/adapters/inbound/http/middleware"
	"github.com/reangeline/backend_applywise/internal/core/domain"
	"github.com/reangeline/backend_applywise/internal/core/ports/inbound"
	"github.com/reangeline/backend_applywise/pkg/security"
)

type LinkedInPostHandler struct {
	linkedInPostService inbound.LinkedInPostService
}

func NewLinkedInPostHandler(linkedInPostService inbound.LinkedInPostService) *LinkedInPostHandler {
	return &LinkedInPostHandler{linkedInPostService: linkedInPostService}
}

type generateTopicsRequestDTO struct {
	ResumeID   string `json:"resume_id" validate:"required"`
	TargetRole string `json:"target_role,omitempty"`
}

// GenerateTopics gera uma lista nova de temas de post e substitui a anterior — spec 018.
func (h *LinkedInPostHandler) GenerateTopics(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(middleware.UserIDContextKey).(string)

	var req generateTopicsRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.ResumeID == "" {
		respondError(w, http.StatusBadRequest, "resume_id is required")
		return
	}

	ideas, err := h.linkedInPostService.GenerateTopics(r.Context(), inbound.GenerateTopicsRequest{
		UserID:     userID,
		ResumeID:   req.ResumeID,
		TargetRole: req.TargetRole,
	})
	if err != nil {
		respondLinkedInPostError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, ideas)
}

// GetLatestTopics retorna o último conjunto de temas salvo, ou 404 se nunca gerou.
func (h *LinkedInPostHandler) GetLatestTopics(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(middleware.UserIDContextKey).(string)

	ideas, err := h.linkedInPostService.GetLatestTopics(r.Context(), userID)
	if err != nil {
		respondLinkedInPostError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, ideas)
}

type draftPostRequestDTO struct {
	ResumeID string `json:"resume_id" validate:"required"`
	Title    string `json:"title" validate:"required"`
	Angle    string `json:"angle" validate:"required"`
	Index    int    `json:"index"`
}

// DraftPost rascunha um post pronto pro tema escolhido — não persiste nada.
func (h *LinkedInPostHandler) DraftPost(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(middleware.UserIDContextKey).(string)

	var req draftPostRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.ResumeID == "" || req.Title == "" || req.Angle == "" {
		respondError(w, http.StatusBadRequest, "resume_id, title and angle are required")
		return
	}

	result, err := h.linkedInPostService.DraftPost(r.Context(), inbound.DraftPostRequest{
		UserID:     userID,
		ResumeID:   req.ResumeID,
		TopicTitle: req.Title,
		TopicAngle: req.Angle,
		TopicIndex: req.Index,
	})
	if err != nil {
		respondLinkedInPostError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, result)
}

func respondLinkedInPostError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrLinkedInPostIdeasNotFound):
		respondError(w, http.StatusNotFound, "nenhum tema gerado ainda")
	case errors.Is(err, domain.ErrResumeNotFound):
		respondError(w, http.StatusNotFound, "resume not found")
	case errors.Is(err, security.ErrInjectionDetected), errors.Is(err, security.ErrInputTooLong):
		respondError(w, http.StatusBadRequest, err.Error())
	default:
		respondError(w, http.StatusUnprocessableEntity, err.Error())
	}
}
