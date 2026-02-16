package main

import (
	"log"

	"github.com/PratikDhanave/event-analytics-service/internal/config"
	"github.com/PratikDhanave/event-analytics-service/internal/httpserver"
	"github.com/PratikDhanave/event-analytics-service/internal/store"
)

// main boots the service: config → DB → schema → HTTP server.
func main() {
	// Load runtime config from environment (DB_URL, API_KEYS).
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	// Connect to durable storage (Postgres) using a connection pool.
	db, err := store.NewPostgresStore(cfg.DBURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Ensure required tables/indexes exist so `docker compose up --build` is enough.
	if err := db.EnsureSchema(); err != nil {
		log.Fatal(err)
	}

	// Build HTTP router (public health + authenticated APIs).
	router := httpserver.NewRouter(cfg, db)

	log.Println("server started on :8080")
	log.Fatal(router.Run(":8080"))
}
