package temporal

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Activities holds shared dependencies for Temporal activities.
type Activities struct {
	pool *pgxpool.Pool
}

// NewActivities creates a new Activities instance.
func NewActivities(pool *pgxpool.Pool) *Activities {
	return &Activities{pool: pool}
}

// StepInput is the input for each workflow step activity.
type StepInput struct {
	WorkflowID    string `json:"workflow_id"`
	StepName      string `json:"step_name"`
	StepOrder     int    `json:"step_order"`
	StepType      string `json:"step_type"`
	EnvironmentID string `json:"environment_id"`
	TeamID        string `json:"team_id"`
	Config        map[string]interface{} `json:"config"`
}

// StepOutput is the result of a workflow step activity.
type StepOutput struct {
	StepID  string                 `json:"step_id"`
	Status  string                 `json:"status"`
	Output  map[string]interface{} `json:"output"`
	Message string                 `json:"message"`
}

// createStep inserts a workflow step and marks it running.
func (a *Activities) createStep(ctx context.Context, input *StepInput) (uuid.UUID, error) {
	wfID, err := uuid.Parse(input.WorkflowID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid workflow_id: %w", err)
	}

	now := time.Now()
	stepID := uuid.New()
	inputJSON, _ := json.Marshal(input.Config)

	_, err = a.pool.Exec(ctx, `
		INSERT INTO workflow_steps (id, workflow_id, name, step_order, step_type, status, input, started_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, 'running', $6, $7, $7, $7)
		ON CONFLICT DO NOTHING`,
		stepID, wfID, input.StepName, input.StepOrder, input.StepType, inputJSON, now,
	)
	if err != nil {
		return uuid.Nil, fmt.Errorf("creating step record: %w", err)
	}
	return stepID, nil
}

// completeStep marks a step as completed with output.
func (a *Activities) completeStep(ctx context.Context, stepID uuid.UUID, output map[string]interface{}) error {
	outputJSON, _ := json.Marshal(output)
	now := time.Now()
	_, err := a.pool.Exec(ctx, `
		UPDATE workflow_steps SET status = 'completed', output = $1, completed_at = $2, updated_at = $2
		WHERE id = $3`, outputJSON, now, stepID,
	)
	return err
}

// failStep marks a step as failed with an error message.
func (a *Activities) failStep(ctx context.Context, stepID uuid.UUID, errMsg string) error {
	now := time.Now()
	_, err := a.pool.Exec(ctx, `
		UPDATE workflow_steps SET status = 'failed', error_message = $1, completed_at = $2, updated_at = $2
		WHERE id = $3`, errMsg, now, stepID,
	)
	return err
}

// updateWorkflowStatus updates the workflow's overall status.
func (a *Activities) updateWorkflowStatus(ctx context.Context, workflowID string, status string) error {
	wfID, _ := uuid.Parse(workflowID)
	now := time.Now()

	var query string
	switch status {
	case "running":
		query = `UPDATE workflows SET status = $1, started_at = $2, updated_at = $2 WHERE id = $3`
	case "completed":
		query = `UPDATE workflows SET status = $1, completed_at = $2, updated_at = $2 WHERE id = $3`
	case "failed":
		query = `UPDATE workflows SET status = $1, completed_at = $2, updated_at = $2 WHERE id = $3`
	default:
		query = `UPDATE workflows SET status = $1, updated_at = $2 WHERE id = $3`
	}

	_, err := a.pool.Exec(ctx, query, status, now, wfID)
	return err
}

// --- Provision Activities ---

func (a *Activities) ValidateEnvironment(ctx context.Context, input *StepInput) (*StepOutput, error) {
	stepID, err := a.createStep(ctx, input)
	if err != nil {
		return nil, err
	}

	// Simulate validation logic
	time.Sleep(500 * time.Millisecond)

	output := map[string]interface{}{
		"validated":    true,
		"environment":  input.EnvironmentID,
		"provider":     input.Config["provider"],
	}
	if err := a.completeStep(ctx, stepID, output); err != nil {
		return nil, err
	}

	return &StepOutput{StepID: stepID.String(), Status: "completed", Output: output, Message: "Environment validated"}, nil
}

func (a *Activities) TerraformPlan(ctx context.Context, input *StepInput) (*StepOutput, error) {
	stepID, err := a.createStep(ctx, input)
	if err != nil {
		return nil, err
	}

	time.Sleep(2 * time.Second) // Simulate terraform plan

	output := map[string]interface{}{
		"resources_to_create": 12,
		"resources_to_modify": 0,
		"resources_to_destroy": 0,
		"plan_file":           fmt.Sprintf("/tmp/plans/%s.tfplan", input.WorkflowID),
	}
	if err := a.completeStep(ctx, stepID, output); err != nil {
		return nil, err
	}

	return &StepOutput{StepID: stepID.String(), Status: "completed", Output: output, Message: "Terraform plan generated"}, nil
}

func (a *Activities) CreateNetwork(ctx context.Context, input *StepInput) (*StepOutput, error) {
	stepID, err := a.createStep(ctx, input)
	if err != nil {
		return nil, err
	}

	time.Sleep(3 * time.Second)

	output := map[string]interface{}{
		"vpc_id":    fmt.Sprintf("vpc-%s", uuid.New().String()[:8]),
		"subnet_ids": []string{"subnet-a", "subnet-b", "subnet-c"},
	}
	if err := a.completeStep(ctx, stepID, output); err != nil {
		return nil, err
	}

	return &StepOutput{StepID: stepID.String(), Status: "completed", Output: output, Message: "Network created"}, nil
}

func (a *Activities) ProvisionCompute(ctx context.Context, input *StepInput) (*StepOutput, error) {
	stepID, err := a.createStep(ctx, input)
	if err != nil {
		return nil, err
	}

	time.Sleep(5 * time.Second)

	output := map[string]interface{}{
		"instance_count": 3,
		"instance_type":  "t3.large",
		"instances":      []string{"i-001", "i-002", "i-003"},
	}
	if err := a.completeStep(ctx, stepID, output); err != nil {
		return nil, err
	}

	return &StepOutput{StepID: stepID.String(), Status: "completed", Output: output, Message: "Compute provisioned"}, nil
}

func (a *Activities) ConfigureDNS(ctx context.Context, input *StepInput) (*StepOutput, error) {
	stepID, err := a.createStep(ctx, input)
	if err != nil {
		return nil, err
	}

	time.Sleep(1 * time.Second)

	output := map[string]interface{}{
		"dns_records_created": 3,
		"domain":             fmt.Sprintf("%s.infra.internal", input.Config["slug"]),
	}
	if err := a.completeStep(ctx, stepID, output); err != nil {
		return nil, err
	}

	return &StepOutput{StepID: stepID.String(), Status: "completed", Output: output, Message: "DNS configured"}, nil
}

func (a *Activities) DeployBaseServices(ctx context.Context, input *StepInput) (*StepOutput, error) {
	stepID, err := a.createStep(ctx, input)
	if err != nil {
		return nil, err
	}

	time.Sleep(4 * time.Second)

	output := map[string]interface{}{
		"services_deployed": []string{"monitoring", "logging", "ingress-controller"},
	}
	if err := a.completeStep(ctx, stepID, output); err != nil {
		return nil, err
	}

	return &StepOutput{StepID: stepID.String(), Status: "completed", Output: output, Message: "Base services deployed"}, nil
}

func (a *Activities) RunHealthChecks(ctx context.Context, input *StepInput) (*StepOutput, error) {
	stepID, err := a.createStep(ctx, input)
	if err != nil {
		return nil, err
	}

	time.Sleep(2 * time.Second)

	output := map[string]interface{}{
		"healthy":     true,
		"checks_passed": 8,
		"checks_total":  8,
	}
	if err := a.completeStep(ctx, stepID, output); err != nil {
		return nil, err
	}

	return &StepOutput{StepID: stepID.String(), Status: "completed", Output: output, Message: "All health checks passed"}, nil
}

func (a *Activities) RegisterInCatalog(ctx context.Context, input *StepInput) (*StepOutput, error) {
	stepID, err := a.createStep(ctx, input)
	if err != nil {
		return nil, err
	}

	time.Sleep(500 * time.Millisecond)

	output := map[string]interface{}{
		"catalog_entry_id": uuid.New().String(),
		"registered":       true,
	}
	if err := a.completeStep(ctx, stepID, output); err != nil {
		return nil, err
	}

	return &StepOutput{StepID: stepID.String(), Status: "completed", Output: output, Message: "Registered in service catalog"}, nil
}

func (a *Activities) FinalizeProvision(ctx context.Context, input *StepInput) (*StepOutput, error) {
	stepID, err := a.createStep(ctx, input)
	if err != nil {
		return nil, err
	}

	// Mark workflow as completed
	if err := a.updateWorkflowStatus(ctx, input.WorkflowID, "completed"); err != nil {
		_ = a.failStep(ctx, stepID, err.Error())
		return nil, err
	}

	output := map[string]interface{}{"finalized": true}
	if err := a.completeStep(ctx, stepID, output); err != nil {
		return nil, err
	}

	return &StepOutput{StepID: stepID.String(), Status: "completed", Output: output, Message: "Provisioning finalized"}, nil
}

// --- Update Activities ---

func (a *Activities) ValidateNewConfig(ctx context.Context, input *StepInput) (*StepOutput, error) {
	stepID, err := a.createStep(ctx, input)
	if err != nil {
		return nil, err
	}

	time.Sleep(1 * time.Second)

	output := map[string]interface{}{
		"config_valid":   true,
		"changes_detected": 4,
	}
	if err := a.completeStep(ctx, stepID, output); err != nil {
		return nil, err
	}

	return &StepOutput{StepID: stepID.String(), Status: "completed", Output: output, Message: "New configuration validated"}, nil
}

func (a *Activities) GenerateChangePlan(ctx context.Context, input *StepInput) (*StepOutput, error) {
	stepID, err := a.createStep(ctx, input)
	if err != nil {
		return nil, err
	}

	time.Sleep(2 * time.Second)

	output := map[string]interface{}{
		"changes": []string{"scale_compute", "update_lb_rules", "modify_security_groups", "update_dns_ttl"},
	}
	if err := a.completeStep(ctx, stepID, output); err != nil {
		return nil, err
	}

	return &StepOutput{StepID: stepID.String(), Status: "completed", Output: output, Message: "Change plan generated"}, nil
}

func (a *Activities) ApplyComputeChanges(ctx context.Context, input *StepInput) (*StepOutput, error) {
	stepID, err := a.createStep(ctx, input)
	if err != nil {
		return nil, err
	}

	time.Sleep(4 * time.Second)

	output := map[string]interface{}{
		"instances_updated": 3,
		"rollback_available": true,
	}
	if err := a.completeStep(ctx, stepID, output); err != nil {
		return nil, err
	}

	return &StepOutput{StepID: stepID.String(), Status: "completed", Output: output, Message: "Compute changes applied"}, nil
}

func (a *Activities) UpdateDNSAndLB(ctx context.Context, input *StepInput) (*StepOutput, error) {
	stepID, err := a.createStep(ctx, input)
	if err != nil {
		return nil, err
	}

	time.Sleep(2 * time.Second)

	output := map[string]interface{}{
		"dns_updated": true,
		"lb_rules_updated": true,
	}
	if err := a.completeStep(ctx, stepID, output); err != nil {
		return nil, err
	}

	return &StepOutput{StepID: stepID.String(), Status: "completed", Output: output, Message: "DNS and load balancer updated"}, nil
}

func (a *Activities) FinalizeUpdate(ctx context.Context, input *StepInput) (*StepOutput, error) {
	stepID, err := a.createStep(ctx, input)
	if err != nil {
		return nil, err
	}

	if err := a.updateWorkflowStatus(ctx, input.WorkflowID, "completed"); err != nil {
		_ = a.failStep(ctx, stepID, err.Error())
		return nil, err
	}

	output := map[string]interface{}{"finalized": true}
	if err := a.completeStep(ctx, stepID, output); err != nil {
		return nil, err
	}

	return &StepOutput{StepID: stepID.String(), Status: "completed", Output: output, Message: "Update finalized"}, nil
}

// --- Decommission Activities ---

func (a *Activities) DrainTraffic(ctx context.Context, input *StepInput) (*StepOutput, error) {
	stepID, err := a.createStep(ctx, input)
	if err != nil {
		return nil, err
	}

	time.Sleep(3 * time.Second)

	output := map[string]interface{}{
		"connections_drained": 42,
		"drain_timeout_sec":   30,
	}
	if err := a.completeStep(ctx, stepID, output); err != nil {
		return nil, err
	}

	return &StepOutput{StepID: stepID.String(), Status: "completed", Output: output, Message: "Traffic drained"}, nil
}

func (a *Activities) DeregisterServices(ctx context.Context, input *StepInput) (*StepOutput, error) {
	stepID, err := a.createStep(ctx, input)
	if err != nil {
		return nil, err
	}

	time.Sleep(1 * time.Second)

	output := map[string]interface{}{
		"services_deregistered": []string{"monitoring", "logging", "ingress-controller"},
	}
	if err := a.completeStep(ctx, stepID, output); err != nil {
		return nil, err
	}

	return &StepOutput{StepID: stepID.String(), Status: "completed", Output: output, Message: "Services deregistered"}, nil
}

func (a *Activities) TerminateCompute(ctx context.Context, input *StepInput) (*StepOutput, error) {
	stepID, err := a.createStep(ctx, input)
	if err != nil {
		return nil, err
	}

	time.Sleep(3 * time.Second)

	output := map[string]interface{}{
		"instances_terminated": 3,
	}
	if err := a.completeStep(ctx, stepID, output); err != nil {
		return nil, err
	}

	return &StepOutput{StepID: stepID.String(), Status: "completed", Output: output, Message: "Compute terminated"}, nil
}

func (a *Activities) RemoveDNS(ctx context.Context, input *StepInput) (*StepOutput, error) {
	stepID, err := a.createStep(ctx, input)
	if err != nil {
		return nil, err
	}

	time.Sleep(1 * time.Second)

	output := map[string]interface{}{
		"dns_records_removed": 3,
	}
	if err := a.completeStep(ctx, stepID, output); err != nil {
		return nil, err
	}

	return &StepOutput{StepID: stepID.String(), Status: "completed", Output: output, Message: "DNS records removed"}, nil
}

func (a *Activities) DeleteNetwork(ctx context.Context, input *StepInput) (*StepOutput, error) {
	stepID, err := a.createStep(ctx, input)
	if err != nil {
		return nil, err
	}

	time.Sleep(2 * time.Second)

	output := map[string]interface{}{
		"vpc_deleted":   true,
		"subnets_deleted": 3,
	}
	if err := a.completeStep(ctx, stepID, output); err != nil {
		return nil, err
	}

	return &StepOutput{StepID: stepID.String(), Status: "completed", Output: output, Message: "Network deleted"}, nil
}

func (a *Activities) CleanupState(ctx context.Context, input *StepInput) (*StepOutput, error) {
	stepID, err := a.createStep(ctx, input)
	if err != nil {
		return nil, err
	}

	// Mark environment as decommissioned
	if input.EnvironmentID != "" {
		envID, _ := uuid.Parse(input.EnvironmentID)
		_, _ = a.pool.Exec(ctx, `UPDATE environments SET status = 'decommissioned', updated_at = NOW() WHERE id = $1`, envID)
	}

	// Mark workflow completed
	if err := a.updateWorkflowStatus(ctx, input.WorkflowID, "completed"); err != nil {
		_ = a.failStep(ctx, stepID, err.Error())
		return nil, err
	}

	time.Sleep(500 * time.Millisecond)

	output := map[string]interface{}{
		"state_cleaned":  true,
		"environment_status": "decommissioned",
	}
	if err := a.completeStep(ctx, stepID, output); err != nil {
		return nil, err
	}

	return &StepOutput{StepID: stepID.String(), Status: "completed", Output: output, Message: "State cleaned up, environment decommissioned"}, nil
}

// --- Approval Gate Activity ---

// RecordApprovalGate creates a step record for the approval gate.
func (a *Activities) RecordApprovalGate(ctx context.Context, input *StepInput) (*StepOutput, error) {
	stepID, err := a.createStep(ctx, input)
	if err != nil {
		return nil, err
	}

	output := map[string]interface{}{
		"waiting_for_approval": true,
		"step_id":             stepID.String(),
	}
	if err := a.completeStep(ctx, stepID, output); err != nil {
		return nil, err
	}

	return &StepOutput{StepID: stepID.String(), Status: "completed", Output: output, Message: "Approval gate recorded, waiting for signal"}, nil
}

// MarkWorkflowRunning marks a workflow as running.
func (a *Activities) MarkWorkflowRunning(ctx context.Context, input *StepInput) (*StepOutput, error) {
	if err := a.updateWorkflowStatus(ctx, input.WorkflowID, "running"); err != nil {
		return nil, err
	}
	return &StepOutput{Status: "completed", Message: "Workflow marked running"}, nil
}
