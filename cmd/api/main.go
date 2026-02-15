package main

import (
	"log"

	// local project packages
	"github.com/PratikDhanave/event-analytics-service/internal/config"
	"github.com/PratikDhanave/event-analytics-service/internal/httpserver"
	"github.com/PratikDhanave/event-analytics-service/internal/store"
)

func main() {

	// Load runtime configuration (DB connection etc.)
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	// Create Postgres connection pool.
	// pgxpool automatically manages connection reuse.
	db, err := store.NewPostgresStore(cfg.DBURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Ensure tables/indexes exist.
	// This allows "docker compose up" to fully bootstrap the service.
	if err := db.EnsureSchema(); err != nil {
		log.Fatal(err)
	}

	// Build HTTP router and inject DB dependency.
	router := httpserver.NewRouter(db)

	log.Println("server started on :8080")

	// Start HTTP server (blocking call).
	if err := router.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
