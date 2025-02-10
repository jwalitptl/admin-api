package audit

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jwalitptl/admin-api/internal/handler"
	"github.com/jwalitptl/admin-api/internal/service/audit"
)

type Handler struct {
	service audit.AuditService
}

func NewHandler(service audit.AuditService) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	audit := r.Group("/audit")
	{
		audit.GET("/logs", h.ListLogs)
		audit.GET("/logs/:id", h.GetLog)
		audit.GET("/logs/entity/:type/:id", h.GetEntityLogs)
		audit.GET("/logs/user/:id", h.GetUserLogs)
		audit.GET("/export", h.ExportLogs)
		audit.GET("/aggregate", h.GetAggregateStats)
	}
}

func (h *Handler) ListLogs(c *gin.Context) {
	// Implementation for listing audit logs
}

func (h *Handler) GetEntityLogs(c *gin.Context) {
	entityType := c.Param("type")
	entityID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse("invalid entity_id"))
		return
	}

	filters := map[string]interface{}{
		"entity_type": entityType,
		"entity_id":   entityID,
	}

	logs, err := h.service.List(c.Request.Context(), filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, handler.NewErrorResponse(err.Error()))
		return
	}

	c.JSON(http.StatusOK, handler.NewSuccessResponse(logs))
}

func (h *Handler) GetUserLogs(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse("invalid user_id"))
		return
	}

	filters := map[string]interface{}{
		"user_id": userID,
	}

	logs, err := h.service.List(c.Request.Context(), filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, handler.NewErrorResponse(err.Error()))
		return
	}

	c.JSON(http.StatusOK, handler.NewSuccessResponse(logs))
}

func (h *Handler) GetLog(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse("invalid log ID"))
		return
	}

	filters := map[string]interface{}{
		"id": id,
	}

	logs, err := h.service.List(c.Request.Context(), filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, handler.NewErrorResponse(err.Error()))
		return
	}

	if len(logs) == 0 {
		c.JSON(http.StatusNotFound, handler.NewErrorResponse("log not found"))
		return
	}

	c.JSON(http.StatusOK, handler.NewSuccessResponse(logs[0]))
}

func (h *Handler) ExportLogs(c *gin.Context) {
	format := c.DefaultQuery("format", "csv")
	if format != "csv" && format != "json" {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse("unsupported format"))
		return
	}

	filters := make(map[string]interface{})
	// Copy filters from ListLogs...

	logs, err := h.service.List(c.Request.Context(), filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, handler.NewErrorResponse(err.Error()))
		return
	}

	filename := fmt.Sprintf("audit_logs_%s.%s", time.Now().Format("20060102_150405"), format)

	switch format {
	case "csv":
		c.Header("Content-Type", "text/csv")
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
		writer := csv.NewWriter(c.Writer)
		writer.Write([]string{"ID", "User ID", "Organization ID", "Action", "Entity Type", "Entity ID", "Created At"})
		for _, log := range logs {
			writer.Write([]string{
				log.ID.String(),
				log.UserID.String(),
				log.OrganizationID.String(),
				log.Action,
				log.EntityType,
				log.EntityID.String(),
				log.CreatedAt.Format(time.RFC3339),
			})
		}
		writer.Flush()
	case "json":
		c.JSON(http.StatusOK, logs)
	}
}

type AggregateResponse struct {
	TotalLogs      int               `json:"total_logs"`
	ActionCounts   map[string]int    `json:"action_counts"`
	EntityCounts   map[string]int    `json:"entity_counts"`
	UserActivity   map[string]int    `json:"user_activity"`
	HourlyActivity map[int]int       `json:"hourly_activity"`
	TopIPs         []IPActivityCount `json:"top_ips"`
}

type IPActivityCount struct {
	IPAddress string `json:"ip_address"`
	Count     int    `json:"count"`
}

func (h *Handler) GetAggregateStats(c *gin.Context) {
	// Parse time range
	startDate := c.DefaultQuery("start_date", time.Now().AddDate(0, 0, -7).Format(time.RFC3339))
	endDate := c.DefaultQuery("end_date", time.Now().Format(time.RFC3339))

	start, err := time.Parse(time.RFC3339, startDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse("invalid start_date format"))
		return
	}

	end, err := time.Parse(time.RFC3339, endDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse("invalid end_date format"))
		return
	}

	filters := map[string]interface{}{
		"start_date": start,
		"end_date":   end,
	}

	if orgID := c.Query("organization_id"); orgID != "" {
		id, err := uuid.Parse(orgID)
		if err != nil {
			c.JSON(http.StatusBadRequest, handler.NewErrorResponse("invalid organization_id"))
			return
		}
		filters["organization_id"] = id
	}

	stats, err := h.service.GetAggregateStats(c.Request.Context(), filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, handler.NewErrorResponse(err.Error()))
		return
	}

	c.JSON(http.StatusOK, handler.NewSuccessResponse(stats))
}
