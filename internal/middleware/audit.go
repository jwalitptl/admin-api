package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jwalitptl/admin-api/internal/service/audit"
)

type AuditMiddleware struct {
	auditSvc *audit.Service
}

func NewAuditMiddleware(auditSvc *audit.Service) *AuditMiddleware {
	return &AuditMiddleware{auditSvc: auditSvc}
}

func (m *AuditMiddleware) AuditLog(entityType string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		orgID, _ := c.Get("organization_id")

		// Execute the handler
		c.Next()

		// Log after the handler has executed
		action := "read"
		switch c.Request.Method {
		case "POST":
			action = "create"
		case "PUT", "PATCH":
			action = "update"
		case "DELETE":
			action = "delete"
		}

		// Get the entity ID from the URL parameter if it exists
		var entityID uuid.UUID
		if id := c.Param("id"); id != "" {
			entityID, _ = uuid.Parse(id)
		}

		m.auditSvc.Log(c,
			userID.(uuid.UUID),
			orgID.(uuid.UUID),
			action,
			entityType,
			entityID,
			&audit.LogOptions{
				Metadata: map[string]interface{}{
					"path":   c.Request.URL.Path,
					"method": c.Request.Method,
					"status": c.Writer.Status(),
				},
			},
		)
	}
}
