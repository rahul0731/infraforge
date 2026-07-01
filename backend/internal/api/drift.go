package api

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"

	"github.com/infraforge/infraforge/internal/models"
	"github.com/infraforge/infraforge/internal/repository"
)

// DriftHandler handles drift detection API endpoints.
type DriftHandler struct {
	drifts *repository.DriftRepository
	audit  *repository.AuditRepository
}

// NewDriftHandler creates a new DriftHandler.
func NewDriftHandler(drifts *repository.DriftRepository, audit *repository.AuditRepository) *DriftHandler {
	return &DriftHandler{drifts: drifts, audit: audit}
}

// Register mounts drift routes on the given mux.
func (h *DriftHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/drift", h.List)
	mux.HandleFunc("POST /api/v1/drift", h.Report)
	mux.HandleFunc("GET /api/v1/drift/{id}", h.Get)
	mux.HandleFunc("POST /api/v1/drift/{id}/resolve", h.Resolve)
}

func (h *DriftHandler) List(w http.ResponseWriter, r *http.Request) {
	envIDStr := r.URL.Query().Get("environment_id")
	unresolvedOnly := r.URL.Query().Get("unresolved") == "true"

	if envIDStr == "" {
		// Return all drift records across all environments
		records, err := h.drifts.ListAll(r.Context(), unresolvedOnly)
		if err != nil {
			Error(w, http.StatusInternalServerError, "failed to list drift records")
			return
		}
		JSON(w, http.StatusOK, records)
		return
	}

	envID, err := uuid.Parse(envIDStr)
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid environment_id")
		return
	}

	records, err := h.drifts.ListByEnvironment(r.Context(), envID, unresolvedOnly)
	if err != nil {
		Error(w, http.StatusInternalServerError, "failed to list drift records")
		return
	}
	JSON(w, http.StatusOK, records)
}

func (h *DriftHandler) Report(w http.ResponseWriter, r *http.Request) {
	var req struct {
		EnvironmentID uuid.UUID       `json:"environment_id"`
		WorkflowID    *uuid.UUID      `json:"workflow_id"`
		ResourceType  string          `json:"resource_type"`
		ResourceID    string          `json:"resource_id"`
		ExpectedState json.RawMessage `json:"expected_state"`
		ActualState   json.RawMessage `json:"actual_state"`
		Severity      string          `json:"severity"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.ResourceType == "" || req.ResourceID == "" {
		Error(w, http.StatusBadRequest, "resource_type and resource_id are required")
		return
	}

	validSeverities := map[string]bool{"low": true, "medium": true, "high": true, "critical": true}
	if req.Severity == "" {
		req.Severity = "medium"
	}
	if !validSeverities[req.Severity] {
		Error(w, http.StatusBadRequest, "severity must be one of: low, medium, high, critical")
		return
	}

	if req.ExpectedState == nil {
		req.ExpectedState = json.RawMessage(`{}`)
	}
	if req.ActualState == nil {
		req.ActualState = json.RawMessage(`{}`)
	}

	record := &models.DriftRecord{
		EnvironmentID: req.EnvironmentID,
		WorkflowID:    req.WorkflowID,
		ResourceType:  req.ResourceType,
		ResourceID:    req.ResourceID,
		ExpectedState: req.ExpectedState,
		ActualState:   req.ActualState,
		Severity:      req.Severity,
	}

	if err := h.drifts.Create(r.Context(), record); err != nil {
		Error(w, http.StatusInternalServerError, "failed to report drift")
		return
	}

	actor := GetActor(r)
	h.audit.LogAction(r.Context(), nil, actor, "drift.reported", "drift_record", &record.ID,
		map[string]string{"resource": req.ResourceType + "/" + req.ResourceID, "severity": req.Severity})

	JSON(w, http.StatusCreated, record)
}

func (h *DriftHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid drift record ID")
		return
	}

	record, err := h.drifts.GetByID(r.Context(), id)
	if err != nil {
		Error(w, http.StatusNotFound, "drift record not found")
		return
	}
	JSON(w, http.StatusOK, record)
}

func (h *DriftHandler) Resolve(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid drift record ID")
		return
	}

	var req struct {
		Resolution string `json:"resolution"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	validResolutions := map[string]bool{
		"auto_remediated": true, "manual_fix": true, "accepted": true, "ignored": true,
	}
	if !validResolutions[req.Resolution] {
		Error(w, http.StatusBadRequest, "resolution must be one of: auto_remediated, manual_fix, accepted, ignored")
		return
	}

	if err := h.drifts.Resolve(r.Context(), id, req.Resolution); err != nil {
		Error(w, http.StatusBadRequest, err.Error())
		return
	}

	actor := GetActor(r)
	h.audit.LogAction(r.Context(), nil, actor, "drift.resolved", "drift_record", &id,
		map[string]string{"resolution": req.Resolution})

	JSON(w, http.StatusOK, map[string]string{"status": "resolved", "resolution": req.Resolution})
}
