CREATE TABLE IF NOT EXISTS appointments (
    id UUID PRIMARY KEY,
    clinic_id UUID NOT NULL REFERENCES clinics(id),
    patient_id UUID NOT NULL REFERENCES users(id),
    clinician_id UUID NOT NULL REFERENCES users(id),
    staff_id UUID NOT NULL REFERENCES users(id),
    service_id UUID NOT NULL REFERENCES services(id),
    appointment_type VARCHAR(50) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'scheduled',
    start_time TIMESTAMP WITH TIME ZONE NOT NULL,
    end_time TIMESTAMP WITH TIME ZONE NOT NULL,
    notes TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT appointments_time_check CHECK (end_time > start_time)
);

CREATE INDEX idx_appointments_clinic ON appointments(clinic_id);
CREATE INDEX idx_appointments_patient ON appointments(patient_id);
CREATE INDEX idx_appointments_clinician ON appointments(clinician_id);
CREATE INDEX idx_appointments_time ON appointments(start_time, end_time); 