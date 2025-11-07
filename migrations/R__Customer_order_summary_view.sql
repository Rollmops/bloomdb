-- Create or replace view for customer order summary
DROP VIEW IF EXISTS customer_order_summary;
CREATE VIEW customer_order_summary AS
SELECT 
    c.id as customer_id,
    c.name as customer_name,
    c.email,
    COUNT(o.id) as order_count,
    COALESCE(SUM(o.total_amount), 0) as total_spent,
    MAX(o.order_date) as last_order_date
FROM customers c
LEFT JOIN orders o ON c.id = o.customer_id
GROUP BY c.id, c.name, c.email;