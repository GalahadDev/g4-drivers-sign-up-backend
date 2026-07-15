-- =============================================================
-- G4 Drivers — Security & Optimization Migration
-- Idempotent: safe to run multiple times on any Supabase project
-- =============================================================

-- ---------------------------------------------------------------
-- §1  Custom Access Token Hook
--     Injects `user_role` into every JWT before it is issued.
--     After running this SQL, activate the hook manually:
--     Dashboard → Authentication → Hooks → custom-access-token
--     → select function public.custom_access_token_hook
-- ---------------------------------------------------------------

CREATE OR REPLACE FUNCTION public.custom_access_token_hook(event jsonb)
RETURNS jsonb
LANGUAGE plpgsql
STABLE
AS $$
DECLARE
  claims    jsonb;
  user_role text;
BEGIN
  SELECT role INTO user_role
  FROM public.profiles
  WHERE id = (event->>'user_id')::uuid;

  claims := jsonb_set(
    event->'claims',
    '{user_role}',
    to_jsonb(COALESCE(user_role, 'driver'))
  );

  RETURN jsonb_set(event, '{claims}', claims);
END;
$$;

-- Grant the auth system permission to call the hook
GRANT USAGE ON SCHEMA public TO supabase_auth_admin;

GRANT EXECUTE
  ON FUNCTION public.custom_access_token_hook
  TO supabase_auth_admin;

REVOKE EXECUTE
  ON FUNCTION public.custom_access_token_hook
  FROM authenticated, anon, public;

GRANT ALL ON TABLE public.profiles TO supabase_auth_admin;

-- ---------------------------------------------------------------
-- §2  RLS Policies
--     Replace per-row subquery admin policies with jwt() claim.
--     Split driver_applications ALL policy into INSERT + SELECT.
-- ---------------------------------------------------------------

-- profiles: admin read policy
DROP POLICY IF EXISTS "Admins can view all" ON profiles;
CREATE POLICY "Admins can view all"
  ON profiles
  FOR SELECT
  TO authenticated
  USING ((auth.jwt() ->> 'user_role') = 'admin');

-- driver_applications: remove ALL, add INSERT + SELECT
DROP POLICY IF EXISTS "Users can manage own application" ON driver_applications;

CREATE POLICY "Users can submit own application"
  ON driver_applications
  FOR INSERT
  TO authenticated
  WITH CHECK (user_id = auth.uid());

CREATE POLICY "Users can view own application"
  ON driver_applications
  FOR SELECT
  TO authenticated
  USING (user_id = auth.uid());

DROP POLICY IF EXISTS "Admins can view all apps" ON driver_applications;
CREATE POLICY "Admins can view all apps"
  ON driver_applications
  FOR SELECT
  TO authenticated
  USING ((auth.jwt() ->> 'user_role') = 'admin');

-- ---------------------------------------------------------------
-- §3  Constraints
-- ---------------------------------------------------------------

ALTER TABLE driver_applications
  ADD CONSTRAINT IF NOT EXISTS driver_applications_user_id_unique
  UNIQUE (user_id);

-- ---------------------------------------------------------------
-- §4  Extensions
-- ---------------------------------------------------------------

CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- ---------------------------------------------------------------
-- §5  Indexes
-- ---------------------------------------------------------------

CREATE INDEX IF NOT EXISTS idx_profiles_role
  ON profiles(role);

CREATE INDEX IF NOT EXISTS idx_driver_apps_category_status
  ON driver_applications(driver_category, status);

CREATE INDEX IF NOT EXISTS idx_profiles_email_trgm
  ON profiles USING gin (email gin_trgm_ops);
