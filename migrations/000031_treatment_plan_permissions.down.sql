DELETE FROM role_permissions WHERE permission_id IN (
    SELECT id FROM permissions
    WHERE module IN ('treatment_plans', 'session_records', 'follow_up_tasks')
);
DELETE FROM permissions
WHERE module IN ('treatment_plans', 'session_records', 'follow_up_tasks');
