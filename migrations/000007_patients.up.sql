CREATE TABLE patients (
    id UUID PRIMARY KEY,
    user_id UUID REFERENCES users(id),
    clinic_id UUID REFERENCES clinics(id),
    organization_id UUID REFERENCES organizations(id),
    first_name VARCHAR(255) NOT NULL,
    last_name VARCHAR(255) NOT NULL,
    email VARCHAR(255),
    phone VARCHAR(50),
    date_of_birth DATE NOT NULL,
    gender VARCHAR(20),
    address TEXT,
    emergency_contact JSONB,
    insurance_info JSONB,
    status VARCHAR(50) NOT NULL,
    region_code VARCHAR(10),
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    deleted_at TIMESTAMP,
    CONSTRAINT chk_patient_status CHECK (status IN ('active', 'inactive', 'archived')),
    CONSTRAINT chk_patient_gender CHECK (gender IN ('male', 'female', 'other', 'prefer_not_to_say'))
);

CREATE INDEX idx_patients_clinic ON patients(clinic_id);
CREATE INDEX idx_patients_email ON patients(email);
CREATE INDEX idx_patients_name ON patients(last_name, first_name); 