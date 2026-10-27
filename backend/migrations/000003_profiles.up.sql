-- The profile (spec 8B): a personal name, the name shown, and who may see
-- each element.
--
-- A level, from wider to narrower (8B.4): public (not logged in),
-- organization members, event participants, station members, private.

ALTER TABLE accounts
    ADD COLUMN first_name    text CHECK (first_name IS NULL OR btrim(first_name) <> ''),
    ADD COLUMN middle_name   text CHECK (middle_name IS NULL OR btrim(middle_name) <> ''),
    ADD COLUMN last_name     text CHECK (last_name IS NULL OR btrim(last_name) <> ''),
    -- Empty: the rule of 8B.2, the personal name, else the username, else
    -- the email address. Set: the person's own choice.
    ADD COLUMN name_shown    text CHECK (name_shown IN ('email', 'username', 'personal_name')),
    -- The email address starts at organization members while it is the only
    -- name the account has (8B.4, v1.04); Register sets private when a
    -- username is given.
    ADD COLUMN email_level   text NOT NULL DEFAULT 'organization',
    ADD COLUMN username_level text NOT NULL DEFAULT 'organization',
    ADD COLUMN name_level    text NOT NULL DEFAULT 'organization',
    ADD COLUMN roles_level   text NOT NULL DEFAULT 'organization',
    -- A personal name has both a first and a last name (8B.2); a middle name
    -- only with them.
    ADD CONSTRAINT accounts_personal_name CHECK (
        (first_name IS NULL AND middle_name IS NULL AND last_name IS NULL)
        OR (first_name IS NOT NULL AND last_name IS NOT NULL)),
    ADD CONSTRAINT accounts_levels CHECK (
        email_level IN ('public', 'organization', 'event', 'station', 'private')
        AND username_level IN ('public', 'organization', 'event', 'station', 'private')
        AND name_level IN ('public', 'organization', 'event', 'station', 'private')
        AND roles_level IN ('public', 'organization', 'event', 'station', 'private'));

-- Accounts made before this migration with a username start with the email
-- address private, as Register does from now on.
UPDATE accounts SET email_level = 'private' WHERE username IS NOT NULL;
