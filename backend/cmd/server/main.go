package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/infraforge/infraforge/internal/api"
	"github.com/infraforge/infraforge/internal/config"
	"github.com/infraforge/infraforge/internal/db"
	"github.com/infraforge/infraforge/internal/repository"
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
	envHandler := api.NewEnvironmentHandler(envRepo, auditRepo)
	workflowHandler := api.NewWorkflowHandler(workflowRepo, auditRepo)
	approvalHandler := api.NewApprovalHandler(approvalRepo, auditRepo)
	driftHandler := api.NewDriftHandler(driftRepo, auditRepo)
	auditHandler := api.NewAuditHandler(auditRepo)
	dashboardHandler := api.NewDashboardHandler(dashboardRepo)

	// Set up router
	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"healthy","service":"infraforge"}`))
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
