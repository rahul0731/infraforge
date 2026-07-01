package api

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/infraforge/infraforge/internal/repository"
)

// DashboardHandler handles dashboard statistics endpoints.
type DashboardHandler struct {
	dashboard *repository.DashboardRepository
}

// NewDashboardHandler creates a new DashboardHandler.
func NewDashboardHandler(dashboard *repository.DashboardRepository) *DashboardHandler {
	return &DashboardHandler{dashboard: dashboard}
}

// Register mounts dashboard routes on the given mux.
func (h *DashboardHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/dashboard/stats", h.Stats)
}

func (h *DashboardHandler) Stats(w http.ResponseWriter, r *http.Request) {
	var teamID *uuid.UUID
	if teamIDStr := r.URL.Query().Get("team_id"); teamIDStr != "" {
		id, err := uuid.Parse(teamIDStr)
		if err != nil {
			Error(w, http.StatusBadRequest, "invalid team_id")
			return
		}
		teamID = &id
	}

	stats, err := h.dashboard.GetStats(r.Context(), teamID)
	if err != nil {
		Error(w, http.StatusInternalServerError, "failed to get dashboard stats")
		return
	}
	JSON(w, http.StatusOK, stats)
}
