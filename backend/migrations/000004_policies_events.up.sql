-- Phase 3: O3, Policies (16B.6, 7C.4, 7B.7) and S1, Create event (5A.4,
-- 5E, 16B.6).

-- The organization's policies for all its events. Only the settings the
-- first release acts on are stored; waiting lists (16B.2, feature 13) and
-- lowering capacity (feature 21) add theirs when they arrive. An
-- organization with no entry has the starting values below.
CREATE TABLE organization_policies (
    organization_id      uuid PRIMARY KEY REFERENCES organizations (id),
    -- O3: "Approve each registration before the person chooses stations"
    -- (4A.11). The starting value is the one the wireframe draws.
    approve_registrations boolean NOT NULL DEFAULT true,
    -- Who participates in an event (7C.4). Starting value as drawn.
    participation        text NOT NULL DEFAULT 'both'
                         CHECK (participation IN ('subscribers', 'registrants', 'both')),
    -- Whether the Event Administrator holds a seat; no seat by default (7B.7).
    event_admin_holds_seat boolean NOT NULL DEFAULT false,
    updated_at           timestamptz NOT NULL DEFAULT now(),
    updated_by           uuid REFERENCES accounts (id)
);

-- An event belongs to its organization from the moment it exists, and starts
-- with no board and no chair (5E.1).
CREATE TABLE events (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id uuid NOT NULL REFERENCES organizations (id),
    name            text NOT NULL CHECK (btrim(name) <> ''),
    event_date      date,
    location        text CHECK (location IS NULL OR btrim(location) <> ''),
    description     text CHECK (description IS NULL OR btrim(description) <> ''),
    -- Includes the chair (section 5); at least 1 (5A.4, step 1).
    board_capacity  integer NOT NULL CHECK (board_capacity >= 1),
    -- The first release has private events only (16B.6); the other two
    -- modes come with feature 14.
    visibility      text NOT NULL DEFAULT 'private'
                    CHECK (visibility IN ('public', 'unlisted', 'private')),
    -- Which comes first, chosen on S1 until rules arrive (5E.1, 16B.6).
    board_order     text NOT NULL CHECK (board_order IN ('board_first', 'chair_first')),
    created_by      uuid NOT NULL REFERENCES accounts (id),
    created_at      timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX events_org ON events (organization_id, event_date);
