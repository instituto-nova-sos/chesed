-- Sprint 9.1 — PostgreSQL Row-Level Security (defense-in-depth campus isolation).
--
-- Model A (non-owner app role): the application connects as chesed_app, which is
-- NOT the table owner and is therefore subject to RLS. golang-migrate and seed
-- run as the owner role (chesed), which bypasses RLS automatically — so no FORCE
-- ROW LEVEL SECURITY is set and migrations/seed keep working without a GUC.
--
-- The campus is carried per request via the session variable app.current_campus,
-- set with set_config(..., is_local => true) inside the request transaction. The
-- policies read it with current_setting('app.current_campus', true) which returns
-- NULL when unset, so an unset GUC matches no rows (fail closed), never all rows.
--
-- The chesed_app role is created here WITHOUT a password. Its password is set out
-- of band (Postgres init / ops), so no secret is committed to source. If the role
-- already exists (e.g. created by an init script), creation is skipped.

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'chesed_app') THEN
        CREATE ROLE chesed_app LOGIN;
    END IF;
END
$$;

GRANT USAGE ON SCHEMA public TO chesed_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO chesed_app;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO chesed_app;
-- Future tables/sequences (later migrations) are granted automatically.
ALTER DEFAULT PRIVILEGES IN SCHEMA public
    GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO chesed_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA public
    GRANT USAGE, SELECT ON SEQUENCES TO chesed_app;

-- Direct-column tables: campus_id lives on the row.
DO $$
DECLARE
    tbl TEXT;
BEGIN
    FOREACH tbl IN ARRAY ARRAY[
        'person', 'triage', 'attendance', 'campaign', 'consent', 'document', 'donation'
    ]
    LOOP
        EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', tbl);
        EXECUTE format(
            'CREATE POLICY %I ON %I '
            || 'USING (campus_id = current_setting(''app.current_campus'', true)::uuid) '
            || 'WITH CHECK (campus_id = current_setting(''app.current_campus'', true)::uuid)',
            tbl || '_campus_isolation', tbl
        );
    END LOOP;
END
$$;

-- Join-inherited tables: no campus_id column; isolate via the parent row.
ALTER TABLE address ENABLE ROW LEVEL SECURITY;
CREATE POLICY address_campus_isolation ON address
    USING (EXISTS (SELECT 1 FROM person p
                   WHERE p.id = address.person_id
                     AND p.campus_id = current_setting('app.current_campus', true)::uuid))
    WITH CHECK (EXISTS (SELECT 1 FROM person p
                        WHERE p.id = address.person_id
                          AND p.campus_id = current_setting('app.current_campus', true)::uuid));

ALTER TABLE assisted_profile ENABLE ROW LEVEL SECURITY;
CREATE POLICY assisted_profile_campus_isolation ON assisted_profile
    USING (EXISTS (SELECT 1 FROM person p
                   WHERE p.id = assisted_profile.person_id
                     AND p.campus_id = current_setting('app.current_campus', true)::uuid))
    WITH CHECK (EXISTS (SELECT 1 FROM person p
                        WHERE p.id = assisted_profile.person_id
                          AND p.campus_id = current_setting('app.current_campus', true)::uuid));

ALTER TABLE campaign_team ENABLE ROW LEVEL SECURITY;
CREATE POLICY campaign_team_campus_isolation ON campaign_team
    USING (EXISTS (SELECT 1 FROM campaign c
                   WHERE c.id = campaign_team.campaign_id
                     AND c.campus_id = current_setting('app.current_campus', true)::uuid))
    WITH CHECK (EXISTS (SELECT 1 FROM campaign c
                        WHERE c.id = campaign_team.campaign_id
                          AND c.campus_id = current_setting('app.current_campus', true)::uuid));

ALTER TABLE attendance_transition ENABLE ROW LEVEL SECURITY;
CREATE POLICY attendance_transition_campus_isolation ON attendance_transition
    USING (EXISTS (SELECT 1 FROM attendance a
                   WHERE a.id = attendance_transition.attendance_id
                     AND a.campus_id = current_setting('app.current_campus', true)::uuid))
    WITH CHECK (EXISTS (SELECT 1 FROM attendance a
                        WHERE a.id = attendance_transition.attendance_id
                          AND a.campus_id = current_setting('app.current_campus', true)::uuid));

-- triage_requested_service: isolate via its parent triage's campus.
ALTER TABLE triage_requested_service ENABLE ROW LEVEL SECURITY;
CREATE POLICY triage_requested_service_campus_isolation ON triage_requested_service
    USING (EXISTS (SELECT 1 FROM triage t
                   WHERE t.id = triage_requested_service.triage_id
                     AND t.campus_id = current_setting('app.current_campus', true)::uuid))
    WITH CHECK (EXISTS (SELECT 1 FROM triage t
                        WHERE t.id = triage_requested_service.triage_id
                          AND t.campus_id = current_setting('app.current_campus', true)::uuid));

-- audit_log is intentionally NOT under RLS: campus_id is nullable and rows are
-- written for pre-campus events (login/provisioning denials). It stays
-- append-only and application-filtered.
