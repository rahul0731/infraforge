package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DashboardStats holds aggregated platform statistics.
type DashboardStats struct {
	TotalTeams          int `json:"total_teams"`
	TotalEnvironments   int `json:"total_environments"`
	ActiveWorkflows     int `json:"active_workflows"`
	CompletedWorkflows  int `json:"completed_workflows"`
	FailedWorkflows     int `json:"failed_workflows"`
	PendingApprovals    int `json:"pending_approvals"`
	UnresolvedDrifts    int `json:"unresolved_drifts"`
	CriticalDrifts      int `json:"critical_drifts"`
}

// DashboardRepository handles dashboard statistics queries.
type DashboardRepository struct {
	pool *pgxpool.Pool
}

// NewDashboardRepository creates a new DashboardRepository.
func NewDashboardRepository(pool *pgxpool.Pool) *DashboardRepository {
	return &DashboardRepository{pool: pool}
}

// GetStats returns aggregated platform statistics, optionally filtered by team.
func (r *DashboardRepository) GetStats(ctx context.Context, teamID *uuid.UUID) (*DashboardStats, error) {
	stats := &DashboardStats{}

	if teamID != nil {
		err := r.pool.QueryRow(ctx,
			`SELECT COUNT(*) FROM environments WHERE team_id = $1`, *teamID,
		).Scan(&stats.TotalEnvironments)
		if err != nil {
			return nil, fmt.Errorf("counting environments: %w", err)
		}

		stats.TotalTeams = 1

		err = r.pool.QueryRow(ctx,
			`SELECT COUNT(*) FROM workflows WHERE team_id = $1 AND status IN ('pending', 'running')`, *teamID,
		).Scan(&stats.ActiveWorkflows)
		if err != nil {
			return nil, fmt.Errorf("counting active workflows: %w", err)
		}

		err = r.pool.QueryRow(ctx,
			`SELECT COUNT(*) FROM workflows WHERE team_id = $1 AND status = 'completed'`, *teamID,
		).Scan(&stats.CompletedWorkflows)
		if err != nil {
			return nil, fmt.Errorf("counting completed workflows: %w", err)
		}

		err = r.pool.QueryRow(ctx,
			`SELECT COUNT(*) FROM workflows WHERE team_id = $1 AND status = 'failed'`, *teamID,
		).Scan(&stats.FailedWorkflows)
		if err != nil {
			return nil, fmt.Errorf("counting failed workflows: %w", err)
		}
	} else {
		err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM teams`).Scan(&stats.TotalTeams)
		if err != nil {
			return nil, fmt.Errorf("counting teams: %w", err)
		}

		err = r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM environments`).Scan(&stats.TotalEnvironments)
		if err != nil {
			return nil, fmt.Errorf("counting environments: %w", err)
		}

		err = r.pool.QueryRow(ctx,
			`SELECT COUNT(*) FROM workflows WHERE status IN ('pending', 'running')`,
		).Scan(&stats.ActiveWorkflows)
		if err != nil {
			return nil, fmt.Errorf("counting active workflows: %w", err)
		}

		err = r.pool.QueryRow(ctx,
			`SELECT COUNT(*) FROM workflows WHERE status = 'completed'`,
		).Scan(&stats.CompletedWorkflows)
		if err != nil {
			return nil, fmt.Errorf("counting completed workflows: %w", err)
		}

		err = r.pool.QueryRow(ctx,
			`SELECT COUNT(*) FROM workflows WHERE status = 'failed'`,
		).Scan(&stats.FailedWorkflows)
		if err != nil {
			return nil, fmt.Errorf("counting failed workflows: %w", err)
		}
	}

	// Pending approvals (always global)
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM approvals WHERE status = 'pending'`,
	).Scan(&stats.PendingApprovals)
	if err != nil {
		return nil, fmt.Errorf("counting pending approvals: %w", err)
	}

	// Unresolved drifts
	err = r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM drift_records WHERE resolved_at IS NULL`,
	).Scan(&stats.UnresolvedDrifts)
	if err != nil {
		return nil, fmt.Errorf("counting unresolved drifts: %w", err)
	}

	// Critical drifts
	err = r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM drift_records WHERE resolved_at IS NULL AND severity = 'critical'`,
	).Scan(&stats.CriticalDrifts)
	if err != nil {
		return nil, fmt.Errorf("counting critical drifts: %w", err)
	}

	return stats, nil
}
