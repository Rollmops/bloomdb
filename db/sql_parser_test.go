package db

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseSQLStatements(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:  "single statement with semicolon",
			input: "SELECT 1;",
			expected: []string{
				"SELECT 1",
			},
		},
		{
			name:  "multiple statements",
			input: "CREATE TABLE users (id INT); INSERT INTO users VALUES (1);",
			expected: []string{
				"CREATE TABLE users (id INT)",
				"INSERT INTO users VALUES (1)",
			},
		},
		{
			name: "statements with newlines",
			input: `CREATE TABLE users (
    id INT,
    name VARCHAR(100)
);

INSERT INTO users (id, name) VALUES (1, 'Alice');`,
			expected: []string{
				"CREATE TABLE users (\n    id INT,\n    name VARCHAR(100)\n)",
				"INSERT INTO users (id, name) VALUES (1, 'Alice')",
			},
		},
		{
			name: "statements with comments",
			input: `-- Create users table
CREATE TABLE users (id INT);

-- Insert test data
INSERT INTO users VALUES (1);`,
			expected: []string{
				"CREATE TABLE users (id INT)",
				"INSERT INTO users VALUES (1)",
			},
		},
		{
			name: "comment-only blocks are skipped",
			input: `-- Just comments
-- More comments

CREATE TABLE users (id INT);`,
			expected: []string{
				"CREATE TABLE users (id INT)",
			},
		},
		{
			name:     "empty content",
			input:    "",
			expected: []string{},
		},
		{
			name:     "only comments",
			input:    "-- Just a comment\n-- Another comment",
			expected: []string{},
		},
		{
			name:     "only whitespace",
			input:    "   \n\n\t\t  ",
			expected: []string{},
		},
		{
			name:  "statement without trailing semicolon",
			input: "SELECT 1",
			expected: []string{
				"SELECT 1",
			},
		},
		{
			name: "complex view creation",
			input: `CREATE OR REPLACE VIEW test_summary AS
SELECT 
    COUNT(*) as total,
    AVG(price) as avg_price
FROM products;

CREATE OR REPLACE VIEW user_summary AS
SELECT COUNT(*) as total_users FROM users;`,
			expected: []string{
				"CREATE OR REPLACE VIEW test_summary AS\nSELECT \n    COUNT(*) as total,\n    AVG(price) as avg_price\nFROM products",
				"CREATE OR REPLACE VIEW user_summary AS\nSELECT COUNT(*) as total_users FROM users",
			},
		},
		{
			name: "statement with string containing semicolon",
			input: `INSERT INTO logs (message) VALUES ('Error: connection failed;');
SELECT * FROM logs;`,
			expected: []string{
				"INSERT INTO logs (message) VALUES ('Error: connection failed",
				"')",
				"SELECT * FROM logs",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseSQLStatements(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsCommentOnly(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "single comment line",
			input:    "-- This is a comment",
			expected: true,
		},
		{
			name:     "multiple comment lines",
			input:    "-- Comment 1\n-- Comment 2",
			expected: true,
		},
		{
			name:     "comment with whitespace",
			input:    "  -- Comment with leading spaces  ",
			expected: true,
		},
		{
			name:     "SQL statement",
			input:    "SELECT 1",
			expected: false,
		},
		{
			name:     "comment followed by SQL",
			input:    "-- Comment\nSELECT 1",
			expected: false,
		},
		{
			name:     "empty string",
			input:    "",
			expected: true,
		},
		{
			name:     "only whitespace",
			input:    "   \n\t  ",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isCommentOnly(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
