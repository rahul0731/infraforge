package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/infraforge/infraforge/internal/models"
)

// WorkflowRepository handles database operations for workflows.
type WorkflowRepository struct {
	pool *pgxpool.Pool
}

// NewWorkflowRepository creates a new WorkflowRepository.
func NewWorkflowRepository(pool *pgxpool.Pool) *WorkflowRepository {
	return &WorkflowRepository{pool: pool}
}

// Create inserts a new workflow.
func (r *WorkflowRepository) Create(ctx context.Context, wf *models.Workflow) error {
	query := `
		INSERT INTO workflows (team_id, environment_id, name, description, workflow_type, status, initiated_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at`

	return r.pool.QueryRow(ctx, query,
		wf.TeamID, wf.EnvironmentID, wf.Name, wf.Description,
		wf.WorkflowType, wf.Status, wf.InitiatedBy,
	).Scan(&wf.ID, &wf.CreatedAt, &wf.UpdatedAt)
}

// GetByID retrieves a workflow by ID.
func (r *WorkflowRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Workflow, error) {
	query := `SELECT id, team_id, environment_id, name, description, workflow_type, status,
		temporal_workflow_id, temporal_run_id, initiated_by, started_at, completed_at,
		error_message, created_at, updated_at
		FROM workflows WHERE id = $1`

	wf := &models.Workflow{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&wf.ID, &wf.TeamID, &wf.EnvironmentID, &wf.Name, &wf.Description,
		&wf.WorkflowType, &wf.Status, &wf.TemporalWorkflowID, &wf.TemporalRunID,
		&wf.InitiatedBy, &wf.StartedAt, &wf.CompletedAt, &wf.ErrorMessage,
		&wf.CreatedAt, &wf.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("getting workflow %s: %w", id, err)
	}
	return wf, nil
}

// ListByTeam returns all workflows for a team with optional status filter.
func (r *WorkflowRepository) ListByTeam(ctx context.Context, teamID uuid.UUID, status string) ([]models.Workflow, error) {
	var query string
	var args []interface{}

	if status != "" {
		query = `SELECT id, team_id, environment_id, name, description, workflow_type, status,
			temporal_workflow_id, temporal_run_id, initiated_by, started_at, completed_at,
			error_message, created_at, updated_at
			FROM workflows WHERE team_id = $1 AND status = $2 ORDER BY created_at DESC`
		args = []interface{}{teamID, status}
	} else {
		query = `SELECT id, team_id, environment_id, name, description, workflow_type, status,
			temporal_workflow_id, temporal_run_id, initiated_by, started_at, completed_at,
			error_message, created_at, updated_at
			FROM workflows WHERE team_id = $1 ORDER BY created_at DESC`
		args = []interface{}{teamID}
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing workflows: %w", err)
	}
	defer rows.Close()

	var workflows []models.Workflow
	for rows.Next() {
		var wf models.Workflow
		if err := rows.Scan(
			&wf.ID, &wf.TeamID, &wf.EnvironmentID, &wf.Name, &wf.Description,
			&wf.WorkflowType, &wf.Status, &wf.TemporalWorkflowID, &wf.TemporalRunID,
			&wf.InitiatedBy, &wf.StartedAt, &wf.CompletedAt, &wf.ErrorMessage,
			&wf.CreatedAt, &wf.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning workflow: %w", err)
		}
		workflows = append(workflows, wf)
	}
	return workflows, nil
}

// UpdateStatus updates a workflow's status.
func (r *WorkflowRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string, errorMsg *string) error {
	query := `UPDATE workflows SET status = $1, error_message = $2, updated_at = NOW()
		WHERE id = $3`
	_, err := r.pool.Exec(ctx, query, status, errorMsg, id)
	return err
}

// GetSteps returns all steps for a workflow.
func (r *WorkflowRepository) GetSteps(ctx context.Context, workflowID uuid.UUID) ([]models.WorkflowStep, error) {
	query := `SELECT id, workflow_id, name, step_order, step_type, status, input, output,
		error_message, started_at, completed_at, created_at, updated_at
		FROM workflow_steps WHERE workflow_id = $1 ORDER BY step_order`

	rows, err := r.pool.Query(ctx, query, workflowID)
	if err != nil {
		return nil, fmt.Errorf("listing workflow steps: %w", err)
	}
	defer rows.Close()

	var steps []models.WorkflowStep
	for rows.Next() {
		var s models.WorkflowStep
		if err := rows.Scan(
			&s.ID, &s.WorkflowID, &s.Name, &s.StepOrder, &s.StepType, &s.Status,
			&s.Input, &s.Output, &s.ErrorMessage, &s.StartedAt, &s.CompletedAt,
			&s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning step: %w", err)
		}
		steps = append(steps, s)
	}
	return steps, nil
}

// CountByTeam returns the number of active workflows for a team (for quota enforcement).
func (r *WorkflowRepository) CountByTeam(ctx context.Context, teamID uuid.UUID) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM workflows WHERE team_id = $1 AND status IN ('pending', 'running')`,
		teamID,
	).Scan(&count)
	return count, err
}

// SetTemporalIDs stores the Temporal workflow and run IDs.
func (r *WorkflowRepository) SetTemporalIDs(ctx context.Context, id uuid.UUID, temporalWfID, runID string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE workflows SET temporal_workflow_id = $1, temporal_run_id = $2, updated_at = NOW() WHERE id = $3`,
		temporalWfID, runID, id,
	)
	return err
}

// DeleteSteps removes all steps for a workflow (used before retry).
func (r *WorkflowRepository) DeleteSteps(ctx context.Context, workflowID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM workflow_steps WHERE workflow_id = $1`, workflowID)
	return err
}
