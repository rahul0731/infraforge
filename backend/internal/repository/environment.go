package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/infraforge/infraforge/internal/models"
)

// EnvironmentRepository handles database operations for environments.
type EnvironmentRepository struct {
	pool *pgxpool.Pool
}

// NewEnvironmentRepository creates a new EnvironmentRepository.
func NewEnvironmentRepository(pool *pgxpool.Pool) *EnvironmentRepository {
	return &EnvironmentRepository{pool: pool}
}

// Create inserts a new environment.
func (r *EnvironmentRepository) Create(ctx context.Context, env *models.Environment) error {
	query := `
		INSERT INTO environments (team_id, name, slug, provider, region, config, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at`

	return r.pool.QueryRow(ctx, query,
		env.TeamID, env.Name, env.Slug, env.Provider, env.Region, env.Config, env.Status,
	).Scan(&env.ID, &env.CreatedAt, &env.UpdatedAt)
}

// GetByID retrieves an environment by ID.
func (r *EnvironmentRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Environment, error) {
	query := `SELECT id, team_id, name, slug, provider, region, config, status, created_at, updated_at
		FROM environments WHERE id = $1`

	env := &models.Environment{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&env.ID, &env.TeamID, &env.Name, &env.Slug, &env.Provider,
		&env.Region, &env.Config, &env.Status, &env.CreatedAt, &env.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("getting environment %s: %w", id, err)
	}
	return env, nil
}

// ListByTeam returns all environments for a team.
func (r *EnvironmentRepository) ListByTeam(ctx context.Context, teamID uuid.UUID) ([]models.Environment, error) {
	query := `SELECT id, team_id, name, slug, provider, region, config, status, created_at, updated_at
		FROM environments WHERE team_id = $1 ORDER BY name`

	rows, err := r.pool.Query(ctx, query, teamID)
	if err != nil {
		return nil, fmt.Errorf("listing environments: %w", err)
	}
	defer rows.Close()

	var envs []models.Environment
	for rows.Next() {
		var e models.Environment
		if err := rows.Scan(&e.ID, &e.TeamID, &e.Name, &e.Slug, &e.Provider,
			&e.Region, &e.Config, &e.Status, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning environment: %w", err)
		}
		envs = append(envs, e)
	}
	return envs, nil
}

// Update updates an environment.
func (r *EnvironmentRepository) Update(ctx context.Context, env *models.Environment) error {
	query := `UPDATE environments SET name = $1, slug = $2, provider = $3, region = $4,
		config = $5, status = $6, updated_at = NOW()
		WHERE id = $7 RETURNING updated_at`

	return r.pool.QueryRow(ctx, query,
		env.Name, env.Slug, env.Provider, env.Region, env.Config, env.Status, env.ID,
	).Scan(&env.UpdatedAt)
}

// Delete removes an environment by ID.
func (r *EnvironmentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM environments WHERE id = $1`, id)
	return err
}

// CountByTeam returns the number of environments for a team (for quota enforcement).
func (r *EnvironmentRepository) CountByTeam(ctx context.Context, teamID uuid.UUID) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM environments WHERE team_id = $1`, teamID).Scan(&count)
	return count, err
}
