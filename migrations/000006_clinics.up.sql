CREATE TABLE clinics (
    id UUID PRIMARY KEY,
    organization_id UUID REFERENCES organizations(id),
    name VARCHAR(255) NOT NULL,
    location TEXT,
    status VARCHAR(50) NOT NULL,
    region_code VARCHAR(10),
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    deleted_at TIMESTAMP,
    CONSTRAINT chk_clinic_status CHECK (status IN ('active', 'inactive', 'suspended'))
);

CREATE INDEX idx_clinics_organization ON clinics(organization_id);

CREATE TABLE clinic_staff (
    clinic_id UUID REFERENCES clinics(id),
    user_id UUID REFERENCES users(id),
    role VARCHAR(50) NOT NULL,
    created_at TIMESTAMP NOT NULL,
    PRIMARY KEY (clinic_id, user_id)
); 