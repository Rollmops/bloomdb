package db

import (
	"fmt"
)

func NewDatabase(dbType DatabaseType) (Database, error) {
	switch dbType {
	case SQLite:
		return NewSQLiteDatabase(), nil
	case PostgreSQL:
		return NewPostgreSQLDatabase(), nil
	case Oracle:
		return NewOracleDatabase(), nil
	default:
		return nil, fmt.Errorf("unsupported database type: %s", dbType)
	}
}

func NewDatabaseFromConnectionString(connectionString string) (Database, error) {
	dbType, err := ParseDatabaseType(connectionString)
	if err != nil {
		return nil, err
	}

	return NewDatabase(dbType)
}
