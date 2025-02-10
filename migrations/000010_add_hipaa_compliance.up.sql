-- Add encryption and audit fields to medical_records
ALTER TABLE medical_records
ADD COLUMN encrypted_data BYTEA,
ADD COLUMN encryption_key_id UUID,
ADD COLUMN access_level VARCHAR(50),
ADD COLUMN last_accessed_by UUID REFERENCES users(id),
ADD COLUMN last_accessed_at TIMESTAMP WITH TIME ZONE;

-- Create audit log table
CREATE TABLE audit_logs (
    id UUID PRIMARY KEY,
    entity_type VARCHAR(50) NOT NULL,
    entity_id UUID NOT NULL,
    action VARCHAR(50) NOT NULL,
    user_id UUID REFERENCES users(id),
    ip_address VARCHAR(45),
    changes JSONB,
    access_reason TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL
);

-- Add indexes for audit queries
CREATE INDEX idx_audit_entity ON audit_logs(entity_type, entity_id);
CREATE INDEX idx_audit_user ON audit_logs(user_id);
CREATE INDEX idx_audit_created_at ON audit_logs(created_at); 