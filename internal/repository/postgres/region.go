package postgres

import (
	"context"
	"fmt"

	"github.com/jwalitptl/admin-api/internal/model"
	"github.com/jwalitptl/admin-api/internal/repository"
)

type regionRepository struct {
	BaseRepository
}

func NewRegionRepository(base BaseRepository) repository.RegionRepository {
	return &regionRepository{base}
}

func (r *regionRepository) GetRegion(ctx context.Context, code string) (*model.Region, error) {
	query := `
		SELECT * FROM regions 
		WHERE code = $1 AND deleted_at IS NULL
	`

	var region model.Region
	if err := r.GetDB().GetContext(ctx, &region, query, code); err != nil {
		return nil, fmt.Errorf("failed to get region: %w", err)
	}

	// Get features for the region
	featuresQuery := `
		SELECT feature_key, is_enabled 
		FROM region_features 
		WHERE region_id = $1
	`
	rows, err := r.GetDB().QueryContext(ctx, featuresQuery, region.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get region features: %w", err)
	}
	defer rows.Close()

	region.Features = make(map[string]bool)
	for rows.Next() {
		var key string
		var enabled bool
		if err := rows.Scan(&key, &enabled); err != nil {
			return nil, err
		}
		region.Features[key] = enabled
	}

	return &region, nil
}

func (r *regionRepository) ListRegions(ctx context.Context) ([]*model.Region, error) {
	query := `
		SELECT * FROM regions 
		WHERE deleted_at IS NULL 
		ORDER BY name
	`

	var regions []*model.Region
	if err := r.GetDB().SelectContext(ctx, &regions, query); err != nil {
		return nil, fmt.Errorf("failed to list regions: %w", err)
	}

	// Get features for all regions
	for _, region := range regions {
		featuresQuery := `
			SELECT feature_key, is_enabled 
			FROM region_features 
			WHERE region_id = $1
		`
		rows, err := r.GetDB().QueryContext(ctx, featuresQuery, region.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get region features: %w", err)
		}
		defer rows.Close()

		region.Features = make(map[string]bool)
		for rows.Next() {
			var key string
			var enabled bool
			if err := rows.Scan(&key, &enabled); err != nil {
				return nil, err
			}
			region.Features[key] = enabled
		}
	}

	return regions, nil
}

func (r *regionRepository) GetRegionCodeForCountry(ctx context.Context, countryCode string) (string, error) {
	query := `
		SELECT r.code 
		FROM regions r
		JOIN region_countries rc ON r.id = rc.region_id
		WHERE rc.country_code = $1 AND r.deleted_at IS NULL
	`

	var regionCode string
	if err := r.GetDB().GetContext(ctx, &regionCode, query, countryCode); err != nil {
		return "", fmt.Errorf("failed to get region code for country: %w", err)
	}

	return regionCode, nil
}

func (r *regionRepository) UpdateRegion(ctx context.Context, region *model.Region) error {
	tx, err := r.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
		UPDATE regions SET
			name = $1,
			locale = $2,
			timezone = $3,
			date_format = $4,
			currency_code = $5,
			data_retention_days = $6,
			gdpr_enabled = $7,
			hipaa_enabled = $8,
			ccpa_enabled = $9,
			updated_at = NOW()
		WHERE id = $10
	`

	_, err = tx.ExecContext(ctx, query,
		region.Name,
		region.Locale,
		region.TimeZone,
		region.DateFormat,
		region.CurrencyCode,
		region.DataRetentionDays,
		region.GDPR,
		region.HIPAA,
		region.CCPA,
		region.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update region: %w", err)
	}

	// Update features
	_, err = tx.ExecContext(ctx, "DELETE FROM region_features WHERE region_id = $1", region.ID)
	if err != nil {
		return fmt.Errorf("failed to delete old features: %w", err)
	}

	for feature, enabled := range region.Features {
		_, err = tx.ExecContext(ctx,
			"INSERT INTO region_features (region_id, feature_key, is_enabled) VALUES ($1, $2, $3)",
			region.ID, feature, enabled,
		)
		if err != nil {
			return fmt.Errorf("failed to insert feature: %w", err)
		}
	}

	return tx.Commit()
}
