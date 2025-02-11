CREATE TABLE settings (
    id UUID PRIMARY KEY,
    organization_id UUID REFERENCES organizations(id),
    category VARCHAR(50) NOT NULL,
    key VARCHAR(255) NOT NULL,
    value JSONB NOT NULL,
    region_code VARCHAR(10),
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    UNIQUE(organization_id, category, key)
);

CREATE INDEX idx_settings_org ON settings(organization_id);
CREATE INDEX idx_settings_category ON settings(category); 