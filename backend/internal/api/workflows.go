package api

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"go.temporal.io/sdk/client"

	"github.com/infraforge/infraforge/internal/models"
	"github.com/infraforge/infraforge/internal/repository"
	infraTemporal "github.com/infraforge/infraforge/internal/temporal"
)

const maxActiveWorkflowsPerTeam = 10

// WorkflowHandler handles workflow API endpoints.
type WorkflowHandler struct {
	workflows      *repository.WorkflowRepository
	environments   *repository.EnvironmentRepository
	audit          *repository.AuditRepository
	temporalClient client.Client
}

// NewWorkflowHandler creates a new WorkflowHandler.
func NewWorkflowHandler(
	workflows *repository.WorkflowRepository,
	environments *repository.EnvironmentRepository,
	audit *repository.AuditRepository,
	temporalClient client.Client,
) *WorkflowHandler {
	return &WorkflowHandler{
		workflows:      workflows,
		environments:   environments,
		audit:          audit,
		temporalClient: temporalClient,
	}
}

// Register mounts workflow routes on the given mux.
func (h *WorkflowHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/workflows", h.List)
	mux.HandleFunc("POST /api/v1/workflows", h.Create)
	mux.HandleFunc("GET /api/v1/workflows/{id}", h.Get)
	mux.HandleFunc("POST /api/v1/workflows/{id}/retry", h.Retry)
	mux.HandleFunc("POST /api/v1/workflows/{id}/cancel", h.Cancel)
	mux.HandleFunc("POST /api/v1/workflows/{id}/signal", h.Signal)
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
		Name          string                 `json:"name"`
		Description   *string                `json:"description"`
		WorkflowType  string                 `json:"workflow_type"`
		EnvironmentID *uuid.UUID             `json:"environment_id"`
		Config        map[string]interface{} `json:"config"`
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

	// Start Temporal workflow
	runID, startErr := h.startTemporalWorkflow(r, wf, req.Config)
	if startErr != nil {
		// Workflow created in DB but Temporal start failed — mark as failed
		errMsg := startErr.Error()
		_ = h.workflows.UpdateStatus(r.Context(), wf.ID, "failed", &errMsg)
		wf.Status = "failed"
		wf.ErrorMessage = &errMsg
	} else {
		// Store Temporal IDs
		temporalWfID := h.temporalWorkflowID(wf)
		_ = h.workflows.SetTemporalIDs(r.Context(), wf.ID, temporalWfID, runID)
		wf.TemporalWorkflowID = &temporalWfID
		wf.TemporalRunID = &runID
	}

	h.audit.LogAction(r.Context(), &teamID, actor, "workflow.created", "workflow", &wf.ID,
		map[string]string{"name": wf.Name, "type": wf.WorkflowType})

	JSON(w, http.StatusCreated, wf)
}

func (h *WorkflowHandler) startTemporalWorkflow(r *http.Request, wf *models.Workflow, config map[string]interface{}) (string, error) {
	if h.temporalClient == nil {
		return "", nil // Temporal not connected, skip
	}

	envID := ""
	if wf.EnvironmentID != nil {
		envID = wf.EnvironmentID.String()
	}

	if config == nil {
		config = make(map[string]interface{})
	}

	// Enrich config with environment data if available
	if wf.EnvironmentID != nil {
		env, err := h.environments.GetByID(r.Context(), *wf.EnvironmentID)
		if err == nil {
			config["provider"] = env.Provider
			config["slug"] = env.Slug
			if env.Status == "active" && env.Slug == "prod" || env.Slug == "production" {
				config["is_production"] = true
			}
		}
	}

	params := infraTemporal.WorkflowParams{
		WorkflowID:    wf.ID.String(),
		TeamID:        wf.TeamID.String(),
		EnvironmentID: envID,
		Config:        config,
	}

	switch wf.WorkflowType {
	case "provision":
		return infraTemporal.StartProvisionWorkflow(r.Context(), h.temporalClient, params)
	case "deploy":
		return infraTemporal.StartUpdateWorkflow(r.Context(), h.temporalClient, params)
	case "destroy":
		return infraTemporal.StartDecommissionWorkflow(r.Context(), h.temporalClient, params)
	default:
		return "", nil // drift_check handled separately
	}
}

func (h *WorkflowHandler) temporalWorkflowID(wf *models.Workflow) string {
	switch wf.WorkflowType {
	case "provision":
		return "provision-" + wf.ID.String()
	case "deploy":
		return "update-" + wf.ID.String()
	case "destroy":
		return "decommission-" + wf.ID.String()
	default:
		return "workflow-" + wf.ID.String()
	}
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

	// Reset to pending
	if err := h.workflows.UpdateStatus(r.Context(), id, "pending", nil); err != nil {
		Error(w, http.StatusInternalServerError, "failed to retry workflow")
		return
	}

	// Clear old steps
	_ = h.workflows.DeleteSteps(r.Context(), id)

	// Restart in Temporal
	var config map[string]interface{}
	if wf.EnvironmentID != nil {
		env, _ := h.environments.GetByID(r.Context(), *wf.EnvironmentID)
		if env != nil {
			config = map[string]interface{}{"provider": env.Provider, "slug": env.Slug}
		}
	}
	runID, startErr := h.startTemporalWorkflow(r, wf, config)
	if startErr == nil && runID != "" {
		temporalWfID := h.temporalWorkflowID(wf)
		_ = h.workflows.SetTemporalIDs(r.Context(), id, temporalWfID, runID)
		wf.TemporalWorkflowID = &temporalWfID
		wf.TemporalRunID = &runID
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

	// Terminate in Temporal
	if h.temporalClient != nil && wf.TemporalWorkflowID != nil {
		_ = infraTemporal.TerminateWorkflow(r.Context(), h.temporalClient, *wf.TemporalWorkflowID, "cancelled by user")
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

func (h *WorkflowHandler) Signal(w http.ResponseWriter, r *http.Request) {
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

	if h.temporalClient == nil || wf.TemporalWorkflowID == nil {
		Error(w, http.StatusBadRequest, "workflow has no temporal execution")
		return
	}

	var req struct {
		Approved bool   `json:"approved"`
		Reason   string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	actor := GetActor(r)
	signal := infraTemporal.ApprovalSignal{
		Approved: req.Approved,
		Actor:    actor,
		Reason:   req.Reason,
	}

	if err := infraTemporal.SendApprovalSignal(r.Context(), h.temporalClient, *wf.TemporalWorkflowID, signal); err != nil {
		Error(w, http.StatusInternalServerError, "failed to signal workflow: "+err.Error())
		return
	}

	// If rejected, terminate
	if !req.Approved {
		reason := "rejected by " + actor
		if req.Reason != "" {
			reason += ": " + req.Reason
		}
		_ = infraTemporal.TerminateWorkflow(r.Context(), h.temporalClient, *wf.TemporalWorkflowID, reason)
		errMsg := reason
		_ = h.workflows.UpdateStatus(r.Context(), id, "failed", &errMsg)
	}

	h.audit.LogAction(r.Context(), &wf.TeamID, actor, "workflow.signaled", "workflow", &id,
		map[string]interface{}{"approved": req.Approved, "reason": req.Reason})

	JSON(w, http.StatusOK, map[string]interface{}{"signaled": true, "approved": req.Approved})
}
