CREATE TABLE rbac_audit_logs (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id),
    action VARCHAR(50) NOT NULL,
    entity_type VARCHAR(50) NOT NULL,
    entity_id UUID NOT NULL,
    changes JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_rbac_audit_logs_user_id ON rbac_audit_logs(user_id);
CREATE INDEX idx_rbac_audit_logs_entity ON rbac_audit_logs(entity_type, entity_id);
CREATE INDEX idx_rbac_audit_logs_created_at ON rbac_audit_logs(created_at); 