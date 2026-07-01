package api

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"go.temporal.io/sdk/client"

	"github.com/infraforge/infraforge/internal/repository"
	infraTemporal "github.com/infraforge/infraforge/internal/temporal"
)

// ApprovalHandler handles approval API endpoints.
type ApprovalHandler struct {
	approvals      *repository.ApprovalRepository
	workflows      *repository.WorkflowRepository
	audit          *repository.AuditRepository
	temporalClient client.Client
}

// NewApprovalHandler creates a new ApprovalHandler.
func NewApprovalHandler(
	approvals *repository.ApprovalRepository,
	workflows *repository.WorkflowRepository,
	audit *repository.AuditRepository,
	temporalClient client.Client,
) *ApprovalHandler {
	return &ApprovalHandler{
		approvals:      approvals,
		workflows:      workflows,
		audit:          audit,
		temporalClient: temporalClient,
	}
}

// Register mounts approval routes on the given mux.
func (h *ApprovalHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/approvals", h.ListPending)
	mux.HandleFunc("GET /api/v1/approvals/history", h.ListHistory)
	mux.HandleFunc("GET /api/v1/approvals/{id}", h.Get)
	mux.HandleFunc("POST /api/v1/approvals/{id}/approve", h.Approve)
	mux.HandleFunc("POST /api/v1/approvals/{id}/reject", h.Reject)
}

func (h *ApprovalHandler) ListPending(w http.ResponseWriter, r *http.Request) {
	assignedTo := r.URL.Query().Get("assigned_to")

	approvals, err := h.approvals.ListPending(r.Context(), assignedTo)
	if err != nil {
		Error(w, http.StatusInternalServerError, "failed to list approvals")
		return
	}
	JSON(w, http.StatusOK, approvals)
}

func (h *ApprovalHandler) ListHistory(w http.ResponseWriter, r *http.Request) {
	approvals, err := h.approvals.ListAll(r.Context(), 50)
	if err != nil {
		Error(w, http.StatusInternalServerError, "failed to list approval history")
		return
	}
	JSON(w, http.StatusOK, approvals)
}

func (h *ApprovalHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid approval ID")
		return
	}

	approval, err := h.approvals.GetByID(r.Context(), id)
	if err != nil {
		Error(w, http.StatusNotFound, "approval not found")
		return
	}
	JSON(w, http.StatusOK, approval)
}

func (h *ApprovalHandler) Approve(w http.ResponseWriter, r *http.Request) {
	h.decide(w, r, "approved")
}

func (h *ApprovalHandler) Reject(w http.ResponseWriter, r *http.Request) {
	h.decide(w, r, "rejected")
}

func (h *ApprovalHandler) decide(w http.ResponseWriter, r *http.Request, decision string) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid approval ID")
		return
	}

	var req struct {
		Reason *string `json:"reason"`
	}
	if r.Body != nil && r.ContentLength > 0 {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}

	// Get approval to find the workflow
	approval, err := h.approvals.GetByID(r.Context(), id)
	if err != nil {
		Error(w, http.StatusNotFound, "approval not found")
		return
	}

	// Update approval in DB
	if err := h.approvals.Decide(r.Context(), id, decision, req.Reason); err != nil {
		Error(w, http.StatusBadRequest, err.Error())
		return
	}

	actor := GetActor(r)

	// Send signal to Temporal workflow
	if h.temporalClient != nil {
		wf, wfErr := h.workflows.GetByID(r.Context(), approval.WorkflowID)
		if wfErr == nil && wf.TemporalWorkflowID != nil {
			signal := infraTemporal.ApprovalSignal{
				Approved: decision == "approved",
				Actor:    actor,
				Reason:   stringVal(req.Reason),
			}

			if signalErr := infraTemporal.SendApprovalSignal(r.Context(), h.temporalClient, *wf.TemporalWorkflowID, signal); signalErr != nil {
				// Log but don't fail the request — the approval decision is recorded
				Error(w, http.StatusInternalServerError, "approval recorded but failed to signal workflow: "+signalErr.Error())
				return
			}

			// If rejected, terminate the workflow
			if decision == "rejected" {
				reason := "rejected by " + actor
				if req.Reason != nil {
					reason += ": " + *req.Reason
				}
				_ = infraTemporal.TerminateWorkflow(r.Context(), h.temporalClient, *wf.TemporalWorkflowID, reason)
				errMsg := reason
				_ = h.workflows.UpdateStatus(r.Context(), wf.ID, "failed", &errMsg)
			}
		}
	}

	h.audit.LogAction(r.Context(), nil, actor, "approval."+decision, "approval", &id,
		map[string]interface{}{"reason": req.Reason, "workflow_id": approval.WorkflowID})

	JSON(w, http.StatusOK, map[string]string{"status": decision})
}

func stringVal(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
