package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/PratikDhanave/event-analytics-service/internal/auth"
	"github.com/PratikDhanave/event-analytics-service/internal/models"
	"github.com/PratikDhanave/event-analytics-service/internal/store"
)

// parseRFC3339 parses an RFC3339 timestamp and normalizes it to UTC.
func parseRFC3339(ts string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return time.Time{}, err
	}
	return t.UTC(), nil
}

// RegisterEventRoutes registers the ingestion-path endpoint.
//
// POST /events
// - Requires X-API-Key (tenant context)
// - Durable: returns success only after DB write completes
// - Idempotent: duplicates detected via (tenant_id, event_id) uniqueness
func RegisterEventRoutes(r gin.IRoutes, st *store.PostgresStore) {
	r.POST("/events", func(c *gin.Context) {
		tenantID := auth.TenantID(c)
		if tenantID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		var req models.EventIngestRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON payload"})
			return
		}

		// Required fields per contract.
		if req.EventName == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "event_name required"})
			return
		}
		if req.Timestamp == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "timestamp required"})
			return
		}

		ts, err := parseRFC3339(req.Timestamp)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "timestamp must be RFC3339"})
			return
		}

		// Idempotency precedence:
		// 1) Idempotency-Key header (recommended for retries)
		// 2) event_id in payload
		// 3) generated UUID (fallback; cannot dedupe client retries)
		eventID := c.GetHeader("Idempotency-Key")
		if eventID == "" {
			eventID = req.EventID
		}
		if eventID == "" {
			eventID = uuid.New().String()
		}

		inserted, err := st.InsertEvent(
			c.Request.Context(),
			tenantID,
			eventID,
			req.EventName,
			ts,
			req.Properties,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "db insert failed"})
			return
		}

		// 201 for new events, 200 for duplicates (idempotent success).
		status := http.StatusCreated
		dup := false
		if !inserted {
			status = http.StatusOK
			dup = true
		}

		c.JSON(status, models.EventIngestResponse{
			EventID:   eventID,
			Duplicate: dup,
		})
	})
}
