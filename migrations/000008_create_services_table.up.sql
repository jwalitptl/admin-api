CREATE TABLE IF NOT EXISTS services (
    id UUID PRIMARY KEY,
    clinic_id UUID REFERENCES clinics(id),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    duration INTEGER NOT NULL,
    price DECIMAL(10,2) NOT NULL,
    status VARCHAR(50) NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    CONSTRAINT chk_service_status CHECK (status IN ('active', 'inactive'))
);

CREATE INDEX IF NOT EXISTS idx_services_clinic ON services(clinic_id);
CREATE INDEX IF NOT EXISTS idx_services_status ON services(status); 