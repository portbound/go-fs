package database

import (
	"database/sql"
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
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(d.Schema)
	if err != nil {
		return nil, err
	}

	return &DBConnection{DB: db}, nil
}

func (dbConn *DBConnection) Close() error {
	return dbConn.DB.Close()
}
