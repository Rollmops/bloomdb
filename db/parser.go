package db

import (
	"fmt"
	"strings"
)

func ParseDatabaseType(connectionString string) (DatabaseType, error) {
	if strings.HasPrefix(connectionString, "sqlite:") {
		return SQLite, nil
	}
	if strings.HasPrefix(connectionString, "postgres://") {
		return PostgreSQL, nil
	}
	if strings.HasPrefix(connectionString, "oracle://") {
		return Oracle, nil
	}

	return "", fmt.Errorf("unable to determine database type from connection string: %s", connectionString)
}

func ExtractConnectionString(connectionString string) (string, error) {
	dbType, err := ParseDatabaseType(connectionString)
	if err != nil {
		return "", err
	}

	switch dbType {
	case SQLite:
		return strings.TrimPrefix(connectionString, "sqlite:"), nil
	case PostgreSQL:
		return connectionString, nil
	case Oracle:
		return connectionString, nil
	default:
		return "", fmt.Errorf("unsupported database type")
	}
}
