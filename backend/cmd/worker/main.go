package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"go.temporal.io/sdk/worker"

	"github.com/infraforge/infraforge/internal/config"
	"github.com/infraforge/infraforge/internal/db"
	infraTemporal "github.com/infraforge/infraforge/internal/temporal"
)

func main() {
	log.Println("Starting InfraForge Temporal worker...")

	cfg := config.Load()

	// Connect to PostgreSQL (activities need DB access)
	pool, err := db.Connect(&cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()
	log.Println("Connected to PostgreSQL")

	// Connect to Temporal
	temporalClient, err := infraTemporal.NewClient(&cfg.Temporal)
	if err != nil {
		log.Fatalf("Failed to connect to Temporal: %v", err)
	}
	defer temporalClient.Close()
	log.Println("Connected to Temporal")

	// Create activities
	activities := infraTemporal.NewActivities(pool)

	// Create worker
	w := worker.New(temporalClient, infraTemporal.TaskQueue, worker.Options{
		MaxConcurrentActivityExecutionSize:     10,
		MaxConcurrentWorkflowTaskExecutionSize: 10,
	})

	// Register workflows
	w.RegisterWorkflow(infraTemporal.ProvisionWorkflow)
	w.RegisterWorkflow(infraTemporal.UpdateWorkflow)
	w.RegisterWorkflow(infraTemporal.DecommissionWorkflow)

	// Register activities
	w.RegisterActivity(activities.MarkWorkflowRunning)
	w.RegisterActivity(activities.MarkWorkflowFailed)
	w.RegisterActivity(activities.ValidateEnvironment)
	w.RegisterActivity(activities.TerraformPlan)
	w.RegisterActivity(activities.RecordApprovalGate)
	w.RegisterActivity(activities.CreateNetwork)
	w.RegisterActivity(activities.ProvisionCompute)
	w.RegisterActivity(activities.ConfigureDNS)
	w.RegisterActivity(activities.DeployBaseServices)
	w.RegisterActivity(activities.RunHealthChecks)
	w.RegisterActivity(activities.RegisterInCatalog)
	w.RegisterActivity(activities.FinalizeProvision)
	w.RegisterActivity(activities.ValidateNewConfig)
	w.RegisterActivity(activities.GenerateChangePlan)
	w.RegisterActivity(activities.ApplyComputeChanges)
	w.RegisterActivity(activities.UpdateDNSAndLB)
	w.RegisterActivity(activities.FinalizeUpdate)
	w.RegisterActivity(activities.DrainTraffic)
	w.RegisterActivity(activities.DeregisterServices)
	w.RegisterActivity(activities.TerminateCompute)
	w.RegisterActivity(activities.RemoveDNS)
	w.RegisterActivity(activities.DeleteNetwork)
	w.RegisterActivity(activities.CleanupState)

	// Start worker
	go func() {
		if err := w.Run(worker.InterruptCh()); err != nil {
			log.Fatalf("Worker failed: %v", err)
		}
	}()

	log.Printf("Worker listening on task queue: %s", infraTemporal.TaskQueue)

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down worker...")
	w.Stop()
	log.Println("Worker stopped")
}
