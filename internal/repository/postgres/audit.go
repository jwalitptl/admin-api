package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/jwalitptl/admin-api/internal/model"
	"github.com/jwalitptl/admin-api/internal/repository"
)

type auditRepository struct {
	BaseRepository
}

func NewAuditRepository(base BaseRepository) repository.AuditRepository {
	return &auditRepository{base}
}

func (r *auditRepository) Create(ctx context.Context, log *model.AuditLog) error {
	query := `
        INSERT INTO audit_logs (
            id, user_id, organization_id, action, entity_type, entity_id,
            changes, metadata, ip_address, user_agent, region_code, created_at
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
    `

	return r.WithTx(ctx, func(tx *sqlx.Tx) error {
		_, err := tx.ExecContext(ctx, query,
			log.ID,
			log.UserID,
			log.OrganizationID,
			log.Action,
			log.EntityType,
			log.EntityID,
			log.Changes,
			log.Metadata,
			log.IPAddress,
			log.UserAgent,
			r.GetRegionFromContext(ctx),
			log.CreatedAt,
		)
		return err
	})
}

func (r *auditRepository) List(ctx context.Context, filters map[string]interface{}) ([]*model.AuditLog, error) {
	query := `
        SELECT * FROM audit_logs WHERE 1=1
    `
	var args []interface{}

	if v, ok := filters["user_id"]; ok {
		query += fmt.Sprintf(" AND user_id = $%d", len(args)+1)
		args = append(args, v)
	}

	if v, ok := filters["organization_id"]; ok {
		query += fmt.Sprintf(" AND organization_id = $%d", len(args)+1)
		args = append(args, v)
	}

	query = r.AddRegionFilter(query, r.GetRegionFromContext(ctx))
	query += " ORDER BY created_at DESC"

	var logs []*model.AuditLog
	if err := r.GetDB().SelectContext(ctx, &logs, query, args...); err != nil {
		return nil, fmt.Errorf("failed to list audit logs: %w", err)
	}

	return logs, nil
}

func (r *auditRepository) Cleanup(ctx context.Context, before time.Time) (int64, error) {
	query := `
        DELETE FROM audit_logs
        WHERE created_at < $1
    `

	result, err := r.GetDB().ExecContext(ctx, query, before)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup audit logs: %w", err)
	}

	return result.RowsAffected()
}

func (r *auditRepository) ListWithPagination(ctx context.Context, filters map[string]interface{}) ([]*model.AuditLog, int64, error) {
	// Build base query
	baseQuery := `FROM audit_logs WHERE 1=1`
	var conditions []string
	var args []interface{}
	var queryArgs []interface{}

	// Add filters
	if v, ok := filters["user_id"]; ok {
		args = append(args, v)
		conditions = append(conditions, fmt.Sprintf("user_id = $%d", len(args)))
	}

	if v, ok := filters["organization_id"]; ok {
		args = append(args, v)
		conditions = append(conditions, fmt.Sprintf("organization_id = $%d", len(args)))
	}

	if v, ok := filters["start_date"]; ok {
		args = append(args, v)
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", len(args)))
	}

	if v, ok := filters["end_date"]; ok {
		args = append(args, v)
		conditions = append(conditions, fmt.Sprintf("created_at <= $%d", len(args)))
	}

	if v, ok := filters["action"]; ok {
		args = append(args, v)
		conditions = append(conditions, fmt.Sprintf("action = $%d", len(args)))
	}

	// Add conditions to base query
	for _, condition := range conditions {
		baseQuery += " AND " + condition
	}
	queryArgs = append(queryArgs, args...)

	// Get total count
	countQuery := "SELECT COUNT(*) " + baseQuery
	var total int64
	if err := r.GetDB().GetContext(ctx, &total, countQuery, queryArgs...); err != nil {
		return nil, 0, fmt.Errorf("failed to get total count: %w", err)
	}

	// Add pagination
	limit := filters["limit"].(int)
	offset := filters["offset"].(int)
	args = append(args, limit, offset)
	query := "SELECT * " + baseQuery + fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", len(args)-1, len(args))
	queryArgs = append(queryArgs, limit, offset)

	var logs []*model.AuditLog
	if err := r.GetDB().SelectContext(ctx, &logs, query, queryArgs...); err != nil {
		return nil, 0, fmt.Errorf("failed to list audit logs: %w", err)
	}

	return logs, total, nil
}

func (r *auditRepository) GetAggregateStats(ctx context.Context, filters map[string]interface{}) (*model.AggregateStats, error) {
	var args []interface{}
	whereClause := "WHERE 1=1"

	if v, ok := filters["start_date"]; ok {
		args = append(args, v)
		whereClause += fmt.Sprintf(" AND created_at >= $%d", len(args))
	}

	if v, ok := filters["end_date"]; ok {
		args = append(args, v)
		whereClause += fmt.Sprintf(" AND created_at <= $%d", len(args))
	}

	if v, ok := filters["organization_id"]; ok {
		args = append(args, v)
		whereClause += fmt.Sprintf(" AND organization_id = $%d", len(args))
	}

	stats := &model.AggregateStats{
		ActionCounts:   make(map[string]int),
		EntityCounts:   make(map[string]int),
		UserActivity:   make(map[string]int),
		HourlyActivity: make(map[int]int),
	}

	// Get total logs
	countQuery := "SELECT COUNT(*) FROM audit_logs " + whereClause
	if err := r.GetDB().GetContext(ctx, &stats.TotalLogs, countQuery, args...); err != nil {
		return nil, fmt.Errorf("failed to get total count: %w", err)
	}

	// Get action counts
	actionQuery := `
        SELECT action, COUNT(*) as count 
        FROM audit_logs ` + whereClause + `
        GROUP BY action
    `
	rows, err := r.GetDB().QueryContext(ctx, actionQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get action counts: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var action string
		var count int
		if err := rows.Scan(&action, &count); err != nil {
			return nil, err
		}
		stats.ActionCounts[action] = count
	}

	// Similar queries for entity counts, user activity, and hourly activity...

	// Get top IPs
	ipQuery := `
        SELECT ip_address, COUNT(*) as count 
        FROM audit_logs ` + whereClause + `
        GROUP BY ip_address 
        ORDER BY count DESC 
        LIMIT 10
    `
	rows, err = r.GetDB().QueryContext(ctx, ipQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get top IPs: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var ipCount model.IPActivityCount
		if err := rows.Scan(&ipCount.IPAddress, &ipCount.Count); err != nil {
			return nil, err
		}
		stats.TopIPs = append(stats.TopIPs, ipCount)
	}

	return stats, nil
}

func (r *auditRepository) DeleteBefore(ctx context.Context, cutoff time.Time) error {
	query := `DELETE FROM audit_logs WHERE created_at < $1`
	_, err := r.GetDB().ExecContext(ctx, query, cutoff)
	return err
}
