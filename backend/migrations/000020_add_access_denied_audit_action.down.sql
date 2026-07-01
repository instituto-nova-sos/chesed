-- Revert the audit_log.action_type domain to the original eight values.
ALTER TABLE audit_log
    DROP CONSTRAINT audit_log_action_type_check;

ALTER TABLE audit_log
    ADD CONSTRAINT audit_log_action_type_check
    CHECK (action_type IN ('CREATE', 'READ', 'UPDATE', 'DELETE', 'LOGIN', 'LOGOUT', 'EXPORT', 'PERMISSION_CHANGE'));
