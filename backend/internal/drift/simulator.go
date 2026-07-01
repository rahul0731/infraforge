package drift

import (
	"context"
	"encoding/json"
	"log"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/infraforge/infraforge/internal/models"
	"github.com/infraforge/infraforge/internal/repository"
)

// Simulator runs a background goroutine that simulates infrastructure drift.
type Simulator struct {
	pool   *pgxpool.Pool
	drifts *repository.DriftRepository
}

// NewSimulator creates a new drift simulator.
func NewSimulator(pool *pgxpool.Pool, drifts *repository.DriftRepository) *Simulator {
	return &Simulator{pool: pool, drifts: drifts}
}

// Start begins the drift simulation loop. Call in a goroutine.
func (s *Simulator) Start(ctx context.Context) {
	log.Println("Drift simulator started (30s interval, 10% chance per active environment)")
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Drift simulator stopped")
			return
		case <-ticker.C:
			s.check(ctx)
		}
	}
}

func (s *Simulator) check(ctx context.Context) {
	// Get all active environments
	rows, err := s.pool.Query(ctx,
		`SELECT id, team_id, name, slug, config FROM environments WHERE status = 'active'`,
	)
	if err != nil {
		log.Printf("drift simulator: failed to query environments: %v", err)
		return
	}
	defer rows.Close()

	type envInfo struct {
		ID     uuid.UUID
		TeamID uuid.UUID
		Name   string
		Slug   string
		Config json.RawMessage
	}

	var envs []envInfo
	for rows.Next() {
		var e envInfo
		if err := rows.Scan(&e.ID, &e.TeamID, &e.Name, &e.Slug, &e.Config); err != nil {
			continue
		}
		envs = append(envs, e)
	}

	for _, env := range envs {
		// 10% chance of drift per environment
		if rand.Float64() >= 0.10 {
			continue
		}

		// Parse config to get instance_count
		var config map[string]interface{}
		if err := json.Unmarshal(env.Config, &config); err != nil {
			continue
		}

		desiredCount := 2 // default
		if ic, ok := config["instance_count"]; ok {
			switch v := ic.(type) {
			case float64:
				desiredCount = int(v)
			case int:
				desiredCount = v
			}
		}

		// Calculate drifted count
		actualCount := desiredCount - 1
		if desiredCount <= 1 {
			actualCount = desiredCount + 1
		}

		resourceType := "compute"
		resourceID := "instance_count"

		// Check for duplicates — don't insert if there's already an unresolved record
		exists, err := s.drifts.HasUnresolvedDrift(ctx, env.ID, resourceType, resourceID)
		if err != nil || exists {
			continue
		}

		// Insert drift record
		expectedState, _ := json.Marshal(map[string]interface{}{
			"field":          "instance_count",
			"desired_value":  desiredCount,
		})
		actualState, _ := json.Marshal(map[string]interface{}{
			"field":        "instance_count",
			"actual_value": actualCount,
		})

		severity := "medium"
		if desiredCount-actualCount > 1 || actualCount == 0 {
			severity = "high"
		}

		record := &models.DriftRecord{
			EnvironmentID: env.ID,
			ResourceType:  resourceType,
			ResourceID:    resourceID,
			ExpectedState: expectedState,
			ActualState:   actualState,
			Severity:      severity,
		}

		if err := s.drifts.Create(ctx, record); err != nil {
			log.Printf("drift simulator: failed to insert drift for %s: %v", env.Name, err)
			continue
		}

		log.Printf("drift simulator: drift detected for %s — instance_count expected=%d actual=%d",
			env.Name, desiredCount, actualCount)
	}
}
