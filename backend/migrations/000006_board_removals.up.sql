-- Removing a person from an event board under 5D.3 (rulings of
-- November 11, 2026):
--   * before an Event Chair exists, a Board Member is removed by a unanimous
--     board vote, that member recused, and an Organization Leadership agent's
--     approval;
--   * a Board Member is terminated when Organization Leadership
--     and the Event Chair both agree; either may start it;
--   * the Event Chair is terminated by a unanimous board vote, the
--     Event Chair recused, and an Organization Leadership agent's approval.
-- A removal is created, read, edited (its cause) and deleted (withdrawn).
CREATE TABLE board_removals (
    id                     uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id               uuid NOT NULL REFERENCES events (id) ON DELETE CASCADE,
    organization_id        uuid NOT NULL REFERENCES organizations (id),
    account_id             uuid NOT NULL REFERENCES accounts (id),
    position               text NOT NULL CHECK (position IN ('chair', 'member')),
    cause                  text NOT NULL DEFAULT '',
    needs_vote             boolean NOT NULL,
    needs_chair            boolean NOT NULL,
    chair_agreed_by        uuid REFERENCES accounts (id),
    leadership_approved_by uuid REFERENCES accounts (id),
    status                 text NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'removed', 'declined')),
    started_by             uuid NOT NULL REFERENCES accounts (id),
    started_at             timestamptz NOT NULL DEFAULT now(),
    closed_at              timestamptz
);
-- One open removal per person per event.
CREATE UNIQUE INDEX board_removals_one_open ON board_removals (event_id, account_id) WHERE status = 'open';

CREATE TABLE board_removal_votes (
    removal_id uuid NOT NULL REFERENCES board_removals (id) ON DELETE CASCADE,
    account_id uuid NOT NULL REFERENCES accounts (id),
    in_favor   boolean NOT NULL,
    voted_at   timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (removal_id, account_id)
);
