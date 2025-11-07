-- Create or replace view for product sales report
DROP VIEW IF EXISTS product_sales_report;
CREATE VIEW product_sales_report AS
SELECT 
    p.id as product_id,
    p.name as product_name,
    p.category,
    p.price,
    COALESCE(SUM(oi.quantity), 0) as total_sold,
    COALESCE(SUM(oi.quantity * oi.unit_price), 0) as total_revenue,
    COUNT(DISTINCT oi.order_id) as order_count
FROM products p
LEFT JOIN order_items oi ON p.id = oi.product_id
GROUP BY p.id, p.name, p.category, p.price;