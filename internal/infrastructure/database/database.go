package database

import (
	"database/sql"
	"fmt"
)

type DBConnection struct {
	DB *sql.DB
}

type DBConnectionDetails struct {
	DriverName string
	ConnStr    string
	Schema     string
}

func NewDBConnection(d *DBConnectionDetails) (*DBConnection, error) {
	db, err := sql.Open(d.DriverName, d.ConnStr)
	if err != nil {
		return nil, fmt.Errorf("[NewDBConnection.NewDB] failed to open sql connection: %w", err)
	}

	err = db.Ping()
	if err != nil {
		return nil, fmt.Errorf("[NewDBConnection.NewDB] pinged db but got no response: %w", err)
	}

	_, err = db.Exec(d.Schema)
	if err != nil {
		return nil, fmt.Errorf("[NewDBConnection.NewDB] failed to create table: %w", err)
	}

	return &DBConnection{DB: db}, nil
}

func (dbConn *DBConnection) Close() error {
	return dbConn.DB.Close()
}
