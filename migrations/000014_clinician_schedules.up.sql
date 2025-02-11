CREATE TABLE clinician_schedules (
    id UUID PRIMARY KEY,
    clinician_id UUID REFERENCES users(id),
    clinic_id UUID REFERENCES clinics(id),
    start_time TIMESTAMP NOT NULL,
    end_time TIMESTAMP NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'available',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_schedule_status CHECK (status IN ('available', 'booked', 'blocked')),
    CONSTRAINT chk_schedule_time CHECK (end_time > start_time)
);

CREATE INDEX idx_clinician_schedules_time ON clinician_schedules(start_time, end_time);
CREATE INDEX idx_clinician_schedules_clinician ON clinician_schedules(clinician_id); 