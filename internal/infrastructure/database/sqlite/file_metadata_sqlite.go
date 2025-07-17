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
		return nil, fmt.Errorf("failed to open sql connection: %w", err)
	}

	err = db.Ping()
	if err != nil {
		return nil, fmt.Errorf("pinged db but got no response: %w", err)
	}

	_, err = db.Exec(schema)
	if err != nil {
		return nil, fmt.Errorf("failed to create table: %w", err)
	}

	queries := New(db)

	return &DB{db: db, Queries: queries}, nil
}

func mapFile(f File) (*models.File, error) {
	uploadDate, err := time.Parse(time.RFC3339, f.UploadDate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse upload date: %w", err)
	}

	var modifiedDate time.Time

	if f.ModifiedDate.Valid {
		modifiedDate, err = time.Parse(time.RFC3339, f.ModifiedDate.String)
		if err != nil {
			return nil, fmt.Errorf("failed to parse last modified date: %w", err)
		}
	}

	return &models.File{
		ID:           f.ID,
		Name:         f.Name,
		OriginalName: f.OriginalName,
		Owner:        f.Owner,
		Type:         f.Type,
		Size:         f.Size,
		Unit:         f.Unit,
		UploadDate:   uploadDate,
		ModifiedDate: modifiedDate,
		StoragePath:  f.StoragePath,
	}, nil
}

func (db *DB) Create(ctx context.Context, file *models.File) error {
	params := CreateParams{
		ID:           file.ID,
		Name:         file.Name,
		OriginalName: file.OriginalName,
		Owner:        file.Owner,
		Type:         file.Type,
		Size:         file.Size,
		Unit:         file.Unit,
		UploadDate:   file.UploadDate.Format(time.RFC3339),
		StoragePath:  file.StoragePath,
	}
	return db.Queries.Create(ctx, params)
}

func (db *DB) Get(ctx context.Context, id uuid.UUID) (*models.File, error) {
	file, err := db.Queries.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return mapFile(file)
}

func (db *DB) GetAll(ctx context.Context) ([]*models.File, error) {
	data, err := db.Queries.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	var files []*models.File
	for _, row := range data {
		file, err := mapFile(row)
		if err != nil {
			return nil, err
		}
		files = append(files, file)
	}
	return files, nil
}

func (db *DB) Update(ctx context.Context, id uuid.UUID, file *models.File) error {
	params := UpdateParams{
		ID:   file.ID,
		Name: file.Name,
		Type: file.Type,
		Size: file.Size,
		Unit: file.Unit,
		ModifiedDate: sql.NullString{
			String: time.Now().UTC().Format(time.RFC3339),
			Valid:  true,
		},
		StoragePath: file.StoragePath,
	}
	return db.Queries.Update(ctx, params)
}

func (db *DB) Delete(ctx context.Context, id uuid.UUID) error {
	return db.Queries.Delete(ctx, id)
}
