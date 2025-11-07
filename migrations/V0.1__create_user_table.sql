-- Create initial users table
-- Version 0.1 - Initial user management

CREATE TABLE users (
                       id SERIAL PRIMARY KEY,
                       username VARCHAR(255) NOT NULL UNIQUE,
                       email VARCHAR(255) NOT NULL UNIQUE,
                       password_hash VARCHAR(255) NOT NULL,
                       first_name VARCHAR(100),
                       last_name VARCHAR(100),
                       created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create index for username lookup
CREATE INDEX idx_users_username ON users(username);

-- Create index for email lookup
CREATE INDEX idx_users_email ON users(email);