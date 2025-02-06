-- Remove the composite primary key
ALTER TABLE clinician_roles DROP CONSTRAINT IF EXISTS clinician_roles_pkey;

-- Add back the original primary key if needed
ALTER TABLE clinician_roles 
ADD CONSTRAINT clinician_roles_pkey 
PRIMARY KEY (clinician_id, role_id, organization_id); 