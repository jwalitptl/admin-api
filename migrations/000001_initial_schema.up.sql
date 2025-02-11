-- Core tables with proper constraints
CREATE TABLE accounts (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL,
    plan VARCHAR(50) DEFAULT 'free',
    billing_email VARCHAR(255),
    subscription_status VARCHAR(50),
    trial_ends_at TIMESTAMP,
    settings JSONB DEFAULT '{}',
    region_code VARCHAR(10),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,
    CONSTRAINT chk_accounts_status CHECK (status IN ('active', 'inactive', 'suspended')),
    CONSTRAINT chk_subscription_status CHECK (subscription_status IN ('trial', 'active', 'past_due', 'canceled')),
    CONSTRAINT chk_plan CHECK (plan IN ('free', 'basic', 'pro', 'enterprise'))
);

CREATE TABLE organizations (
    id UUID PRIMARY KEY,
    account_id UUID NOT NULL REFERENCES accounts(id),
    name VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL,
    billing_email VARCHAR(255),
    phone VARCHAR(50),
    address JSONB,
    settings JSONB DEFAULT '{}',
    region_code VARCHAR(10),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,
    CONSTRAINT chk_organization_status CHECK (status IN ('active', 'inactive', 'suspended')),
    CONSTRAINT uq_organization_name_account UNIQUE (account_id, name)
);

CREATE TABLE users (
    id UUID PRIMARY KEY,
    organization_id UUID REFERENCES organizations(id),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    first_name VARCHAR(255),
    last_name VARCHAR(255),
    phone VARCHAR(50),
    type VARCHAR(50),
    status VARCHAR(50),
    email_verified BOOLEAN DEFAULT FALSE,
    region_code VARCHAR(10),
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    deleted_at TIMESTAMP,
    CONSTRAINT chk_users_status CHECK (status IN ('active', 'inactive', 'pending', 'suspended', 'locked')),
    CONSTRAINT chk_users_type CHECK (type IN ('admin', 'staff', 'provider', 'support', 'patient'))
);

-- Indexes
CREATE INDEX idx_organizations_account ON organizations(account_id);
CREATE INDEX idx_organizations_name ON organizations(name);
CREATE INDEX idx_organizations_status ON organizations(status);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_organization ON users(organization_id);

-- Additional useful indexes
CREATE INDEX idx_users_name ON users(first_name);
CREATE INDEX idx_users_phone ON users(phone);
CREATE INDEX idx_users_type ON users(type);

-- Email format constraint
ALTER TABLE users
    ADD CONSTRAINT chk_users_email_format
    CHECK (email ~* '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$');

-- Phone format constraint (basic)
ALTER TABLE users
    ADD CONSTRAINT chk_users_phone_format
    CHECK (phone IS NULL OR phone ~ '^\+?[0-9\s-\(\)]{10,20}$');

-- Add indexes for account lookups
CREATE INDEX idx_accounts_email ON accounts(email);
CREATE INDEX idx_accounts_status ON accounts(status);
CREATE INDEX idx_accounts_plan ON accounts(plan);
CREATE INDEX idx_accounts_subscription ON accounts(subscription_status); 