-- Additional test migration for integration testing
-- PostgreSQL-compatible versioned migration

CREATE TABLE test_orders (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    product_id INTEGER NOT NULL,
    quantity INTEGER NOT NULL DEFAULT 1,
    order_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES test_users(id),
    FOREIGN KEY (product_id) REFERENCES test_products(id)
);

CREATE INDEX idx_test_orders_user_id ON test_orders(user_id);
CREATE INDEX idx_test_orders_product_id ON test_orders(product_id);