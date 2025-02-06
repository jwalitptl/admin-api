-- Drop the type if it exists
DROP TYPE IF EXISTS appointment_status;

-- Create the enum type
CREATE TYPE appointment_status AS ENUM ('scheduled', 'confirmed', 'cancelled', 'completed');

CREATE TABLE appointments (
    id UUID PRIMARY KEY,
    clinic_id UUID NOT NULL REFERENCES clinics(id),
    clinician_id UUID NOT NULL REFERENCES clinicians(id),
    patient_id UUID NOT NULL REFERENCES patients(id),
    start_time TIMESTAMP WITH TIME ZONE NOT NULL,
    end_time TIMESTAMP WITH TIME ZONE NOT NULL,
    status appointment_status NOT NULL DEFAULT 'scheduled',
    notes TEXT,
    cancel_reason TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- Add indexes for common queries
    CONSTRAINT appointments_time_check CHECK (end_time > start_time)
);

CREATE INDEX idx_appointments_clinic_id ON appointments(clinic_id);
CREATE INDEX idx_appointments_clinician_id ON appointments(clinician_id);
CREATE INDEX idx_appointments_patient_id ON appointments(patient_id);
CREATE INDEX idx_appointments_start_time ON appointments(start_time);
CREATE INDEX idx_appointments_status ON appointments(status); 