package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/reangeline/backend_applywise/internal/core/ports/inbound"
)

type ResumeHandler struct {
	resumeService inbound.ResumeOptimizerService
}

func NewResumeHandler(resumeService inbound.ResumeOptimizerService) *ResumeHandler {
	return &ResumeHandler{resumeService: resumeService}
}

type UploadResumeRequestDTO struct {
	Content string `json:"content" validate:"required"`
}

type OptimizeResumeRequestDTO struct {
	ResumeID       string `json:"resume_id" validate:"required"`
	JobDescription string `json:"job_description" validate:"required"`
}

func (h *ResumeHandler) UploadResume(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(string)

	var req UploadResumeRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resume, err := h.resumeService.UploadResume(r.Context(), inbound.UploadResumeRequest{
		UserID:  userID,
		Content: req.Content,
	})
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, resume)
}

func (h *ResumeHandler) ListResumes(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(string)

	resumes, err := h.resumeService.ListResumes(r.Context(), userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, resumes)
}

func (h *ResumeHandler) GetResume(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(string)
	resumeID := chi.URLParam(r, "resumeID")

	resume, err := h.resumeService.GetResume(r.Context(), userID, resumeID)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, resume)
}

func (h *ResumeHandler) OptimizeResume(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(string)

	var req OptimizeResumeRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	optimized, err := h.resumeService.OptimizeResume(r.Context(), inbound.OptimizeResumeRequest{
		UserID:         userID,
		ResumeID:       req.ResumeID,
		JobDescription: req.JobDescription,
	})
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, optimized)
}

func (h *ResumeHandler) ListOptimizedResumes(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(string)

	optimized, err := h.resumeService.ListOptimizedResumes(r.Context(), userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, optimized)
}
