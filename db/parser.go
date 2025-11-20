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
	if strings.HasPrefix(connectionString, "mysql://") || strings.HasPrefix(connectionString, "mysql:") {
		return MySQL, nil
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
	case MySQL:
		// MySQL driver expects DSN without scheme, but we might receive it with scheme
		// Standard format: user:password@tcp(host:port)/dbname
		// If it starts with mysql://, strip it
		clean := strings.TrimPrefix(connectionString, "mysql://")
		clean = strings.TrimPrefix(clean, "mysql:")
		return clean, nil
	default:
		return "", fmt.Errorf("unsupported database type")
	}
}

// ParseSQLStatements splits SQL content into individual statements
// Each statement is separated by a semicolon (;)
// Trailing semicolons are stripped from each statement
// Empty statements (whitespace only) are filtered out
func ParseSQLStatements(content string) []string {
	// Split by semicolon
	parts := strings.Split(content, ";")

	statements := []string{} // Initialize as empty slice, not nil
	for _, part := range parts {
		// Trim whitespace
		trimmed := strings.TrimSpace(part)

		// Skip empty statements
		if trimmed == "" {
			continue
		}

		// Skip comment-only blocks
		if isCommentOnly(trimmed) {
			continue
		}

		// Remove leading comment-only lines from the statement
		trimmed = removeLeadingComments(trimmed)
		if trimmed == "" {
			continue
		}

		statements = append(statements, trimmed)
	}

	return statements
}

// isCommentOnly checks if a statement contains only comments
func isCommentOnly(statement string) bool {
	lines := strings.Split(statement, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Skip empty lines
		if trimmed == "" {
			continue
		}
		// If we find a non-comment line, it's not comment-only
		if !strings.HasPrefix(trimmed, "--") {
			return false
		}
	}
	return true
}

// removeLeadingComments removes comment-only lines from the beginning of a statement
func removeLeadingComments(statement string) string {
	lines := strings.Split(statement, "\n")

	// Find the first non-comment, non-empty line
	startIndex := 0
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && !strings.HasPrefix(trimmed, "--") {
			startIndex = i
			break
		}
	}

	// If all lines are comments or empty, return empty string
	if startIndex == 0 && (len(lines) == 0 || isCommentOnly(statement)) {
		return ""
	}

	// Return the statement starting from the first non-comment line
	return strings.Join(lines[startIndex:], "\n")
}
