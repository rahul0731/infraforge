package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/infraforge/infraforge/internal/models"
)

// ApprovalRepository handles database operations for approvals.
type ApprovalRepository struct {
	pool *pgxpool.Pool
}

// NewApprovalRepository creates a new ApprovalRepository.
func NewApprovalRepository(pool *pgxpool.Pool) *ApprovalRepository {
	return &ApprovalRepository{pool: pool}
}

// Create inserts a new approval request.
func (r *ApprovalRepository) Create(ctx context.Context, a *models.Approval) error {
	query := `
		INSERT INTO approvals (workflow_id, workflow_step_id, requested_by, assigned_to, status, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at`

	return r.pool.QueryRow(ctx, query,
		a.WorkflowID, a.WorkflowStepID, a.RequestedBy, a.AssignedTo, a.Status, a.ExpiresAt,
	).Scan(&a.ID, &a.CreatedAt, &a.UpdatedAt)
}

// GetByID retrieves an approval by ID.
func (r *ApprovalRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Approval, error) {
	query := `SELECT id, workflow_id, workflow_step_id, requested_by, assigned_to, status,
		decision_reason, decided_at, expires_at, created_at, updated_at
		FROM approvals WHERE id = $1`

	a := &models.Approval{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&a.ID, &a.WorkflowID, &a.WorkflowStepID, &a.RequestedBy, &a.AssignedTo,
		&a.Status, &a.DecisionReason, &a.DecidedAt, &a.ExpiresAt, &a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("getting approval %s: %w", id, err)
	}
	return a, nil
}

// ListPending returns all pending approvals, optionally filtered by assignee.
func (r *ApprovalRepository) ListPending(ctx context.Context, assignedTo string) ([]models.Approval, error) {
	var query string
	var args []interface{}

	if assignedTo != "" {
		query = `SELECT id, workflow_id, workflow_step_id, requested_by, assigned_to, status,
			decision_reason, decided_at, expires_at, created_at, updated_at
			FROM approvals WHERE status = 'pending' AND assigned_to = $1 ORDER BY created_at DESC`
		args = []interface{}{assignedTo}
	} else {
		query = `SELECT id, workflow_id, workflow_step_id, requested_by, assigned_to, status,
			decision_reason, decided_at, expires_at, created_at, updated_at
			FROM approvals WHERE status = 'pending' ORDER BY created_at DESC`
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing pending approvals: %w", err)
	}
	defer rows.Close()

	var approvals []models.Approval
	for rows.Next() {
		var a models.Approval
		if err := rows.Scan(
			&a.ID, &a.WorkflowID, &a.WorkflowStepID, &a.RequestedBy, &a.AssignedTo,
			&a.Status, &a.DecisionReason, &a.DecidedAt, &a.ExpiresAt, &a.CreatedAt, &a.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning approval: %w", err)
		}
		approvals = append(approvals, a)
	}
	return approvals, nil
}

// Decide approves or rejects an approval.
func (r *ApprovalRepository) Decide(ctx context.Context, id uuid.UUID, status string, reason *string) error {
	now := time.Now()
	query := `UPDATE approvals SET status = $1, decision_reason = $2, decided_at = $3, updated_at = NOW()
		WHERE id = $4 AND status = 'pending'`

	tag, err := r.pool.Exec(ctx, query, status, reason, now, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("approval %s not found or already decided", id)
	}
	return nil
}
