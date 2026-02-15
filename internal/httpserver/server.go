package httpserver

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Store defines the minimal dependency needed by HTTP layer.
// We depend on an interface, not Postgres directly.
// This keeps layers decoupled and testable.
type Store interface {
	Ping(ctx context.Context) error
}

// NewRouter builds the Gin HTTP router.
func NewRouter(st Store) *gin.Engine {

	// ReleaseMode removes debug logs (production-friendly).
	gin.SetMode(gin.ReleaseMode)

	r := gin.New()

	// Panic recovery middleware.
	r.Use(gin.Recovery())

	// Liveness endpoint.
	// Used by container orchestrators to check the process is alive.
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Readiness endpoint.
	// Verifies DB connectivity so traffic is only routed when dependencies are healthy.
	r.GET("/ready", func(c *gin.Context) {

		ctx, cancel := context.WithTimeout(c.Request.Context(), time.Second)
		defer cancel()

		if err := st.Ping(ctx); err != nil {

			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "not_ready",
				"error":  err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})

	return r
}
