-- Remove appointment permissions from roles
DELETE FROM role_permissions
WHERE permission_id IN (
    SELECT id FROM permissions WHERE name LIKE 'appointment:%'
);

-- Remove appointment permissions
DELETE FROM permissions WHERE name LIKE 'appointment:%'; 