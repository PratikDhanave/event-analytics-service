package auth

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// tenantCtxKey is the Gin context key used to store the authenticated tenant ID.
const tenantCtxKey = "tenant_id"

// APIKeyMiddleware enforces multi-tenancy by mapping X-API-Key → tenantID.
// In production this mapping would typically come from IAM/JWT/Secret Manager.
func APIKeyMiddleware(keys map[string]string) gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := strings.TrimSpace(c.GetHeader("X-API-Key"))
		tenantID, ok := keys[apiKey]
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		c.Set(tenantCtxKey, tenantID)
		c.Next()
	}
}

// TenantID returns the authenticated tenant ID from the request context.
func TenantID(c *gin.Context) string {
	v, _ := c.Get(tenantCtxKey)
	s, _ := v.(string)
	return s
}
