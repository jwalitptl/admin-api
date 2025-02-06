#!/bin/bash

# Colors for output
GREEN='\033[0;32m'
NC='\033[0m'

echo -e "${GREEN}=== Cleaning up database ===${NC}"

# SQL commands to truncate all tables in the correct order
docker exec -i aiclinic-postgres-1 psql -U postgres -d aiclinic << EOF
-- Disable foreign key checks
SET session_replication_role = 'replica';

-- Truncate all tables in the correct order
TRUNCATE TABLE 
    appointments,
    clinician_roles,
    role_permissions,
    clinic_clinicians,
    permissions,
    roles,
    clinicians,
    clinics,
    organizations,
    accounts
CASCADE;

-- Re-enable foreign key checks
SET session_replication_role = 'origin';

-- Reset all sequences
SELECT setval(c.oid, 1, false)
FROM pg_class c
JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE c.relkind = 'S'
AND n.nspname = 'public';
EOF

echo -e "${GREEN}Database cleaned successfully${NC}" 