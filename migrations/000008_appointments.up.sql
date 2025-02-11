CREATE TABLE appointments (
    id UUID PRIMARY KEY,
    clinic_id UUID REFERENCES clinics(id),
    patient_id UUID REFERENCES patients(id),
    clinician_id UUID REFERENCES users(id),
    service_id UUID,
    start_time TIMESTAMP NOT NULL,
    end_time TIMESTAMP NOT NULL,
    status VARCHAR(50) NOT NULL,
    notes TEXT,
    cancel_reason TEXT,
    completed_at TIMESTAMP,
    region_code VARCHAR(10),
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    deleted_at TIMESTAMP,
    CONSTRAINT chk_appointment_status CHECK (status IN ('scheduled', 'confirmed', 'cancelled', 'completed', 'no_show')),
    CONSTRAINT chk_appointment_time CHECK (end_time > start_time)
);

CREATE INDEX idx_appointments_clinic ON appointments(clinic_id);
CREATE INDEX idx_appointments_patient ON appointments(patient_id);
CREATE INDEX idx_appointments_clinician ON appointments(clinician_id);
CREATE INDEX idx_appointments_time ON appointments(start_time, end_time); 