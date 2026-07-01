package api

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"

	"github.com/infraforge/infraforge/internal/models"
	"github.com/infraforge/infraforge/internal/repository"
)

const maxActiveWorkflowsPerTeam = 10

// WorkflowHandler handles workflow API endpoints.
type WorkflowHandler struct {
	workflows *repository.WorkflowRepository
	audit     *repository.AuditRepository
}

// NewWorkflowHandler creates a new WorkflowHandler.
func NewWorkflowHandler(workflows *repository.WorkflowRepository, audit *repository.AuditRepository) *WorkflowHandler {
	return &WorkflowHandler{workflows: workflows, audit: audit}
}

// Register mounts workflow routes on the given mux.
func (h *WorkflowHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/workflows", h.List)
	mux.HandleFunc("POST /api/v1/workflows", h.Create)
	mux.HandleFunc("GET /api/v1/workflows/{id}", h.Get)
	mux.HandleFunc("POST /api/v1/workflows/{id}/retry", h.Retry)
	mux.HandleFunc("POST /api/v1/workflows/{id}/cancel", h.Cancel)
}

func (h *WorkflowHandler) List(w http.ResponseWriter, r *http.Request) {
	teamID, err := GetTeamID(r)
	if err != nil {
		Error(w, http.StatusBadRequest, "X-Team-ID header required")
		return
	}

	status := r.URL.Query().Get("status")
	workflows, err := h.workflows.ListByTeam(r.Context(), teamID, status)
	if err != nil {
		Error(w, http.StatusInternalServerError, "failed to list workflows")
		return
	}
	JSON(w, http.StatusOK, workflows)
}

func (h *WorkflowHandler) Create(w http.ResponseWriter, r *http.Request) {
	teamID, err := GetTeamID(r)
	if err != nil {
		Error(w, http.StatusBadRequest, "X-Team-ID header required")
		return
	}

	// Quota enforcement
	count, err := h.workflows.CountByTeam(r.Context(), teamID)
	if err != nil {
		Error(w, http.StatusInternalServerError, "failed to check quota")
		return
	}
	if count >= maxActiveWorkflowsPerTeam {
		Error(w, http.StatusForbidden, "active workflow quota exceeded (max 10 per team)")
		return
	}

	var req struct {
		Name          string     `json:"name"`
		Description   *string    `json:"description"`
		WorkflowType  string     `json:"workflow_type"`
		EnvironmentID *uuid.UUID `json:"environment_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" || req.WorkflowType == "" {
		Error(w, http.StatusBadRequest, "name and workflow_type are required")
		return
	}

	validTypes := map[string]bool{"deploy": true, "provision": true, "destroy": true, "drift_check": true}
	if !validTypes[req.WorkflowType] {
		Error(w, http.StatusBadRequest, "workflow_type must be one of: deploy, provision, destroy, drift_check")
		return
	}

	actor := GetActor(r)
	wf := &models.Workflow{
		TeamID:        teamID,
		EnvironmentID: req.EnvironmentID,
		Name:          req.Name,
		Description:   req.Description,
		WorkflowType:  req.WorkflowType,
		Status:        "pending",
		InitiatedBy:   actor,
	}

	if err := h.workflows.Create(r.Context(), wf); err != nil {
		Error(w, http.StatusInternalServerError, "failed to create workflow")
		return
	}

	h.audit.LogAction(r.Context(), &teamID, actor, "workflow.created", "workflow", &wf.ID,
		map[string]string{"name": wf.Name, "type": wf.WorkflowType})

	JSON(w, http.StatusCreated, wf)
}

// WorkflowDetail includes the workflow and its steps.
type WorkflowDetail struct {
	*models.Workflow
	Steps []models.WorkflowStep `json:"steps"`
}

func (h *WorkflowHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid workflow ID")
		return
	}

	wf, err := h.workflows.GetByID(r.Context(), id)
	if err != nil {
		Error(w, http.StatusNotFound, "workflow not found")
		return
	}

	steps, err := h.workflows.GetSteps(r.Context(), id)
	if err != nil {
		Error(w, http.StatusInternalServerError, "failed to get workflow steps")
		return
	}

	detail := &WorkflowDetail{Workflow: wf, Steps: steps}
	JSON(w, http.StatusOK, detail)
}

func (h *WorkflowHandler) Retry(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid workflow ID")
		return
	}

	wf, err := h.workflows.GetByID(r.Context(), id)
	if err != nil {
		Error(w, http.StatusNotFound, "workflow not found")
		return
	}

	if wf.Status != "failed" && wf.Status != "cancelled" {
		Error(w, http.StatusBadRequest, "only failed or cancelled workflows can be retried")
		return
	}

	if err := h.workflows.UpdateStatus(r.Context(), id, "pending", nil); err != nil {
		Error(w, http.StatusInternalServerError, "failed to retry workflow")
		return
	}

	actor := GetActor(r)
	h.audit.LogAction(r.Context(), &wf.TeamID, actor, "workflow.retried", "workflow", &id,
		map[string]string{"name": wf.Name})

	wf.Status = "pending"
	JSON(w, http.StatusOK, wf)
}

func (h *WorkflowHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid workflow ID")
		return
	}

	wf, err := h.workflows.GetByID(r.Context(), id)
	if err != nil {
		Error(w, http.StatusNotFound, "workflow not found")
		return
	}

	if wf.Status != "pending" && wf.Status != "running" {
		Error(w, http.StatusBadRequest, "only pending or running workflows can be cancelled")
		return
	}

	if err := h.workflows.UpdateStatus(r.Context(), id, "cancelled", nil); err != nil {
		Error(w, http.StatusInternalServerError, "failed to cancel workflow")
		return
	}

	actor := GetActor(r)
	h.audit.LogAction(r.Context(), &wf.TeamID, actor, "workflow.cancelled", "workflow", &id,
		map[string]string{"name": wf.Name})

	wf.Status = "cancelled"
	JSON(w, http.StatusOK, wf)
}
