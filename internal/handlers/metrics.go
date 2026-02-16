package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/PratikDhanave/event-analytics-service/internal/auth"
	"github.com/PratikDhanave/event-analytics-service/internal/store"
)

// RegisterMetricRoutes registers the serving-path endpoint.
//
// GET /metrics?event_name=...&from=...&to=...
// - Requires X-API-Key (tenant context)
// - Returns count for the window [from,to)
func RegisterMetricRoutes(r gin.IRoutes, st *store.PostgresStore) {
	r.GET("/metrics", func(c *gin.Context) {
		tenantID := auth.TenantID(c)
		if tenantID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		eventName := c.Query("event_name")
		fromStr := c.Query("from")
		toStr := c.Query("to")

		// Required query params per contract.
		if eventName == "" || fromStr == "" || toStr == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "event_name, from, to are required"})
			return
		}

		from, err := time.Parse(time.RFC3339, fromStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "from must be RFC3339"})
			return
		}
		to, err := time.Parse(time.RFC3339, toStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "to must be RFC3339"})
			return
		}

		from = from.UTC()
		to = to.UTC()

		// Validate window to avoid confusing results.
		if !from.Before(to) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "from must be < to"})
			return
		}

		count, err := st.CountEvents(c.Request.Context(), tenantID, eventName, from, to)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "db query failed"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"event_name": eventName,
			"count":      count,
		})
	})
}
