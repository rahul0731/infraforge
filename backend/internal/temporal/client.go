package temporal

import (
	"context"
	"fmt"

	"go.temporal.io/sdk/client"

	"github.com/infraforge/infraforge/internal/config"
)

// NewClient creates a Temporal SDK client.
func NewClient(cfg *config.TemporalConfig) (client.Client, error) {
	c, err := client.Dial(client.Options{
		HostPort:  cfg.Address(),
		Namespace: cfg.Namespace,
	})
	if err != nil {
		return nil, fmt.Errorf("connecting to temporal: %w", err)
	}
	return c, nil
}

// StartProvisionWorkflow starts a provision workflow.
func StartProvisionWorkflow(ctx context.Context, c client.Client, params WorkflowParams) (string, error) {
	opts := client.StartWorkflowOptions{
		ID:        fmt.Sprintf("provision-%s", params.WorkflowID),
		TaskQueue: TaskQueue,
	}

	run, err := c.ExecuteWorkflow(ctx, opts, ProvisionWorkflow, params)
	if err != nil {
		return "", fmt.Errorf("starting provision workflow: %w", err)
	}
	return run.GetRunID(), nil
}

// StartUpdateWorkflow starts an update workflow.
func StartUpdateWorkflow(ctx context.Context, c client.Client, params WorkflowParams) (string, error) {
	opts := client.StartWorkflowOptions{
		ID:        fmt.Sprintf("update-%s", params.WorkflowID),
		TaskQueue: TaskQueue,
	}

	run, err := c.ExecuteWorkflow(ctx, opts, UpdateWorkflow, params)
	if err != nil {
		return "", fmt.Errorf("starting update workflow: %w", err)
	}
	return run.GetRunID(), nil
}

// StartDecommissionWorkflow starts a decommission workflow.
func StartDecommissionWorkflow(ctx context.Context, c client.Client, params WorkflowParams) (string, error) {
	opts := client.StartWorkflowOptions{
		ID:        fmt.Sprintf("decommission-%s", params.WorkflowID),
		TaskQueue: TaskQueue,
	}

	run, err := c.ExecuteWorkflow(ctx, opts, DecommissionWorkflow, params)
	if err != nil {
		return "", fmt.Errorf("starting decommission workflow: %w", err)
	}
	return run.GetRunID(), nil
}

// SendApprovalSignal sends an approval/rejection signal to a waiting workflow.
func SendApprovalSignal(ctx context.Context, c client.Client, temporalWorkflowID string, signal ApprovalSignal) error {
	return c.SignalWorkflow(ctx, temporalWorkflowID, "", ApprovalSignalChannel, signal)
}

// TerminateWorkflow terminates a running workflow.
func TerminateWorkflow(ctx context.Context, c client.Client, temporalWorkflowID, reason string) error {
	return c.TerminateWorkflow(ctx, temporalWorkflowID, "", reason)
}
