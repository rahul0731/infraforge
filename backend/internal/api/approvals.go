package api

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"

	"github.com/infraforge/infraforge/internal/repository"
)

// ApprovalHandler handles approval API endpoints.
type ApprovalHandler struct {
	approvals *repository.ApprovalRepository
	audit     *repository.AuditRepository
}

// NewApprovalHandler creates a new ApprovalHandler.
func NewApprovalHandler(approvals *repository.ApprovalRepository, audit *repository.AuditRepository) *ApprovalHandler {
	return &ApprovalHandler{approvals: approvals, audit: audit}
}

// Register mounts approval routes on the given mux.
func (h *ApprovalHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/approvals", h.ListPending)
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

	if err := h.approvals.Decide(r.Context(), id, decision, req.Reason); err != nil {
		Error(w, http.StatusBadRequest, err.Error())
		return
	}

	actor := GetActor(r)
	h.audit.LogAction(r.Context(), nil, actor, "approval."+decision, "approval", &id,
		map[string]interface{}{"reason": req.Reason})

	JSON(w, http.StatusOK, map[string]string{"status": decision})
}
