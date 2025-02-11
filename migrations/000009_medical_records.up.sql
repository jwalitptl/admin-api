CREATE TABLE medical_records (
    id UUID PRIMARY KEY,
    patient_id UUID REFERENCES patients(id),
    organization_id UUID REFERENCES organizations(id),
    type VARCHAR(50) NOT NULL,
    description TEXT,
    diagnosis JSONB,
    treatment JSONB,
    medications JSONB,
    attachments JSONB,
    access_level VARCHAR(20) NOT NULL,
    created_by UUID REFERENCES users(id),
    last_accessed_by UUID REFERENCES users(id),
    last_accessed_at TIMESTAMP,
    region_code VARCHAR(10),
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    deleted_at TIMESTAMP,
    CONSTRAINT chk_record_access_level CHECK (access_level IN ('public', 'private', 'restricted'))
);

CREATE INDEX idx_medical_records_patient ON medical_records(patient_id);
CREATE INDEX idx_medical_records_type ON medical_records(type);
CREATE INDEX idx_medical_records_created_by ON medical_records(created_by); 