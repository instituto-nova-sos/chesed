-- Extend the audit_log.action_type domain with ACCESS_DENIED so RBAC 403
-- denials can be recorded (security Finding 4). This is additive: it does not
-- grant UPDATE/DELETE on audit_log, so the table stays append-only.
ALTER TABLE audit_log
    DROP CONSTRAINT audit_log_action_type_check;

ALTER TABLE audit_log
    ADD CONSTRAINT audit_log_action_type_check
    CHECK (action_type IN ('CREATE', 'READ', 'UPDATE', 'DELETE', 'LOGIN', 'LOGOUT', 'EXPORT', 'PERMISSION_CHANGE', 'ACCESS_DENIED'));
