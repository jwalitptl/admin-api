CREATE TABLE regions (
    id UUID PRIMARY KEY,
    code VARCHAR(10) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL,
    config JSONB DEFAULT '{}',
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    deleted_at TIMESTAMP,
    CONSTRAINT chk_region_status CHECK (status IN ('active', 'inactive'))
);

CREATE TABLE region_features (
    id UUID PRIMARY KEY,
    region_id UUID REFERENCES regions(id),
    name VARCHAR(255) NOT NULL,
    enabled BOOLEAN DEFAULT false,
    config JSONB DEFAULT '{}',
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    UNIQUE(region_id, name)
);

CREATE INDEX idx_regions_code ON regions(code);
CREATE INDEX idx_region_features_name ON region_features(name);

-- Insert default region
INSERT INTO regions (id, code, name, status, created_at, updated_at)
VALUES ('00000000-0000-0000-0000-000000000000', 'US', 'United States', 'active', NOW(), NOW()); 