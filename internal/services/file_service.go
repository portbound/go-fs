// Package services
package services

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/storage"
	"github.com/portbound/go-fs/internal/models"
	"github.com/portbound/go-fs/internal/repositories"
	"github.com/portbound/go-fs/internal/utils"
)

type FileService struct {
	db          repositories.FileRepository
	storage     repositories.StorageRepository
	thumbnailer *ThumbnailService
	logger      *slog.Logger
	logFile     io.Closer
	TmpDir      string
}

func NewFileService(fileRepo repositories.FileRepository, storageRepo repositories.StorageRepository, tmpDir string, logsDir string) (*FileService, error) {

	log, logger, err := setupLogging(logsDir)
	if err != nil {
		return nil, fmt.Errorf("NewFileService: failed to setup logging: %w", err)
	}

	return &FileService{
		db:          fileRepo,
		storage:     storageRepo,
		thumbnailer: NewThumbnailService(),
		logFile:     log,
		logger:      logger,
		TmpDir:      tmpDir,
	}, nil
}

func (fs *FileService) CloseLog() error {
	return fs.logFile.Close()
}

func (fs *FileService) ProcessBatch(ctx context.Context, batch []*models.FileMeta) []error {
	var wg sync.WaitGroup
	var batchErrs []error

	ch := make(chan error)
	for _, fm := range batch {
		wg.Add(1)
		go func() {
			defer wg.Done()

			t := strings.Split(fm.ContentType, "/")
			if t[0] == "image" || t[0] == "video" {
				thumbnailReader, err := fs.thumbnailer.Generate(ctx, fm)
				if err != nil {
					ch <- fmt.Errorf("services.ProcessBatch: failed to generate thumbnail for '%s': %w", fm.Name, err)
					return
				}

				if thumbnailReader != nil {
					tfm := &models.FileMeta{
						ID:          fmt.Sprintf("thumb-%s", fm.ID),
						ParentID:    fm.ID,
						ThumbID:     "",
						Name:        fmt.Sprintf("thumb-%s", fm.Name),
						ContentType: "image/jpeg",
						Owner:       fm.Owner,
					}
					fm.ThumbID = tfm.ID

					tfm.TmpFilePath, err = utils.StageFileToDisk(ctx, fs.TmpDir, tfm.ID, thumbnailReader)
					if err != nil {
						ch <- fmt.Errorf("services.Processbatch: failed to stage thumbnail for %s to disk: %w", fm.Name, err)
					}
					defer os.Remove(tfm.TmpFilePath)

					if err = fs.processFile(ctx, tfm); err != nil {
						ch <- fmt.Errorf("services.ProcessBatch: failed to process thumbnail for %s: %w", fm.Name, err)
						return
					}

					fs.logger.Info("Thumbnail Upload: Success", "file", fm.Name)
				}
			}

			fileReader, err := os.Open(fm.TmpFilePath)
			if err != nil {
				ch <- fmt.Errorf("services.ProcessBatch: failed to open %s: %w", fm.TmpFilePath, err)
				return
			}
			defer fileReader.Close()

			if err := fs.processFile(ctx, fm); err != nil {
				if fm.ThumbID != "" {
					if err = fs.DeleteFile(ctx, fm.ThumbID); err != nil {
						fs.logger.Error("Delete File: CRITICAL - Failed to delete orphaned thumbnail", "thumb_id", fm.ThumbID, "error", err)
						ch <- fmt.Errorf("CRITICAL services.ProcessBatch: failed to delete orphaned thumbnail %s: %v", fm.ThumbID, err)
					}
				}
				ch <- fmt.Errorf("services:ProcessBatch: failed to process %s: %w", fm.Name, err)
				return
			}

			fs.logger.Info("File Upload: Success", "file", fm.Name)
			ch <- nil
		}()
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	for err := range ch {
		if err != nil {
			batchErrs = append(batchErrs, err)
		}
	}

	return batchErrs
}

func (fs *FileService) GetFile(ctx context.Context, id string) (io.ReadCloser, error) {
	gcsReader, err := fs.storage.Download(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("services.GetFile: failed to get file from storage: %w", err)
	}

	return gcsReader, nil
}

func (fs *FileService) GetThumbnails(ctx context.Context) ([]string, error) {
	fileNames, err := fs.storage.ListObjects(ctx, &storage.Query{Prefix: "thumb-"})
	if err != nil {
		return nil, fmt.Errorf("services.GetBatch: failed to get fileNames from storage: %w", err)
	}

	return fileNames, nil
}

func (fs *FileService) DeleteFile(ctx context.Context, id string) error {
	if err := fs.storage.Delete(ctx, id); err != nil {
		return fmt.Errorf("services.DeleteFile: failed to delete %s from storage: %w", id, err)
	}
	fs.logger.Info("File Delete: Success", "ID", id)
	return nil
}

func (fs *FileService) LookupFileMeta(ctx context.Context, id string) (*models.FileMeta, error) {
	fm, err := fs.db.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("services.LookupFileMeta: failed to get file for id '%s': %w", id, err)
	}
	return fm, nil
}

func (fs *FileService) SaveFileMeta(ctx context.Context, fm *models.FileMeta) error {
	if err := fs.db.Create(ctx, fm); err != nil {
		return fmt.Errorf("services.SaveFileMeta: failed to save file metadata: %w", err)
	}
	return nil
}

func (fs *FileService) DeleteFileMeta(ctx context.Context, id string) error {
	if err := fs.db.Delete(ctx, id); err != nil {
		return fmt.Errorf("services.DeleteFileMeta: failed to delete file metadata: %w", err)
	}
	return nil
}

func (fs *FileService) processFile(ctx context.Context, fm *models.FileMeta) error {
	if err := fs.storage.Upload(ctx, fm.ID, fm.TmpFilePath); err != nil {
		return fmt.Errorf("upload failed for %s: %w", fm.Name, err)
	}

	if err := fs.SaveFileMeta(ctx, fm); err != nil {
		if rbErr := fs.DeleteFile(ctx, fm.ID); rbErr != nil {
			fs.logger.Error("File Upload: CRITICAL - failed to delete orphaned file from storage", "file", fm.Name, "error", rbErr)
			return fmt.Errorf("CRITICAL: failed to delete orphaned file %s from storage: %w", fm.Name, rbErr)
		}
		return fmt.Errorf("save metadata failed for %s: %w", fm.Name, err)
	}

	return nil
}

func setupLogging(logsDir string) (*os.File, *slog.Logger, error) {
	var log *os.File
	logName := fmt.Sprintf("%s-application.log", time.Now().Format("2006-01-02"))

	_, err := os.Stat(logName)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			log, err = os.Create(filepath.Join(logsDir, "application.log"))
			if err != nil {
				return nil, nil, fmt.Errorf("fileService.setupLogging: failed to create log file: %w", err)
			}
		}
		if err != nil {
			return nil, nil, fmt.Errorf("fileService.setupLogging: failed to setup logging: %w", err)
		}
	}

	logger := slog.New(slog.NewTextHandler(log, nil))
	return log, logger, nil
}
