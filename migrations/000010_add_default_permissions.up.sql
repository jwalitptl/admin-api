-- First add permissions
INSERT INTO permissions (id, name, description) 
VALUES 
    (gen_random_uuid(), 'patient:read', 'Can read patient data'),
    (gen_random_uuid(), 'patient:write', 'Can create/update patient data'),
    (gen_random_uuid(), 'appointment:read', 'Can read appointments'),
    (gen_random_uuid(), 'appointment:write', 'Can create/update appointments'),
    (gen_random_uuid(), 'appointment:create', 'Can create appointments'),
    (gen_random_uuid(), 'appointment:update', 'Can update appointments'),
    (gen_random_uuid(), 'appointment:delete', 'Can delete appointments'),
    (gen_random_uuid(), 'clinic:read', 'Can read clinic data'),
    (gen_random_uuid(), 'clinic:write', 'Can create/update clinic data'),
    (gen_random_uuid(), 'clinician:read', 'Can read clinician data'),
    (gen_random_uuid(), 'clinician:write', 'Can create/update clinician data')
ON CONFLICT (name) DO UPDATE 
SET description = EXCLUDED.description;

-- Then create admin role if it doesn't exist
INSERT INTO roles (id, name, description, is_system_role)
VALUES (gen_random_uuid(), 'admin', 'System Administrator', true)
ON CONFLICT (name) DO NOTHING;

-- Finally assign permissions to admin role
WITH new_permissions AS (
    SELECT r.id as role_id, p.id as permission_id
    FROM roles r
    CROSS JOIN permissions p
    WHERE r.name = 'admin'
)
INSERT INTO role_permissions (role_id, permission_id)
SELECT role_id, permission_id FROM new_permissions
ON CONFLICT (role_id, permission_id) DO NOTHING;