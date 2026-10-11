-- Phase 1, the development foundation (spec section 16, as 2A.3 amends it).
-- This migration exists so that the migration path itself is proven on every
-- host. The domain tables begin in Phase 2.

CREATE TABLE app_info (
    key   text PRIMARY KEY,
    value text NOT NULL
);

INSERT INTO app_info (key, value) VALUES ('installed_at', now()::text);
