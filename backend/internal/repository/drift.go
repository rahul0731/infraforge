package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/infraforge/infraforge/internal/models"
)

// DriftRepository handles database operations for drift records.
type DriftRepository struct {
	pool *pgxpool.Pool
}

// NewDriftRepository creates a new DriftRepository.
func NewDriftRepository(pool *pgxpool.Pool) *DriftRepository {
	return &DriftRepository{pool: pool}
}

// Create inserts a new drift record.
func (r *DriftRepository) Create(ctx context.Context, d *models.DriftRecord) error {
	query := `
		INSERT INTO drift_records (environment_id, workflow_id, resource_type, resource_id,
			expected_state, actual_state, severity)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, drift_detected_at, created_at, updated_at`

	return r.pool.QueryRow(ctx, query,
		d.EnvironmentID, d.WorkflowID, d.ResourceType, d.ResourceID,
		d.ExpectedState, d.ActualState, d.Severity,
	).Scan(&d.ID, &d.DriftDetectedAt, &d.CreatedAt, &d.UpdatedAt)
}

// GetByID retrieves a drift record by ID.
func (r *DriftRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.DriftRecord, error) {
	query := `SELECT id, environment_id, workflow_id, resource_type, resource_id,
		expected_state, actual_state, drift_detected_at, resolved_at, resolution, severity,
		created_at, updated_at
		FROM drift_records WHERE id = $1`

	d := &models.DriftRecord{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&d.ID, &d.EnvironmentID, &d.WorkflowID, &d.ResourceType, &d.ResourceID,
		&d.ExpectedState, &d.ActualState, &d.DriftDetectedAt, &d.ResolvedAt,
		&d.Resolution, &d.Severity, &d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("getting drift record %s: %w", id, err)
	}
	return d, nil
}

// ListByEnvironment returns drift records for an environment.
func (r *DriftRepository) ListByEnvironment(ctx context.Context, envID uuid.UUID, unresolvedOnly bool) ([]models.DriftRecord, error) {
	var query string
	if unresolvedOnly {
		query = `SELECT id, environment_id, workflow_id, resource_type, resource_id,
			expected_state, actual_state, drift_detected_at, resolved_at, resolution, severity,
			created_at, updated_at
			FROM drift_records WHERE environment_id = $1 AND resolved_at IS NULL ORDER BY drift_detected_at DESC`
	} else {
		query = `SELECT id, environment_id, workflow_id, resource_type, resource_id,
			expected_state, actual_state, drift_detected_at, resolved_at, resolution, severity,
			created_at, updated_at
			FROM drift_records WHERE environment_id = $1 ORDER BY drift_detected_at DESC`
	}

	rows, err := r.pool.Query(ctx, query, envID)
	if err != nil {
		return nil, fmt.Errorf("listing drift records: %w", err)
	}
	defer rows.Close()

	var records []models.DriftRecord
	for rows.Next() {
		var d models.DriftRecord
		if err := rows.Scan(
			&d.ID, &d.EnvironmentID, &d.WorkflowID, &d.ResourceType, &d.ResourceID,
			&d.ExpectedState, &d.ActualState, &d.DriftDetectedAt, &d.ResolvedAt,
			&d.Resolution, &d.Severity, &d.CreatedAt, &d.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning drift record: %w", err)
		}
		records = append(records, d)
	}
	return records, nil
}

// Resolve marks a drift record as resolved.
func (r *DriftRepository) Resolve(ctx context.Context, id uuid.UUID, resolution string) error {
	now := time.Now()
	query := `UPDATE drift_records SET resolved_at = $1, resolution = $2, updated_at = NOW()
		WHERE id = $3 AND resolved_at IS NULL`

	tag, err := r.pool.Exec(ctx, query, now, resolution, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("drift record %s not found or already resolved", id)
	}
	return nil
}

// CountUnresolved returns the number of unresolved drift records for an environment.
func (r *DriftRepository) CountUnresolved(ctx context.Context, envID uuid.UUID) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM drift_records WHERE environment_id = $1 AND resolved_at IS NULL`,
		envID,
	).Scan(&count)
	return count, err
}

// ListAll returns all drift records across all environments.
func (r *DriftRepository) ListAll(ctx context.Context, unresolvedOnly bool) ([]models.DriftRecord, error) {
	var query string
	if unresolvedOnly {
		query = `SELECT id, environment_id, workflow_id, resource_type, resource_id,
			expected_state, actual_state, drift_detected_at, resolved_at, resolution, severity,
			created_at, updated_at
			FROM drift_records WHERE resolved_at IS NULL ORDER BY drift_detected_at DESC LIMIT 100`
	} else {
		query = `SELECT id, environment_id, workflow_id, resource_type, resource_id,
			expected_state, actual_state, drift_detected_at, resolved_at, resolution, severity,
			created_at, updated_at
			FROM drift_records ORDER BY drift_detected_at DESC LIMIT 100`
	}

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("listing all drift records: %w", err)
	}
	defer rows.Close()

	var records []models.DriftRecord
	for rows.Next() {
		var d models.DriftRecord
		if err := rows.Scan(
			&d.ID, &d.EnvironmentID, &d.WorkflowID, &d.ResourceType, &d.ResourceID,
			&d.ExpectedState, &d.ActualState, &d.DriftDetectedAt, &d.ResolvedAt,
			&d.Resolution, &d.Severity, &d.CreatedAt, &d.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning drift record: %w", err)
		}
		records = append(records, d)
	}
	return records, nil
}

// HasUnresolvedDrift checks if there's already an unresolved drift record for a given environment and resource type/id.
func (r *DriftRepository) HasUnresolvedDrift(ctx context.Context, envID uuid.UUID, resourceType, resourceID string) (bool, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM drift_records WHERE environment_id = $1 AND resource_type = $2 AND resource_id = $3 AND resolved_at IS NULL`,
		envID, resourceType, resourceID,
	).Scan(&count)
	return count > 0, err
}

// ResolveAllByEnvironment resolves all unresolved drift records for an environment.
func (r *DriftRepository) ResolveAllByEnvironment(ctx context.Context, envID uuid.UUID, resolution string) error {
	now := time.Now()
	_, err := r.pool.Exec(ctx,
		`UPDATE drift_records SET resolved_at = $1, resolution = $2, updated_at = NOW() WHERE environment_id = $3 AND resolved_at IS NULL`,
		now, resolution, envID,
	)
	return err
}
