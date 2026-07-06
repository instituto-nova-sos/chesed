-- Revert Sprint 9.1 RLS. Drop policies, disable RLS, revoke grants, drop role.

DO $$
DECLARE
    tbl TEXT;
BEGIN
    FOREACH tbl IN ARRAY ARRAY[
        'person', 'triage', 'attendance', 'campaign', 'consent', 'document', 'donation',
        'address', 'assisted_profile', 'campaign_team', 'attendance_transition',
        'triage_requested_service'
    ]
    LOOP
        EXECUTE format('DROP POLICY IF EXISTS %I ON %I', tbl || '_campus_isolation', tbl);
        EXECUTE format('ALTER TABLE %I DISABLE ROW LEVEL SECURITY', tbl);
    END LOOP;
END
$$;

ALTER DEFAULT PRIVILEGES IN SCHEMA public
    REVOKE SELECT, INSERT, UPDATE, DELETE ON TABLES FROM chesed_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA public
    REVOKE USAGE, SELECT ON SEQUENCES FROM chesed_app;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'chesed_app') THEN
        REVOKE ALL PRIVILEGES ON ALL TABLES IN SCHEMA public FROM chesed_app;
        REVOKE ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public FROM chesed_app;
        REVOKE USAGE ON SCHEMA public FROM chesed_app;
        DROP ROLE chesed_app;
    END IF;
END
$$;
