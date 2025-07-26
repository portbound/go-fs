package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "embed"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"github.com/portbound/go-fs/internal/models"
)

// go:embed schema.sql
var schema string

type DB struct {
	*Queries
	db *sql.DB
}

func NewDB(connStr string) (*DB, error) {
	db, err := sql.Open("sqlite3", connStr)
	if err != nil {
		return nil, fmt.Errorf("sqlite.NewDB: failed to open sql connection: %w", err)
	}

	err = db.Ping()
	if err != nil {
		return nil, fmt.Errorf("sqlite.NewDB: pinged db but got no response: %w", err)
	}

	_, err = db.Exec(schema)
	if err != nil {
		return nil, fmt.Errorf("sqlite.NewDB: failed to create table: %w", err)
	}

	queries := New(db)

	return &DB{db: db, Queries: queries}, nil
}

func (db *DB) Create(ctx context.Context, filemeta *models.FileMeta) error {
	params := CreateParams{
		ID:          filemeta.ID,
		Name:        filemeta.Name,
		Owner:       filemeta.Owner,
		ContentType: filemeta.Type,
		Size:        filemeta.Size,
		UploadDate:  filemeta.UploadDate.Format(time.RFC3339),
		StoragePath: filemeta.StoragePath,
	}
	return db.Queries.Create(ctx, params)
}

func (db *DB) Get(ctx context.Context, id uuid.UUID) (*models.FileMeta, error) {
	file, err := db.Queries.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return mapToFile(file)
}

func (db *DB) GetAll(ctx context.Context) ([]*models.FileMeta, error) {
	data, err := db.Queries.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	var files []*models.FileMeta
	for _, row := range data {
		file, err := mapToFile(row)
		if err != nil {
			return nil, err
		}
		files = append(files, file)
	}
	return files, nil
}

func (db *DB) Delete(ctx context.Context, id uuid.UUID) error {
	return db.Queries.Delete(ctx, id)
}

func mapToFile(f File) (*models.FileMeta, error) {
	uploadDate, err := time.Parse(time.RFC3339, f.UploadDate)
	if err != nil {
		return nil, fmt.Errorf("sqlite.mapToFile: failed to parse upload date: %w", err)
	}

	return &models.FileMeta{
		ID:          f.ID,
		Name:        f.Name,
		Owner:       f.Owner,
		Type:        f.ContentType,
		Size:        f.Size,
		UploadDate:  uploadDate,
		StoragePath: f.StoragePath,
	}, nil
}
