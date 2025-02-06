-- Remove default permissions
DELETE FROM permissions 
WHERE name IN (
    'patient:read', 
    'patient:write',
    'appointment:read',
    'appointment:write',
    'clinic:read',
    'clinic:write',
    'clinician:read',
    'clinician:write'
); 