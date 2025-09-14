// Package service
package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/portbound/go-fs/internal/models"
	"github.com/portbound/go-fs/internal/repositories"
)

type FileService interface {
	ProcessFile(ctx context.Context, fm *models.FileMeta, owner *models.User) error
	DownloadFile(ctx context.Context, id string, owner *models.User) (io.ReadCloser, error)
	DeleteFile(ctx context.Context, id string, owner *models.User) error
	StageFileToDisk(ctx context.Context, fileName string, reader io.Reader) (string, int64, error)
}

type fileService struct {
	storage repositories.StorageRepository
	fms     FileMetaService
	tmpDir  string
}

func NewFileService(storageRepo repositories.StorageRepository, fileMetaService FileMetaService, tmpDir string) FileService {
	return &fileService{
		storage: storageRepo,
		fms:     fileMetaService,
		tmpDir:  tmpDir,
	}
}

func (fs *fileService) ProcessFile(ctx context.Context, fm *models.FileMeta, owner *models.User) error {
	_, err := fs.fms.LookupFileMetaByNameAndOwner(ctx, fm.Name, owner)
	if err == nil {
		return errors.New("file already exists")
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	thumbReader, err := GenerateThumbnail(ctx, fm)
	if err != nil {
		return fmt.Errorf("thumbnail generation failed: %w", err)
	}

	tfm := &models.FileMeta{
		ID:          fmt.Sprintf("thumb-%s", fm.ID),
		ParentID:    fm.ID,
		Name:        fmt.Sprintf("thumb-%s", fm.Name),
		ContentType: "image/jpeg",
		Owner:       fm.Owner,
	}

	tfm.Size, tfm.UploadDate, err = fs.storage.Upload(ctx, tfm.ID, owner.BucketName, thumbReader)
	if err != nil {
		return fmt.Errorf("upload failed: %w", err)
	}

	if err := fs.fms.SaveFileMeta(ctx, tfm); err != nil {
		rbErr := fs.DeleteFile(ctx, tfm.ID, owner)
		if rbErr != nil {
			err = errors.Join(err, fmt.Errorf("CRITICAL - failed to clean up orphaned file '%s': %w", tfm.ID, rbErr))
		}
		return fmt.Errorf("failed to save file meta: %w", err)
	}

	fileReader, err := os.Open(fm.TmpFilePath)
	if err != nil {
		return fmt.Errorf("failed to read file from disk: %w", err)
	}
	defer fileReader.Close()
	defer os.Remove(fm.TmpFilePath)

	fm.ThumbID = tfm.ID

	// TODO we need to nuke the thumbnail if this fails :(
	fm.Size, fm.UploadDate, err = fs.storage.Upload(ctx, fm.ID, owner.BucketName, thumbReader)
	if err != nil {
		return fmt.Errorf("upload failed: %w", err)
	}

	if err := fs.fms.SaveFileMeta(ctx, fm); err != nil {
		rbErr := fs.DeleteFile(ctx, tfm.ID, owner)
		if rbErr != nil {
			err = errors.Join(err, fmt.Errorf("CRITICAL - failed to clean up orphaned file '%s': %w", tfm.ID, rbErr))
		}
		return fmt.Errorf("failed to save file meta: %w", err)
	}

	return nil
}

func (fs *fileService) DownloadFile(ctx context.Context, id string, owner *models.User) (io.ReadCloser, error) {
	gcsReader, err := fs.storage.Download(ctx, id, owner.BucketName)
	if err != nil {
		return nil, err
	}

	return gcsReader, nil
}

func (fs *fileService) DeleteFile(ctx context.Context, id string, owner *models.User) error {
	if err := fs.storage.Delete(ctx, id, owner.BucketName); err != nil {
		return err
	}
	return nil
}

func (fs *fileService) StageFileToDisk(ctx context.Context, fileName string, reader io.Reader) (string, int64, error) {
	type chanl struct {
		bytesWritten int64
		err          error
	}

	path := fs.tmpDir
	if err := os.MkdirAll(path, 0755); err != nil {
		return "", 0, fmt.Errorf("failed to create tmp dir at '%s': %w", path, err)
	}

	file, err := os.Create(filepath.Join(path, fileName))
	if err != nil {
		return "", 0, fmt.Errorf("failed to create tmp file for '%s': %w", fileName, err)
	}
	defer file.Close()

	ch := make(chan *chanl, 1)
	go func() {
		bytesWritten, copyErr := io.Copy(file, reader)
		ch <- &chanl{bytesWritten: bytesWritten, err: copyErr}
	}()

	select {
	case <-ctx.Done():
		defer os.Remove(file.Name())
		return "", 0, ctx.Err()
	case result := <-ch:
		if result.err != nil {
			defer os.Remove(file.Name())
			return "", 0, err
		}
		return file.Name(), result.bytesWritten, nil
	}
}
