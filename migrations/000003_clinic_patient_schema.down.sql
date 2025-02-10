-- Drop indexes
DROP INDEX IF EXISTS idx_appointments_start_time;
DROP INDEX IF EXISTS idx_appointments_patient_id;
DROP INDEX IF EXISTS idx_appointments_user_id;
DROP INDEX IF EXISTS idx_appointments_clinic_id;
DROP INDEX IF EXISTS idx_patients_clinic_id;
DROP INDEX IF EXISTS idx_clinics_organization_id;

-- Drop tables in correct order
DROP TABLE IF EXISTS appointments;
DROP TABLE IF EXISTS patients;
DROP TABLE IF EXISTS clinics; 