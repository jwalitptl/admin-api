CREATE TABLE regions (
    id UUID PRIMARY KEY,
    code VARCHAR(10) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    locale VARCHAR(20) NOT NULL,
    timezone VARCHAR(50) NOT NULL,
    date_format VARCHAR(50) NOT NULL,
    currency_code VARCHAR(3) NOT NULL,
    data_retention_days INTEGER NOT NULL DEFAULT 365,
    gdpr_enabled BOOLEAN NOT NULL DEFAULT false,
    hipaa_enabled BOOLEAN NOT NULL DEFAULT false,
    ccpa_enabled BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);

CREATE TABLE region_features (
    region_id UUID REFERENCES regions(id),
    feature_key VARCHAR(100) NOT NULL,
    is_enabled BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY (region_id, feature_key)
);

CREATE TABLE region_countries (
    region_id UUID REFERENCES regions(id),
    country_code VARCHAR(2) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY (region_id, country_code)
);

-- Create indexes
CREATE INDEX idx_regions_code ON regions(code);
CREATE INDEX idx_region_countries_country ON region_countries(country_code);

-- Insert default global region
INSERT INTO regions (
    id, code, name, locale, timezone, date_format, currency_code
) VALUES (
    '00000000-0000-0000-0000-000000000000',
    'GLOBAL',
    'Global',
    'en-US',
    'UTC',
    'YYYY-MM-DD',
    'USD'
); 