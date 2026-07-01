package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.temporal.io/sdk/client"

	"github.com/infraforge/infraforge/internal/api"
	"github.com/infraforge/infraforge/internal/config"
	"github.com/infraforge/infraforge/internal/db"
	"github.com/infraforge/infraforge/internal/drift"
	"github.com/infraforge/infraforge/internal/repository"
	infraTemporal "github.com/infraforge/infraforge/internal/temporal"
)

func main() {
	log.Println("Starting InfraForge server...")

	cfg := config.Load()

	// Connect to PostgreSQL
	pool, err := db.Connect(&cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()
	log.Println("Connected to PostgreSQL")

	// Connect to Temporal (non-fatal if unavailable)
	var temporalClient client.Client
	tc, err := infraTemporal.NewClient(&cfg.Temporal)
	if err != nil {
		log.Printf("WARNING: Temporal unavailable (%v) - workflow features disabled", err)
	} else {
		temporalClient = tc
		defer temporalClient.Close()
		log.Println("Connected to Temporal")
	}

	// Initialize repositories
	teamRepo := repository.NewTeamRepository(pool)
	envRepo := repository.NewEnvironmentRepository(pool)
	workflowRepo := repository.NewWorkflowRepository(pool)
	approvalRepo := repository.NewApprovalRepository(pool)
	driftRepo := repository.NewDriftRepository(pool)
	auditRepo := repository.NewAuditRepository(pool)
	dashboardRepo := repository.NewDashboardRepository(pool)

	// Initialize handlers
	teamHandler := api.NewTeamHandler(teamRepo, auditRepo)
	envHandler := api.NewEnvironmentHandler(envRepo, workflowRepo, driftRepo, auditRepo, temporalClient)
	workflowHandler := api.NewWorkflowHandler(workflowRepo, envRepo, auditRepo, temporalClient)
	approvalHandler := api.NewApprovalHandler(approvalRepo, workflowRepo, auditRepo, temporalClient)
	driftHandler := api.NewDriftHandler(driftRepo, auditRepo)
	auditHandler := api.NewAuditHandler(auditRepo)
	dashboardHandler := api.NewDashboardHandler(dashboardRepo)

	// Set up router
	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		temporalStatus := "disconnected"
		if temporalClient != nil {
			temporalStatus = "connected"
		}
		_, _ = w.Write([]byte(`{"status":"healthy","service":"infraforge","temporal":"` + temporalStatus + `"}`))
	})

	// Register API routes
	teamHandler.Register(mux)
	envHandler.Register(mux)
	workflowHandler.Register(mux)
	approvalHandler.Register(mux)
	driftHandler.Register(mux)
	auditHandler.Register(mux)
	dashboardHandler.Register(mux)

	// Apply middleware
	handler := api.Chain(mux,
		api.CORS("*"),
		api.Logger,
	)

	server := &http.Server{
		Addr:         cfg.Server.Address(),
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	go func() {
		log.Printf("HTTP server listening on %s", cfg.Server.Address())
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// Start drift simulator
	driftSimCtx, driftSimCancel := context.WithCancel(context.Background())
	defer driftSimCancel()
	driftSim := drift.NewSimulator(pool, driftRepo)
	go driftSim.Start(driftSimCtx)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server shutdown error: %v", err)
	}
	log.Println("Server stopped")
}
