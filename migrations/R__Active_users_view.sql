-- Create or replace view for active users
DROP VIEW IF EXISTS active_users;
CREATE VIEW active_users AS
SELECT 
    ua.id,
    ua.username,
    c.name as customer_name,
    c.email,
    ua.created_at
FROM user_accounts ua
JOIN customers c ON ua.customer_id = c.id
WHERE ua.is_active = TRUE;