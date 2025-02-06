-- Drop the existing primary key if it exists
ALTER TABLE clinician_roles DROP CONSTRAINT IF EXISTS clinician_roles_pkey;

-- Add the new composite primary key
ALTER TABLE clinician_roles 
ADD CONSTRAINT clinician_roles_pkey 
PRIMARY KEY (clinician_id, role_id); 