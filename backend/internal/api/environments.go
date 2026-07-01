package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"go.temporal.io/sdk/client"

	"github.com/infraforge/infraforge/internal/models"
	"github.com/infraforge/infraforge/internal/repository"
	infraTemporal "github.com/infraforge/infraforge/internal/temporal"
)

const maxEnvironmentsPerTeam = 20

// EnvironmentHandler handles environment API endpoints.
type EnvironmentHandler struct {
	envs           *repository.EnvironmentRepository
	workflows      *repository.WorkflowRepository
	drifts         *repository.DriftRepository
	audit          *repository.AuditRepository
	temporalClient client.Client
}

// NewEnvironmentHandler creates a new EnvironmentHandler.
func NewEnvironmentHandler(
	envs *repository.EnvironmentRepository,
	workflows *repository.WorkflowRepository,
	drifts *repository.DriftRepository,
	audit *repository.AuditRepository,
	temporalClient client.Client,
) *EnvironmentHandler {
	return &EnvironmentHandler{
		envs:           envs,
		workflows:      workflows,
		drifts:         drifts,
		audit:          audit,
		temporalClient: temporalClient,
	}
}

// Register mounts environment routes on the given mux.
func (h *EnvironmentHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/environments", h.List)
	mux.HandleFunc("POST /api/v1/environments", h.Create)
	mux.HandleFunc("GET /api/v1/environments/{id}", h.Get)
	mux.HandleFunc("PUT /api/v1/environments/{id}", h.Update)
	mux.HandleFunc("DELETE /api/v1/environments/{id}", h.Delete)
	mux.HandleFunc("POST /api/v1/environments/{id}/decommission", h.Decommission)
}

func (h *EnvironmentHandler) List(w http.ResponseWriter, r *http.Request) {
	teamID, err := GetTeamID(r)
	if err != nil {
		Error(w, http.StatusBadRequest, "X-Team-ID header required")
		return
	}

	envs, err := h.envs.ListByTeam(r.Context(), teamID)
	if err != nil {
		Error(w, http.StatusInternalServerError, "failed to list environments")
		return
	}
	JSON(w, http.StatusOK, envs)
}

func (h *EnvironmentHandler) Create(w http.ResponseWriter, r *http.Request) {
	teamID, err := GetTeamID(r)
	if err != nil {
		Error(w, http.StatusBadRequest, "X-Team-ID header required")
		return
	}

	// Quota enforcement
	count, err := h.envs.CountByTeam(r.Context(), teamID)
	if err != nil {
		Error(w, http.StatusInternalServerError, "failed to check quota")
		return
	}
	if count >= maxEnvironmentsPerTeam {
		Error(w, http.StatusForbidden, "environment quota exceeded (max 20 per team)")
		return
	}

	var req struct {
		Name     string          `json:"name"`
		Slug     string          `json:"slug"`
		Provider string          `json:"provider"`
		Region   *string         `json:"region"`
		Config   json.RawMessage `json:"config"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" || req.Slug == "" || req.Provider == "" {
		Error(w, http.StatusBadRequest, "name, slug, and provider are required")
		return
	}

	validProviders := map[string]bool{"aws": true, "gcp": true, "azure": true, "k8s": true}
	if !validProviders[req.Provider] {
		Error(w, http.StatusBadRequest, "provider must be one of: aws, gcp, azure, k8s")
		return
	}

	config := req.Config
	if config == nil {
		config = json.RawMessage(`{}`)
	}

	env := &models.Environment{
		TeamID:   teamID,
		Name:     req.Name,
		Slug:     strings.ToLower(req.Slug),
		Provider: req.Provider,
		Region:   req.Region,
		Config:   config,
		Status:   "provisioning",
	}

	if err := h.envs.Create(r.Context(), env); err != nil {
		if strings.Contains(err.Error(), "duplicate") {
			Error(w, http.StatusConflict, "environment with this slug already exists for this team")
			return
		}
		Error(w, http.StatusInternalServerError, "failed to create environment")
		return
	}

	actor := GetActor(r)
	h.audit.LogAction(r.Context(), &teamID, actor, "environment.created", "environment", &env.ID,
		map[string]string{"name": env.Name, "provider": env.Provider})

	// Auto-trigger Provision workflow
	provisionWf := h.triggerProvision(r, env, actor)

	response := map[string]interface{}{
		"environment": env,
		"workflow":    provisionWf,
	}
	JSON(w, http.StatusCreated, response)
}

// triggerProvision creates a workflow record and starts it in Temporal.
func (h *EnvironmentHandler) triggerProvision(r *http.Request, env *models.Environment, actor string) *models.Workflow {
	wf := &models.Workflow{
		TeamID:        env.TeamID,
		EnvironmentID: &env.ID,
		Name:          fmt.Sprintf("Provision: %s", env.Name),
		WorkflowType:  "provision",
		Status:        "pending",
		InitiatedBy:   actor,
	}

	if err := h.workflows.Create(r.Context(), wf); err != nil {
		return nil
	}

	// Build config from environment
	var configMap map[string]interface{}
	_ = json.Unmarshal(env.Config, &configMap)
	if configMap == nil {
		configMap = make(map[string]interface{})
	}
	configMap["provider"] = env.Provider
	configMap["slug"] = env.Slug
	if env.Slug == "prod" || env.Slug == "production" || configMap["tier"] == "production" {
		configMap["is_production"] = true
	}
	configMap["initiated_by"] = actor
	configMap["environment_name"] = env.Name

	// Start in Temporal
	if h.temporalClient != nil {
		params := infraTemporal.WorkflowParams{
			WorkflowID:    wf.ID.String(),
			TeamID:        env.TeamID.String(),
			EnvironmentID: env.ID.String(),
			Config:        configMap,
		}

		runID, err := infraTemporal.StartProvisionWorkflow(r.Context(), h.temporalClient, params)
		if err == nil {
			temporalWfID := "provision-" + wf.ID.String()
			_ = h.workflows.SetTemporalIDs(r.Context(), wf.ID, temporalWfID, runID)
			wf.TemporalWorkflowID = &temporalWfID
			wf.TemporalRunID = &runID
		} else {
			errMsg := err.Error()
			_ = h.workflows.UpdateStatus(r.Context(), wf.ID, "failed", &errMsg)
			wf.Status = "failed"
			wf.ErrorMessage = &errMsg
		}
	}

	h.audit.LogAction(r.Context(), &env.TeamID, actor, "workflow.created", "workflow", &wf.ID,
		map[string]string{"name": wf.Name, "type": "provision", "trigger": "auto"})

	return wf
}

func (h *EnvironmentHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid environment ID")
		return
	}

	env, err := h.envs.GetByID(r.Context(), id)
	if err != nil {
		Error(w, http.StatusNotFound, "environment not found")
		return
	}
	JSON(w, http.StatusOK, env)
}

func (h *EnvironmentHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid environment ID")
		return
	}

	existing, err := h.envs.GetByID(r.Context(), id)
	if err != nil {
		Error(w, http.StatusNotFound, "environment not found")
		return
	}

	var req struct {
		Name     *string         `json:"name"`
		Slug     *string         `json:"slug"`
		Provider *string         `json:"provider"`
		Region   *string         `json:"region"`
		Config   json.RawMessage `json:"config"`
		Status   *string         `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name != nil {
		existing.Name = *req.Name
	}
	if req.Slug != nil {
		existing.Slug = strings.ToLower(*req.Slug)
	}
	if req.Provider != nil {
		existing.Provider = *req.Provider
	}
	if req.Region != nil {
		existing.Region = req.Region
	}
	if req.Config != nil {
		existing.Config = req.Config
	}
	if req.Status != nil {
		existing.Status = *req.Status
	}

	if err := h.envs.Update(r.Context(), existing); err != nil {
		Error(w, http.StatusInternalServerError, "failed to update environment")
		return
	}

	actor := GetActor(r)
	h.audit.LogAction(r.Context(), &existing.TeamID, actor, "environment.updated", "environment", &id,
		map[string]string{"name": existing.Name})

	// Auto-resolve any drift records for this environment (config was updated)
	_ = h.drifts.ResolveAllByEnvironment(r.Context(), id, "manual_fix")

	// Auto-trigger Update workflow
	updateWf := h.triggerUpdate(r, existing, actor)

	response := map[string]interface{}{
		"environment": existing,
		"workflow":    updateWf,
	}
	JSON(w, http.StatusOK, response)
}

// triggerUpdate creates an Update workflow and starts it in Temporal.
func (h *EnvironmentHandler) triggerUpdate(r *http.Request, env *models.Environment, actor string) *models.Workflow {
	wf := &models.Workflow{
		TeamID:        env.TeamID,
		EnvironmentID: &env.ID,
		Name:          fmt.Sprintf("Update: %s", env.Name),
		WorkflowType:  "deploy",
		Status:        "pending",
		InitiatedBy:   actor,
	}

	if err := h.workflows.Create(r.Context(), wf); err != nil {
		return nil
	}

	var configMap map[string]interface{}
	_ = json.Unmarshal(env.Config, &configMap)
	if configMap == nil {
		configMap = make(map[string]interface{})
	}
	configMap["provider"] = env.Provider
	configMap["slug"] = env.Slug
	configMap["initiated_by"] = actor
	configMap["environment_name"] = env.Name

	if h.temporalClient != nil {
		params := infraTemporal.WorkflowParams{
			WorkflowID:    wf.ID.String(),
			TeamID:        env.TeamID.String(),
			EnvironmentID: env.ID.String(),
			Config:        configMap,
		}

		runID, err := infraTemporal.StartUpdateWorkflow(r.Context(), h.temporalClient, params)
		if err == nil {
			temporalWfID := "update-" + wf.ID.String()
			_ = h.workflows.SetTemporalIDs(r.Context(), wf.ID, temporalWfID, runID)
			wf.TemporalWorkflowID = &temporalWfID
			wf.TemporalRunID = &runID
		} else {
			errMsg := err.Error()
			_ = h.workflows.UpdateStatus(r.Context(), wf.ID, "failed", &errMsg)
			wf.Status = "failed"
			wf.ErrorMessage = &errMsg
		}
	}

	h.audit.LogAction(r.Context(), &env.TeamID, actor, "workflow.created", "workflow", &wf.ID,
		map[string]string{"name": wf.Name, "type": "deploy", "trigger": "auto"})

	return wf
}

func (h *EnvironmentHandler) Decommission(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid environment ID")
		return
	}

	env, err := h.envs.GetByID(r.Context(), id)
	if err != nil {
		Error(w, http.StatusNotFound, "environment not found")
		return
	}

	if env.Status == "decommissioned" {
		Error(w, http.StatusBadRequest, "environment is already decommissioned")
		return
	}

	// Mark as decommissioning
	env.Status = "decommissioning"
	_ = h.envs.Update(r.Context(), env)

	// Auto-resolve all drift records for this environment
	_ = h.drifts.ResolveAllByEnvironment(r.Context(), id, "auto_remediated")

	actor := GetActor(r)
	h.audit.LogAction(r.Context(), &env.TeamID, actor, "environment.decommission_started", "environment", &id,
		map[string]string{"name": env.Name})

	// Trigger Decommission workflow
	wf := &models.Workflow{
		TeamID:        env.TeamID,
		EnvironmentID: &env.ID,
		Name:          fmt.Sprintf("Decommission: %s", env.Name),
		WorkflowType:  "destroy",
		Status:        "pending",
		InitiatedBy:   actor,
	}

	if err := h.workflows.Create(r.Context(), wf); err != nil {
		Error(w, http.StatusInternalServerError, "failed to create decommission workflow")
		return
	}

	var configMap map[string]interface{}
	_ = json.Unmarshal(env.Config, &configMap)
	if configMap == nil {
		configMap = make(map[string]interface{})
	}
	configMap["provider"] = env.Provider
	configMap["slug"] = env.Slug
	configMap["initiated_by"] = actor
	configMap["environment_name"] = env.Name

	if h.temporalClient != nil {
		params := infraTemporal.WorkflowParams{
			WorkflowID:    wf.ID.String(),
			TeamID:        env.TeamID.String(),
			EnvironmentID: env.ID.String(),
			Config:        configMap,
		}

		runID, err := infraTemporal.StartDecommissionWorkflow(r.Context(), h.temporalClient, params)
		if err == nil {
			temporalWfID := "decommission-" + wf.ID.String()
			_ = h.workflows.SetTemporalIDs(r.Context(), wf.ID, temporalWfID, runID)
			wf.TemporalWorkflowID = &temporalWfID
			wf.TemporalRunID = &runID
		} else {
			errMsg := err.Error()
			_ = h.workflows.UpdateStatus(r.Context(), wf.ID, "failed", &errMsg)
			wf.Status = "failed"
			wf.ErrorMessage = &errMsg
		}
	}

	h.audit.LogAction(r.Context(), &env.TeamID, actor, "workflow.created", "workflow", &wf.ID,
		map[string]string{"name": wf.Name, "type": "destroy", "trigger": "decommission"})

	response := map[string]interface{}{
		"environment": env,
		"workflow":    wf,
	}
	JSON(w, http.StatusOK, response)
}

func (h *EnvironmentHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid environment ID")
		return
	}

	env, err := h.envs.GetByID(r.Context(), id)
	if err != nil {
		Error(w, http.StatusNotFound, "environment not found")
		return
	}

	if err := h.envs.Delete(r.Context(), id); err != nil {
		Error(w, http.StatusInternalServerError, "failed to delete environment")
		return
	}

	actor := GetActor(r)
	h.audit.LogAction(r.Context(), &env.TeamID, actor, "environment.deleted", "environment", &id,
		map[string]string{"name": env.Name})

	w.WriteHeader(http.StatusNoContent)
}
