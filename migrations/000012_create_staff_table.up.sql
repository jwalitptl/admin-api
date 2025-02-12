CREATE TABLE IF NOT EXISTS staff (
    id UUID PRIMARY KEY,
    clinic_id UUID NOT NULL REFERENCES clinics(id),
    user_id UUID NOT NULL REFERENCES users(id),
    role VARCHAR(50) NOT NULL,
    status VARCHAR(50) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE,
    UNIQUE(clinic_id, user_id)
);

CREATE INDEX idx_staff_clinic_id ON staff(clinic_id);
CREATE INDEX idx_staff_user_id ON staff(user_id); 