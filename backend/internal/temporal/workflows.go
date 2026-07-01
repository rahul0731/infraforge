package temporal

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	TaskQueue             = "infraforge-worker"
	ApprovalSignalChannel = "approval-signal"
)

// WorkflowParams is the input to all InfraForge workflows.
type WorkflowParams struct {
	WorkflowID    string                 `json:"workflow_id"`
	TeamID        string                 `json:"team_id"`
	EnvironmentID string                 `json:"environment_id"`
	Config        map[string]interface{} `json:"config"`
}

// ApprovalSignal is sent to approve or reject a workflow at the approval gate.
type ApprovalSignal struct {
	Approved bool   `json:"approved"`
	Actor    string `json:"actor"`
	Reason   string `json:"reason"`
}

var defaultActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 5 * time.Minute,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    time.Second,
		BackoffCoefficient: 2.0,
		MaximumInterval:    time.Minute,
		MaximumAttempts:    3,
	},
}

// markFailed is a helper that marks the workflow as failed in the DB when an activity errors out.
func markFailed(ctx workflow.Context, params WorkflowParams, err error) error {
	failCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Second,
	})
	failInput := &StepInput{
		WorkflowID: params.WorkflowID,
		Config:     map[string]interface{}{"error_message": err.Error()},
	}
	var result StepOutput
	_ = workflow.ExecuteActivity(failCtx, "MarkWorkflowFailed", failInput).Get(ctx, &result)
	return err
}

// ProvisionWorkflow orchestrates full environment provisioning (10 steps).
func ProvisionWorkflow(ctx workflow.Context, params WorkflowParams) error {
	actCtx := workflow.WithActivityOptions(ctx, defaultActivityOptions)

	makeInput := func(name string, order int, stepType string) *StepInput {
		return &StepInput{
			WorkflowID:    params.WorkflowID,
			StepName:      name,
			StepOrder:     order,
			StepType:      stepType,
			EnvironmentID: params.EnvironmentID,
			TeamID:        params.TeamID,
			Config:        params.Config,
		}
	}

	// Step 0: Mark workflow running
	var markResult StepOutput
	if err := workflow.ExecuteActivity(actCtx, "MarkWorkflowRunning", makeInput("mark_running", 0, "validate")).Get(ctx, &markResult); err != nil {
		return markFailed(ctx, params, err)
	}

	// Step 1: Validate environment
	var result StepOutput
	if err := workflow.ExecuteActivity(actCtx, "ValidateEnvironment", makeInput("validate_environment", 1, "validate")).Get(ctx, &result); err != nil {
		return markFailed(ctx, params, err)
	}

	// Step 2: Terraform plan
	if err := workflow.ExecuteActivity(actCtx, "TerraformPlan", makeInput("terraform_plan", 2, "plan")).Get(ctx, &result); err != nil {
		return markFailed(ctx, params, err)
	}

	// Step 3: Approval gate (pauses for prod environments)
	isProd := params.Config["is_production"] == true || params.Config["tier"] == "production"
	if isProd {
		// Record the approval gate step
		if err := workflow.ExecuteActivity(actCtx, "RecordApprovalGate", makeInput("approval_gate", 3, "approve")).Get(ctx, &result); err != nil {
			return markFailed(ctx, params, err)
		}

		// Wait for approval signal
		signalCh := workflow.GetSignalChannel(ctx, ApprovalSignalChannel)
		var signal ApprovalSignal
		signalCh.Receive(ctx, &signal)

		if !signal.Approved {
			return markFailed(ctx, params, fmt.Errorf("workflow rejected by %s: %s", signal.Actor, signal.Reason))
		}
	}

	// Step 4: Create network
	if err := workflow.ExecuteActivity(actCtx, "CreateNetwork", makeInput("create_network", 4, "apply")).Get(ctx, &result); err != nil {
		return markFailed(ctx, params, err)
	}

	// Step 5: Provision compute
	if err := workflow.ExecuteActivity(actCtx, "ProvisionCompute", makeInput("provision_compute", 5, "apply")).Get(ctx, &result); err != nil {
		return markFailed(ctx, params, err)
	}

	// Step 6: Configure DNS
	if err := workflow.ExecuteActivity(actCtx, "ConfigureDNS", makeInput("configure_dns", 6, "apply")).Get(ctx, &result); err != nil {
		return markFailed(ctx, params, err)
	}

	// Step 7: Deploy base services
	if err := workflow.ExecuteActivity(actCtx, "DeployBaseServices", makeInput("deploy_base_services", 7, "apply")).Get(ctx, &result); err != nil {
		return markFailed(ctx, params, err)
	}

	// Step 8: Health checks
	if err := workflow.ExecuteActivity(actCtx, "RunHealthChecks", makeInput("health_checks", 8, "validate")).Get(ctx, &result); err != nil {
		return markFailed(ctx, params, err)
	}

	// Step 9: Register in catalog
	if err := workflow.ExecuteActivity(actCtx, "RegisterInCatalog", makeInput("register_catalog", 9, "notify")).Get(ctx, &result); err != nil {
		return markFailed(ctx, params, err)
	}

	// Step 10: Finalize
	if err := workflow.ExecuteActivity(actCtx, "FinalizeProvision", makeInput("finalize", 10, "validate")).Get(ctx, &result); err != nil {
		return markFailed(ctx, params, err)
	}

	return nil
}

// UpdateWorkflow orchestrates environment updates (6 steps).
func UpdateWorkflow(ctx workflow.Context, params WorkflowParams) error {
	actCtx := workflow.WithActivityOptions(ctx, defaultActivityOptions)

	makeInput := func(name string, order int, stepType string) *StepInput {
		return &StepInput{
			WorkflowID:    params.WorkflowID,
			StepName:      name,
			StepOrder:     order,
			StepType:      stepType,
			EnvironmentID: params.EnvironmentID,
			TeamID:        params.TeamID,
			Config:        params.Config,
		}
	}

	// Mark running
	var result StepOutput
	if err := workflow.ExecuteActivity(actCtx, "MarkWorkflowRunning", makeInput("mark_running", 0, "validate")).Get(ctx, &result); err != nil {
		return markFailed(ctx, params, err)
	}

	// Step 1: Validate new config
	if err := workflow.ExecuteActivity(actCtx, "ValidateNewConfig", makeInput("validate_config", 1, "validate")).Get(ctx, &result); err != nil {
		return markFailed(ctx, params, err)
	}

	// Step 2: Generate change plan
	if err := workflow.ExecuteActivity(actCtx, "GenerateChangePlan", makeInput("generate_plan", 2, "plan")).Get(ctx, &result); err != nil {
		return markFailed(ctx, params, err)
	}

	// Step 3: Apply compute changes
	if err := workflow.ExecuteActivity(actCtx, "ApplyComputeChanges", makeInput("apply_compute", 3, "apply")).Get(ctx, &result); err != nil {
		return markFailed(ctx, params, err)
	}

	// Step 4: Update DNS/LB
	if err := workflow.ExecuteActivity(actCtx, "UpdateDNSAndLB", makeInput("update_dns_lb", 4, "apply")).Get(ctx, &result); err != nil {
		return markFailed(ctx, params, err)
	}

	// Step 5: Health checks
	if err := workflow.ExecuteActivity(actCtx, "RunHealthChecks", makeInput("health_checks", 5, "validate")).Get(ctx, &result); err != nil {
		return markFailed(ctx, params, err)
	}

	// Step 6: Finalize
	if err := workflow.ExecuteActivity(actCtx, "FinalizeUpdate", makeInput("finalize", 6, "validate")).Get(ctx, &result); err != nil {
		return markFailed(ctx, params, err)
	}

	return nil
}

// DecommissionWorkflow orchestrates environment teardown (6 steps).
func DecommissionWorkflow(ctx workflow.Context, params WorkflowParams) error {
	actCtx := workflow.WithActivityOptions(ctx, defaultActivityOptions)

	makeInput := func(name string, order int, stepType string) *StepInput {
		return &StepInput{
			WorkflowID:    params.WorkflowID,
			StepName:      name,
			StepOrder:     order,
			StepType:      stepType,
			EnvironmentID: params.EnvironmentID,
			TeamID:        params.TeamID,
			Config:        params.Config,
		}
	}

	// Mark running
	var result StepOutput
	if err := workflow.ExecuteActivity(actCtx, "MarkWorkflowRunning", makeInput("mark_running", 0, "validate")).Get(ctx, &result); err != nil {
		return markFailed(ctx, params, err)
	}

	// Step 1: Drain traffic
	if err := workflow.ExecuteActivity(actCtx, "DrainTraffic", makeInput("drain_traffic", 1, "apply")).Get(ctx, &result); err != nil {
		return markFailed(ctx, params, err)
	}

	// Step 2: Deregister services
	if err := workflow.ExecuteActivity(actCtx, "DeregisterServices", makeInput("deregister", 2, "apply")).Get(ctx, &result); err != nil {
		return markFailed(ctx, params, err)
	}

	// Step 3: Terminate compute
	if err := workflow.ExecuteActivity(actCtx, "TerminateCompute", makeInput("terminate_compute", 3, "apply")).Get(ctx, &result); err != nil {
		return markFailed(ctx, params, err)
	}

	// Step 4: Remove DNS
	if err := workflow.ExecuteActivity(actCtx, "RemoveDNS", makeInput("remove_dns", 4, "apply")).Get(ctx, &result); err != nil {
		return markFailed(ctx, params, err)
	}

	// Step 5: Delete network
	if err := workflow.ExecuteActivity(actCtx, "DeleteNetwork", makeInput("delete_network", 5, "apply")).Get(ctx, &result); err != nil {
		return markFailed(ctx, params, err)
	}

	// Step 6: Cleanup state
	if err := workflow.ExecuteActivity(actCtx, "CleanupState", makeInput("cleanup_state", 6, "apply")).Get(ctx, &result); err != nil {
		return markFailed(ctx, params, err)
	}

	return nil
}
