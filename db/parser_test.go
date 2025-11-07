package db

import (
	"testing"
)

func TestParseDatabaseType(t *testing.T) {
	tests := []struct {
		name             string
		connectionString string
		expectedType     DatabaseType
		expectError      bool
	}{
		{
			name:             "SQLite connection string",
			connectionString: "sqlite:./bloom.db",
			expectedType:     SQLite,
			expectError:      false,
		},
		{
			name:             "PostgreSQL connection string",
			connectionString: "postgres://user:password@localhost/bloom?sslmode=disable",
			expectedType:     PostgreSQL,
			expectError:      false,
		},
		{
			name:             "Oracle connection string",
			connectionString: "oracle://user:password@localhost:1521/XE",
			expectedType:     Oracle,
			expectError:      false,
		},
		{
			name:             "Invalid connection string",
			connectionString: "invalid://test",
			expectedType:     "",
			expectError:      true,
		},
		{
			name:             "Empty connection string",
			connectionString: "",
			expectedType:     "",
			expectError:      true,
		},
		{
			name:             "MySQL connection string (unsupported)",
			connectionString: "mysql://user:password@localhost/test",
			expectedType:     "",
			expectError:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dbType, err := ParseDatabaseType(tt.connectionString)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for connection string '%s', but got none", tt.connectionString)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error for connection string '%s': %v", tt.connectionString, err)
				return
			}

			if dbType != tt.expectedType {
				t.Errorf("Expected database type '%s', got '%s'", tt.expectedType, dbType)
			}
		})
	}
}

func TestExtractConnectionString(t *testing.T) {
	tests := []struct {
		name             string
		connectionString string
		expectedConnStr  string
		expectError      bool
	}{
		{
			name:             "SQLite connection string extraction",
			connectionString: "sqlite:./bloom.db",
			expectedConnStr:  "./bloom.db",
			expectError:      false,
		},
		{
			name:             "PostgreSQL connection string extraction",
			connectionString: "postgres://user:password@localhost/bloom?sslmode=disable",
			expectedConnStr:  "postgres://user:password@localhost/bloom?sslmode=disable",
			expectError:      false,
		},
		{
			name:             "Oracle connection string extraction",
			connectionString: "oracle://user:password@localhost:1521/XE",
			expectedConnStr:  "oracle://user:password@localhost:1521/XE",
			expectError:      false,
		},
		{
			name:             "Invalid connection string",
			connectionString: "invalid://test",
			expectedConnStr:  "",
			expectError:      true,
		},
		{
			name:             "SQLite with absolute path",
			connectionString: "sqlite:/tmp/bloom.db",
			expectedConnStr:  "/tmp/bloom.db",
			expectError:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			connStr, err := ExtractConnectionString(tt.connectionString)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for connection string '%s', but got none", tt.connectionString)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error for connection string '%s': %v", tt.connectionString, err)
				return
			}

			if connStr != tt.expectedConnStr {
				t.Errorf("Expected connection string '%s', got '%s'", tt.expectedConnStr, connStr)
			}
		})
	}
}

func TestDatabaseTypeConstants(t *testing.T) {
	if SQLite != "sqlite" {
		t.Errorf("Expected SQLite constant to be 'sqlite', got '%s'", SQLite)
	}

	if PostgreSQL != "postgresql" {
		t.Errorf("Expected PostgreSQL constant to be 'postgresql', got '%s'", PostgreSQL)
	}

	if Oracle != "oracle" {
		t.Errorf("Expected Oracle constant to be 'oracle', got '%s'", Oracle)
	}
}
