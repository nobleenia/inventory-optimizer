// Inventory Optimizer — main entry point.
//
// Modes:
//
//	CLI mode (default):
//	  inventory-optimizer -sales data/sales_history.csv -params data/sku_parameters.csv
//
//	Web mode:
//	  inventory-optimizer -web
//	  inventory-optimizer -web -port :3000
//
//	API mode:
//	  inventory-optimizer -api
//	  inventory-optimizer -api -port :8080
//
// The program loads CSV inputs, computes demand statistics, derives
// optimal inventory policies, runs Monte-Carlo simulations, and
// presents the results via terminal, web browser, or JSON API.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/noble-ch/inventory-optimizer/internal/api"
	"github.com/noble-ch/inventory-optimizer/internal/auth"
	"github.com/noble-ch/inventory-optimizer/internal/engine"
	"github.com/noble-ch/inventory-optimizer/internal/models"
	"github.com/noble-ch/inventory-optimizer/internal/reporting"
	"github.com/noble-ch/inventory-optimizer/internal/store"
	"github.com/noble-ch/inventory-optimizer/internal/web"
)

func main() {
	// ── CLI flags ──────────────────────────────────────────────────────
	webMode := flag.Bool("web", false,
		"Start the web server instead of running CLI analysis")
	apiMode := flag.Bool("api", false,
		"Start the REST API server (requires PostgreSQL)")
	port := flag.String("port", ":8080",
		"Port for the web/api server (used with -web or -api)")
	salesPath := flag.String("sales", "data/sales_history.csv",
		"Path to weekly sales history CSV")
	paramsPath := flag.String("params", "data/sku_parameters.csv",
		"Path to SKU parameters CSV")
	outputPath := flag.String("output", "",
		"Path for CSV export (optional; leave blank to skip)")
	showVersion := flag.Bool("version", false, "Print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Printf("inventory-optimizer v%s\n", models.Version)
		os.Exit(0)
	}

	switch {
	case *apiMode:
		runAPI(*port)
	case *webMode:
		runWeb(*port)
	default:
		runCLI(*salesPath, *paramsPath, *outputPath)
	}
}

// ---------------------------------------------------------------------------
// API mode
// ---------------------------------------------------------------------------

func runAPI(port string) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://inventory:inventory@localhost:5433/inventory?sslmode=disable"
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "dev-secret-change-in-production"
		log.Println("WARNING: Using default JWT secret. Set JWT_SECRET env var in production.")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := store.New(ctx, dsn)
	if err != nil {
		log.Fatalf("Database connection failed: %v\n", err)
	}
	defer db.Close()

	authSvc := auth.NewService(auth.DefaultConfig(jwtSecret))
	apiServer := api.NewServer(db, authSvc)

	srv := &http.Server{
		Addr:         port,
		Handler:      apiServer.Handler(),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Graceful shutdown.
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("API server listening on %s\n", port)
		log.Printf("Docs: docs/openapi.yaml\n")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v\n", err)
		}
	}()

	<-done
	log.Println("Shutting down API server …")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Shutdown error: %v\n", err)
	}
	log.Println("API server stopped.")
}

// ---------------------------------------------------------------------------
// Web mode
// ---------------------------------------------------------------------------

func runWeb(port string) {
	// Try to connect to database for auth + saved reports.
	// If DATABASE_URL is not set or connection fails, run in guest-only mode.
	var db *store.DB
	var authSvc *auth.Service

	dsn := os.Getenv("DATABASE_URL")
	if dsn != "" {
		jwtSecret := os.Getenv("JWT_SECRET")
		if jwtSecret == "" {
			jwtSecret = "dev-secret-change-in-production"
			log.Println("WARNING: Using default JWT secret. Set JWT_SECRET env var in production.")
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		var err error
		db, err = store.New(ctx, dsn)
		if err != nil {
			log.Printf("Database connection failed: %v", err)
			log.Println("Starting web server in guest-only mode (no auth, no saved reports).")
		} else {
			authSvc = auth.NewService(auth.DefaultConfig(jwtSecret))
			log.Println("Database connected — auth and saved reports enabled.")
		}
	} else {
		log.Println("DATABASE_URL not set — starting in guest-only mode.")
	}

	if db != nil {
		defer db.Close()
	}

	server := web.NewServer(port, db, authSvc)
	if err := server.Start(); err != nil {
		log.Fatalf("Server error: %v\n", err)
	}
}

// ---------------------------------------------------------------------------
// CLI mode
// ---------------------------------------------------------------------------

func runCLI(salesPath, paramsPath, outputPath string) {
	start := time.Now()
	opts := engine.DefaultOptions()

	log.Println("Loading data and running analysis …")
	reports, warnings, err := engine.RunFromFiles(salesPath, paramsPath, opts)
	if err != nil {
		log.Fatalf("Analysis failed: %v\n", err)
	}

	for _, w := range warnings {
		log.Printf("  ⚠  %s\n", w)
	}

	reporting.PrintCLI(os.Stdout, reports)

	if outputPath != "" {
		if err := reporting.ExportCSV(outputPath, reports); err != nil {
			log.Fatalf("CSV export failed: %v\n", err)
		}
		log.Printf("CSV report saved to %s\n", outputPath)
	}

	log.Printf("Done in %s.\n", time.Since(start).Round(time.Millisecond))
}
