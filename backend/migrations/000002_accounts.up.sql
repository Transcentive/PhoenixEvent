-- Phase 2, step 1: organizations, their accounts, and signing in (spec 4, 4A,
-- 7B.1, 8, 8A). Every record that belongs to an organization carries it.

CREATE TABLE organizations (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name       text NOT NULL CHECK (btrim(name) <> ''),
    -- The address name: lincolnpta in lincolnpta.phoenixevent.com (4A.5).
    -- Fixed once set in the first release (4A.6).
    slug       text NOT NULL UNIQUE
               CHECK (slug ~ '^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$'),
    created_at timestamptz NOT NULL DEFAULT now()
);

-- An account belongs to one organization (4A.7).
CREATE TABLE accounts (
    id                 uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id    uuid NOT NULL REFERENCES organizations (id),
    email              text NOT NULL CHECK (email LIKE '%_@_%'),
    -- Optional (8B.1). Never contains @, so it can never match an email
    -- address (4A.8).
    username           text CHECK (username IS NULL OR (username NOT LIKE '%@%' AND btrim(username) <> '')),
    -- Empty until the first Organization Administrator sets one from the
    -- emailed link (4A.6).
    password_hash      text,
    email_confirmed_at timestamptz,
    created_at         timestamptz NOT NULL DEFAULT now(),
    updated_at         timestamptz NOT NULL DEFAULT now()
);

-- Unique within the organization, without regard to capitals (4A.8).
CREATE UNIQUE INDEX accounts_email_unique ON accounts (organization_id, lower(email));
CREATE UNIQUE INDEX accounts_username_unique ON accounts (organization_id, lower(username))
    WHERE username IS NOT NULL;

-- The two organization roles a person may be given (7B.1). Member is every
-- account and is not stored.
CREATE TABLE organization_roles (
    organization_id uuid NOT NULL REFERENCES organizations (id),
    account_id      uuid NOT NULL REFERENCES accounts (id),
    role            text NOT NULL
                    CHECK (role IN ('organization_administrator', 'organization_leadership')),
    granted_at      timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (account_id, role)
);
CREATE INDEX organization_roles_org ON organization_roles (organization_id);

-- One row per sign-in in one browser. The refresh token's hash changes each
-- time it is used; the row's id stays, and the access token names it (2A.3).
CREATE TABLE sessions (
    id                 uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id    uuid NOT NULL REFERENCES organizations (id),
    account_id         uuid NOT NULL REFERENCES accounts (id),
    refresh_token_hash text NOT NULL UNIQUE,
    created_at         timestamptz NOT NULL DEFAULT now(),
    last_used_at       timestamptz NOT NULL DEFAULT now(),
    -- 30 days from the last use (8A.2).
    expires_at         timestamptz NOT NULL,
    revoked_at         timestamptz
);
CREATE INDEX sessions_account ON sessions (account_id);

-- Single-use links sent by email (8A.2). Only the hash is kept.
CREATE TABLE email_links (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id uuid NOT NULL REFERENCES accounts (id),
    purpose    text NOT NULL CHECK (purpose IN ('confirm_email', 'reset_password', 'set_password')),
    token_hash text NOT NULL UNIQUE,
    created_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL,
    used_at    timestamptz
);
CREATE INDEX email_links_account ON email_links (account_id, purpose);

-- Failed logins, counted per username for the pause in 8A.2.
CREATE TABLE login_failures (
    organization_id uuid NOT NULL REFERENCES organizations (id),
    login_key       text NOT NULL,
    failed_at       timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX login_failures_lookup ON login_failures (organization_id, login_key, failed_at);

-- The audit record (section 2): who did what, in which organization.
CREATE TABLE audit_log (
    id              bigserial PRIMARY KEY,
    organization_id uuid REFERENCES organizations (id),
    account_id      uuid REFERENCES accounts (id),
    action          text NOT NULL,
    detail          jsonb NOT NULL DEFAULT '{}',
    created_at      timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX audit_log_org ON audit_log (organization_id, created_at);
