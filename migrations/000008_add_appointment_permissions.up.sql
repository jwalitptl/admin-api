-- Add appointment permissions
INSERT INTO permissions (id, name, description, created_at, updated_at) 
VALUES 
    (gen_random_uuid(), 'appointment:create', 'Can create appointments', NOW(), NOW()),
    (gen_random_uuid(), 'appointment:read', 'Can read appointments', NOW(), NOW()),
    (gen_random_uuid(), 'appointment:update', 'Can update appointments', NOW(), NOW()),
    (gen_random_uuid(), 'appointment:delete', 'Can delete appointments', NOW(), NOW())
ON CONFLICT (name) DO UPDATE 
SET description = EXCLUDED.description;

-- Add appointment permissions to admin role
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'admin'
AND p.name LIKE 'appointment:%'
ON CONFLICT DO NOTHING; 