package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/infraforge/infraforge/internal/models"
)

// AuditRepository handles database operations for audit logs.
type AuditRepository struct {
	pool *pgxpool.Pool
}

// NewAuditRepository creates a new AuditRepository.
func NewAuditRepository(pool *pgxpool.Pool) *AuditRepository {
	return &AuditRepository{pool: pool}
}

// Log creates an audit log entry.
func (r *AuditRepository) Log(ctx context.Context, entry *models.AuditLog) error {
	query := `
		INSERT INTO audit_log (team_id, actor, action, resource_type, resource_id, details, ip_address)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at`

	return r.pool.QueryRow(ctx, query,
		entry.TeamID, entry.Actor, entry.Action, entry.ResourceType,
		entry.ResourceID, entry.Details, entry.IPAddress,
	).Scan(&entry.ID, &entry.CreatedAt)
}

// LogAction is a convenience method for creating audit entries.
func (r *AuditRepository) LogAction(ctx context.Context, teamID *uuid.UUID, actor, action, resourceType string, resourceID *uuid.UUID, details interface{}) {
	detailsJSON, _ := json.Marshal(details)
	entry := &models.AuditLog{
		TeamID:       teamID,
		Actor:        actor,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Details:      json.RawMessage(detailsJSON),
	}
	// Fire and forget - audit logging should not block requests
	_ = r.Log(ctx, entry)
}

// List returns audit logs with optional filters.
func (r *AuditRepository) List(ctx context.Context, teamID *uuid.UUID, limit, offset int) ([]models.AuditLog, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	var query string
	var args []interface{}

	if teamID != nil {
		query = `SELECT id, team_id, actor, action, resource_type, resource_id, details, ip_address, created_at
			FROM audit_log WHERE team_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`
		args = []interface{}{*teamID, limit, offset}
	} else {
		query = `SELECT id, team_id, actor, action, resource_type, resource_id, details, ip_address, created_at
			FROM audit_log ORDER BY created_at DESC LIMIT $1 OFFSET $2`
		args = []interface{}{limit, offset}
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing audit logs: %w", err)
	}
	defer rows.Close()

	var logs []models.AuditLog
	for rows.Next() {
		var l models.AuditLog
		if err := rows.Scan(
			&l.ID, &l.TeamID, &l.Actor, &l.Action, &l.ResourceType,
			&l.ResourceID, &l.Details, &l.IPAddress, &l.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning audit log: %w", err)
		}
		logs = append(logs, l)
	}
	return logs, nil
}
