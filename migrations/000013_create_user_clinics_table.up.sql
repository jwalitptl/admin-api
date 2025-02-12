CREATE TABLE IF NOT EXISTS user_clinics (
    user_id UUID NOT NULL REFERENCES users(id),
    clinic_id UUID NOT NULL REFERENCES clinics(id),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE,
    PRIMARY KEY (user_id, clinic_id)
);

CREATE INDEX idx_user_clinics_user_id ON user_clinics(user_id);
CREATE INDEX idx_user_clinics_clinic_id ON user_clinics(clinic_id); 