package sqlite

import (
	_ "embed"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
	"github.com/portbound/go-fs/internal/infrastructure/database"
)

//go:embed schema.sql
var schema string

type SQLiteDB struct {
	*Queries
	Conn *database.DBConnection
}

func NewSQLiteDB(connStr string) (*SQLiteDB, error) {
	conn, err := database.NewDBConnection("sqlite3", connStr, schema)
	if err != nil {
		return nil, fmt.Errorf("sqlite.NewSQLiteDB: failed to create new sqlite connection: %w", err)
	}
	return &SQLiteDB{Queries: New(conn.DB), Conn: conn}, nil
}
