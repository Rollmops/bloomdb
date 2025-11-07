-- Test repeatable migration for integration testing
-- PostgreSQL-compatible repeatable migration

CREATE OR REPLACE VIEW test_user_summary AS
SELECT 
    COUNT(*) as total_users,
    COUNT(CASE WHEN created_at > NOW() - INTERVAL '7 days' THEN 1 END) as new_users_this_week
FROM test_users;

CREATE OR REPLACE VIEW test_product_summary AS
SELECT 
    COUNT(*) as total_products,
    ROUND(AVG(price), 2) as average_price
FROM test_products;