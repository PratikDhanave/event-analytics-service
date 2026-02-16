package httpserver

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/PratikDhanave/event-analytics-service/internal/auth"
	"github.com/PratikDhanave/event-analytics-service/internal/config"
	"github.com/PratikDhanave/event-analytics-service/internal/handlers"
	"github.com/PratikDhanave/event-analytics-service/internal/store"
)

// NewRouter wires public endpoints and authenticated APIs.
// Public: /health, /ready
// Authenticated: /events, /metrics
func NewRouter(cfg config.Config, st *store.PostgresStore) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	r := gin.New()
	r.Use(gin.Recovery())

	// Liveness: confirms the process is running.
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Readiness: confirms the DB dependency is reachable.
	r.GET("/ready", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), time.Second)
		defer cancel()

		if err := st.Ping(ctx); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not_ready", "error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})

	// Auth group enforces tenant context via X-API-Key.
	authGroup := r.Group("/")
	authGroup.Use(auth.APIKeyMiddleware(cfg.APIKeys))

	handlers.RegisterEventRoutes(authGroup, st)
	handlers.RegisterMetricRoutes(authGroup, st)

	return r
}
