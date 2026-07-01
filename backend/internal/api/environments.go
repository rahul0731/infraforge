package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/infraforge/infraforge/internal/models"
	"github.com/infraforge/infraforge/internal/repository"
)

const maxEnvironmentsPerTeam = 20

// EnvironmentHandler handles environment API endpoints.
type EnvironmentHandler struct {
	envs  *repository.EnvironmentRepository
	audit *repository.AuditRepository
}

// NewEnvironmentHandler creates a new EnvironmentHandler.
func NewEnvironmentHandler(envs *repository.EnvironmentRepository, audit *repository.AuditRepository) *EnvironmentHandler {
	return &EnvironmentHandler{envs: envs, audit: audit}
}

// Register mounts environment routes on the given mux.
func (h *EnvironmentHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/environments", h.List)
	mux.HandleFunc("POST /api/v1/environments", h.Create)
	mux.HandleFunc("GET /api/v1/environments/{id}", h.Get)
	mux.HandleFunc("PUT /api/v1/environments/{id}", h.Update)
	mux.HandleFunc("DELETE /api/v1/environments/{id}", h.Delete)
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
		Status:   "active",
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

	JSON(w, http.StatusCreated, env)
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

	JSON(w, http.StatusOK, existing)
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
