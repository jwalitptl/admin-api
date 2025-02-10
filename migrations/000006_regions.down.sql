-- Drop indexes
DROP INDEX IF EXISTS idx_region_countries_country;
DROP INDEX IF EXISTS idx_regions_code;

-- Drop tables in correct order
DROP TABLE IF EXISTS region_countries;
DROP TABLE IF EXISTS region_features;
DROP TABLE IF EXISTS regions; 