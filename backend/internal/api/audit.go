package api

import (
	"net/http"
	"strconv"

	"github.com/google/uuid"

	"github.com/infraforge/infraforge/internal/repository"
)

// AuditHandler handles audit log API endpoints.
type AuditHandler struct {
	audit *repository.AuditRepository
}

// NewAuditHandler creates a new AuditHandler.
func NewAuditHandler(audit *repository.AuditRepository) *AuditHandler {
	return &AuditHandler{audit: audit}
}

// Register mounts audit routes on the given mux.
func (h *AuditHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/audit", h.List)
}

func (h *AuditHandler) List(w http.ResponseWriter, r *http.Request) {
	var teamID *uuid.UUID
	if teamIDStr := r.URL.Query().Get("team_id"); teamIDStr != "" {
		id, err := uuid.Parse(teamIDStr)
		if err != nil {
			Error(w, http.StatusBadRequest, "invalid team_id")
			return
		}
		teamID = &id
	}

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 100 {
			limit = v
		}
	}

	offset := 0
	if o := r.URL.Query().Get("offset"); o != "" {
		if v, err := strconv.Atoi(o); err == nil && v >= 0 {
			offset = v
		}
	}

	logs, err := h.audit.List(r.Context(), teamID, limit, offset)
	if err != nil {
		Error(w, http.StatusInternalServerError, "failed to list audit logs")
		return
	}
	JSON(w, http.StatusOK, logs)
}
