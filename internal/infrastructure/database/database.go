package database

import (
	"database/sql"
	"fmt"
)

type DBConnection struct {
	DB *sql.DB
}

func NewDBConnection(driverName string, connStr string, schema string) (*DBConnection, error) {
	db, err := sql.Open(driverName, connStr)
	if err != nil {
		return nil, fmt.Errorf("NewDBConnection.NewDB: failed to open sql connection: %w", err)
	}

	err = db.Ping()
	if err != nil {
		return nil, fmt.Errorf("NewDBConnection.NewDB: pinged db but got no response: %w", err)
	}

	_, err = db.Exec(schema)
	if err != nil {
		return nil, fmt.Errorf("NewDBConnection.NewDB: failed to create table: %w", err)
	}

	return &DBConnection{DB: db}, nil
}

func (dbConn *DBConnection) Close() error {
	return dbConn.DB.Close()
}
