package models

import (
	"time"

	"github.com/google/uuid"
)

// Team represents an engineering team using InfraForge.
type Team struct {
	ID          uuid.UUID `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Slug        string    `json:"slug" db:"slug"`
	Description *string   `json:"description,omitempty" db:"description"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// Environment represents a deployment target (e.g., staging, production).
type Environment struct {
	ID        uuid.UUID `json:"id" db:"id"`
	TeamID    uuid.UUID `json:"team_id" db:"team_id"`
	Name      string    `json:"name" db:"name"`
	Slug      string    `json:"slug" db:"slug"`
	Provider  string    `json:"provider" db:"provider"` // aws, gcp, azure, k8s
	Region    *string   `json:"region,omitempty" db:"region"`
	Config    []byte    `json:"config" db:"config"` // JSONB
	Status    string    `json:"status" db:"status"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// Workflow represents an infrastructure operation orchestrated by Temporal.
type Workflow struct {
	ID                 uuid.UUID  `json:"id" db:"id"`
	TeamID             uuid.UUID  `json:"team_id" db:"team_id"`
	EnvironmentID      *uuid.UUID `json:"environment_id,omitempty" db:"environment_id"`
	Name               string     `json:"name" db:"name"`
	Description        *string    `json:"description,omitempty" db:"description"`
	WorkflowType       string     `json:"workflow_type" db:"workflow_type"` // deploy, provision, destroy, drift_check
	Status             string     `json:"status" db:"status"`               // pending, running, completed, failed, cancelled
	TemporalWorkflowID *string    `json:"temporal_workflow_id,omitempty" db:"temporal_workflow_id"`
	TemporalRunID      *string    `json:"temporal_run_id,omitempty" db:"temporal_run_id"`
	InitiatedBy        string     `json:"initiated_by" db:"initiated_by"`
	StartedAt          *time.Time `json:"started_at,omitempty" db:"started_at"`
	CompletedAt        *time.Time `json:"completed_at,omitempty" db:"completed_at"`
	ErrorMessage       *string    `json:"error_message,omitempty" db:"error_message"`
	CreatedAt          time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at" db:"updated_at"`
}

// WorkflowStep represents a single step within a workflow.
type WorkflowStep struct {
	ID           uuid.UUID  `json:"id" db:"id"`
	WorkflowID   uuid.UUID  `json:"workflow_id" db:"workflow_id"`
	Name         string     `json:"name" db:"name"`
	StepOrder    int        `json:"step_order" db:"step_order"`
	StepType     string     `json:"step_type" db:"step_type"` // plan, apply, approve, notify, validate
	Status       string     `json:"status" db:"status"`       // pending, running, completed, failed, skipped
	Input        []byte     `json:"input" db:"input"`         // JSONB
	Output       []byte     `json:"output" db:"output"`       // JSONB
	ErrorMessage *string    `json:"error_message,omitempty" db:"error_message"`
	StartedAt    *time.Time `json:"started_at,omitempty" db:"started_at"`
	CompletedAt  *time.Time `json:"completed_at,omitempty" db:"completed_at"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`
}

// Approval represents a human approval gate in a workflow.
type Approval struct {
	ID             uuid.UUID  `json:"id" db:"id"`
	WorkflowID     uuid.UUID  `json:"workflow_id" db:"workflow_id"`
	WorkflowStepID *uuid.UUID `json:"workflow_step_id,omitempty" db:"workflow_step_id"`
	RequestedBy    string     `json:"requested_by" db:"requested_by"`
	AssignedTo     string     `json:"assigned_to" db:"assigned_to"`
	Status         string     `json:"status" db:"status"` // pending, approved, rejected, expired
	DecisionReason *string    `json:"decision_reason,omitempty" db:"decision_reason"`
	DecidedAt      *time.Time `json:"decided_at,omitempty" db:"decided_at"`
	ExpiresAt      *time.Time `json:"expires_at,omitempty" db:"expires_at"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at" db:"updated_at"`
}

// AuditLog records all significant actions for compliance and debugging.
type AuditLog struct {
	ID           uuid.UUID  `json:"id" db:"id"`
	TeamID       *uuid.UUID `json:"team_id,omitempty" db:"team_id"`
	Actor        string     `json:"actor" db:"actor"`
	Action       string     `json:"action" db:"action"`
	ResourceType string     `json:"resource_type" db:"resource_type"`
	ResourceID   *uuid.UUID `json:"resource_id,omitempty" db:"resource_id"`
	Details      []byte     `json:"details" db:"details"` // JSONB
	IPAddress    *string    `json:"ip_address,omitempty" db:"ip_address"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
}

// DriftRecord captures detected infrastructure drift.
type DriftRecord struct {
	ID             uuid.UUID  `json:"id" db:"id"`
	EnvironmentID  uuid.UUID  `json:"environment_id" db:"environment_id"`
	WorkflowID     *uuid.UUID `json:"workflow_id,omitempty" db:"workflow_id"`
	ResourceType   string     `json:"resource_type" db:"resource_type"`
	ResourceID     string     `json:"resource_id" db:"resource_id"`
	ExpectedState  []byte     `json:"expected_state" db:"expected_state"` // JSONB
	ActualState    []byte     `json:"actual_state" db:"actual_state"`     // JSONB
	DriftDetectedAt time.Time `json:"drift_detected_at" db:"drift_detected_at"`
	ResolvedAt     *time.Time `json:"resolved_at,omitempty" db:"resolved_at"`
	Resolution     *string    `json:"resolution,omitempty" db:"resolution"` // auto_remediated, manual_fix, accepted, ignored
	Severity       string     `json:"severity" db:"severity"`               // low, medium, high, critical
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at" db:"updated_at"`
}
