package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/infraforge/infraforge/internal/models"
)

// TeamRepository handles database operations for teams.
type TeamRepository struct {
	pool *pgxpool.Pool
}

// NewTeamRepository creates a new TeamRepository.
func NewTeamRepository(pool *pgxpool.Pool) *TeamRepository {
	return &TeamRepository{pool: pool}
}

// Create inserts a new team.
func (r *TeamRepository) Create(ctx context.Context, team *models.Team) error {
	query := `
		INSERT INTO teams (name, slug, description)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at`

	return r.pool.QueryRow(ctx, query, team.Name, team.Slug, team.Description).
		Scan(&team.ID, &team.CreatedAt, &team.UpdatedAt)
}

// GetByID retrieves a team by its ID.
func (r *TeamRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Team, error) {
	query := `SELECT id, name, slug, description, created_at, updated_at FROM teams WHERE id = $1`

	team := &models.Team{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&team.ID, &team.Name, &team.Slug, &team.Description, &team.CreatedAt, &team.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("getting team %s: %w", id, err)
	}
	return team, nil
}

// List returns all teams.
func (r *TeamRepository) List(ctx context.Context) ([]models.Team, error) {
	query := `SELECT id, name, slug, description, created_at, updated_at FROM teams ORDER BY name`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("listing teams: %w", err)
	}
	defer rows.Close()

	var teams []models.Team
	for rows.Next() {
		var t models.Team
		if err := rows.Scan(&t.ID, &t.Name, &t.Slug, &t.Description, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning team: %w", err)
		}
		teams = append(teams, t)
	}
	return teams, nil
}

// Delete removes a team by ID.
func (r *TeamRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM teams WHERE id = $1`, id)
	return err
}
