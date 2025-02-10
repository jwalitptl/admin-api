-- Drop indexes
DROP INDEX IF EXISTS idx_audit_logs_created_at;
DROP INDEX IF EXISTS idx_audit_logs_entity;
DROP INDEX IF EXISTS idx_audit_logs_org;
DROP INDEX IF EXISTS idx_audit_logs_user;
DROP INDEX IF EXISTS idx_medical_records_patient;
DROP INDEX IF EXISTS idx_medical_records_created_by;

-- Drop tables
DROP TABLE IF EXISTS medical_records;
DROP TABLE IF EXISTS audit_logs; 