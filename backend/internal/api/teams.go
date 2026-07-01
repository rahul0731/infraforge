package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/infraforge/infraforge/internal/models"
	"github.com/infraforge/infraforge/internal/repository"
)

// TeamHandler handles team API endpoints.
type TeamHandler struct {
	teams *repository.TeamRepository
	audit *repository.AuditRepository
}

// NewTeamHandler creates a new TeamHandler.
func NewTeamHandler(teams *repository.TeamRepository, audit *repository.AuditRepository) *TeamHandler {
	return &TeamHandler{teams: teams, audit: audit}
}

// Register mounts team routes on the given mux.
func (h *TeamHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/teams", h.List)
	mux.HandleFunc("POST /api/v1/teams", h.Create)
	mux.HandleFunc("GET /api/v1/teams/{id}", h.Get)
	mux.HandleFunc("DELETE /api/v1/teams/{id}", h.Delete)
}

func (h *TeamHandler) List(w http.ResponseWriter, r *http.Request) {
	teams, err := h.teams.List(r.Context())
	if err != nil {
		Error(w, http.StatusInternalServerError, "failed to list teams")
		return
	}
	JSON(w, http.StatusOK, teams)
}

func (h *TeamHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string  `json:"name"`
		Slug        string  `json:"slug"`
		Description *string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" || req.Slug == "" {
		Error(w, http.StatusBadRequest, "name and slug are required")
		return
	}

	team := &models.Team{
		Name:        req.Name,
		Slug:        strings.ToLower(req.Slug),
		Description: req.Description,
	}

	if err := h.teams.Create(r.Context(), team); err != nil {
		if strings.Contains(err.Error(), "duplicate") {
			Error(w, http.StatusConflict, "team with this name or slug already exists")
			return
		}
		Error(w, http.StatusInternalServerError, "failed to create team")
		return
	}

	actor := GetActor(r)
	h.audit.LogAction(r.Context(), &team.ID, actor, "team.created", "team", &team.ID, map[string]string{"name": team.Name})

	JSON(w, http.StatusCreated, team)
}

func (h *TeamHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid team ID")
		return
	}

	team, err := h.teams.GetByID(r.Context(), id)
	if err != nil {
		Error(w, http.StatusNotFound, "team not found")
		return
	}
	JSON(w, http.StatusOK, team)
}

func (h *TeamHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid team ID")
		return
	}

	if err := h.teams.Delete(r.Context(), id); err != nil {
		Error(w, http.StatusInternalServerError, "failed to delete team")
		return
	}

	actor := GetActor(r)
	h.audit.LogAction(r.Context(), &id, actor, "team.deleted", "team", &id, nil)

	w.WriteHeader(http.StatusNoContent)
}
