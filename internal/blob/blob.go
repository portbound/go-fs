package blob

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	_ "image/gif"
	_ "image/png"

	"github.com/google/uuid"
	"github.com/portbound/go-fs/internal/user"
)

type Metadata struct {
	ID          string    `json:"id"`
	ParentID    string    `json:"parentId"`
	ThumbID     string    `json:"thumbId"`
	Name        string    `json:"name"`
	ContentType string    `json:"type"`
	Size        int64     `json:"size"`
	UploadDate  time.Time `json:"uploadDate"`
	Owner       string    `json:"owner"`
	TmpFilePath string    `json:"tmpFile"`
}

type FileMetaStore interface {
	Save(ctx context.Context, m *Metadata) error
	ById(ctx context.Context, id string, owner *user.User) (*Metadata, error)
	ByNameAndOwner(ctx context.Context, name string, owner *user.User) (*Metadata, error)
	All(ctx context.Context, owner *user.User) ([]*Metadata, error)
	Delete(ctx context.Context, id string, owner *user.User) error
}

type BlobStore interface {
	Upload(ctx context.Context, fileName string, bucket string, src io.Reader) (int64, time.Time, error)
	Download(ctx context.Context, fileName string, bucket string) (io.ReadCloser, error)
	Delete(ctx context.Context, fileName string, bucket string) error
}

type Service struct {
	meta FileMetaStore
	blob BlobStore
}

func NewService(m FileMetaStore, b BlobStore) *Service {
	return &Service{meta: m, blob: b}
}

func (s *Service) Upload(ctx context.Context, part *multipart.Part, owner *user.User) error {
	fileMeta := Metadata{
		ID:          uuid.New().String(),
		Name:        filepath.Base(part.FileName()),
		ContentType: part.Header.Get("Content-Type"),
		Owner:       owner.Email,
	}

	dbCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	_, err := s.meta.ByNameAndOwner(dbCtx, fileMeta.Name, owner)
	if err == nil {
		return errors.New("file already exists")
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	thumbReader, err := GenerateThumbnail(ctx, fileMeta)
	if err != nil {
		return fmt.Errorf("thumbnail generation failed: %w", err)
	}

	thumbMeta := &Metadata{
		ID:          fmt.Sprintf("thumb-%s", fileMeta.ID),
		ParentID:    fileMeta.ID,
		Name:        fmt.Sprintf("thumb-%s", fileMeta.Name),
		ContentType: "image/jpeg",
		Owner:       fileMeta.Owner,
	}

	thumbMeta.Size, thumbMeta.UploadDate, err = s.blob.Upload(ctx, thumbMeta.ID, owner.BucketName, thumbReader)
	if err != nil {
		return fmt.Errorf("upload failed: %w", err)
	}

	dbCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := s.meta.Save(dbCtx, fileMeta); err != nil {
		rbErr := fs.DeleteFile(ctx, thumbMeta.ID, owner)
		if rbErr != nil {
			err = errors.Join(err, fmt.Errorf("CRITICAL - failed to clean up orphaned file '%s': %w", thumbMeta.ID, rbErr))
		}
		return fmt.Errorf("failed to save file meta: %w", err)
	}

	fileReader, err := os.Open(fileMeta.TmpFilePath)
	if err != nil {
		return fmt.Errorf("failed to read file from disk: %w", err)
	}
	defer fileReader.Close()
	defer os.Remove(fileMeta.TmpFilePath)

	fileMeta.ThumbID = thumbMeta.ID

	// TODO we need to nuke the thumbnail if this fails :(
	fileMeta.Size, fileMeta.UploadDate, err = fs.storage.Upload(ctx, fileMeta.ID, owner.BucketName, fileReader)
	if err != nil {
		return fmt.Errorf("upload failed: %w", err)
	}

	if err := fs.fms.SaveFileMeta(ctx, fileMeta); err != nil {
		rbErr := fs.DeleteFile(ctx, fileMeta.ID, owner)
		if rbErr != nil {
			err = errors.Join(err, fmt.Errorf("CRITICAL - failed to clean up orphaned file '%s': %w", fileMeta.ID, rbErr))
		}
		return fmt.Errorf("failed to save file meta: %w", err)
	}

	return nil
}

func (fs *fileService) DownloadFile(ctx context.Context, id string, owner *user.User) (io.ReadCloser, error) {
	gcsReader, err := fs.storage.Download(ctx, id, owner.BucketName)
	if err != nil {
		return nil, err
	}

	return gcsReader, nil
}

func (fs *fileService) DeleteFile(ctx context.Context, id string, owner *user.User) error {
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

func GenerateThumbnail(ctx context.Context, fm *models.FileMeta) (io.Reader, error) {
	var buf bytes.Buffer

	args := []string{
		"-i", fm.TmpFilePath,
		"-vf", "scale=150:150:force_original_aspect_ratio=increase,crop=150:150",
		"-vframes", "1",
		"-f", "mjpeg",
		"-",
	}

	ffmpegCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ffmpegCtx, "ffmpeg", args...)
	cmd.Stdout = &buf

	if err := cmd.Run(); err != nil {
		return nil, err
	}

	return &buf, nil
}

// func (s *Service) SaveFileMeta(ctx context.Context, fm *models.FileMeta) error {
// 	dbCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
// 	defer cancel()
// 	if err := fms.db.CreateFileMeta(dbCtx, fm); err != nil {
// 		return err
// 	}
// 	return nil
// }

// func (fms *FileMetaService) LookupFileMeta(ctx context.Context, id string, owner *models.User) (*models.FileMeta, error) {
// 	dbCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
// 	defer cancel()
// 	fm, err := fms.db.GetFileMeta(dbCtx, id, owner)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return fm, nil
// }
//
// func (s *Service) ByNameAndOwner(ctx context.Context, name string, owner *models.User) (*models.FileMeta, error) {
// 	dbCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
// 	defer cancel()
// 	fm, err := s.meta.ByNameAndOwner(dbCtx, name, owner)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return fm, nil
// }
//
// func (fms *fileMetaService) LookupAllFileMeta(ctx context.Context, owner *models.User) ([]*models.FileMeta, error) {
// 	dbCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
// 	defer cancel()
// 	data, err := fms.db.GetAllFileMeta(dbCtx, owner)
// 	if err != nil {
// 		return nil, err
// 	}
//
// 	var fm []*models.FileMeta
// 	for _, item := range data {
// 		if item.ParentID == "" {
// 			fm = append(fm, item)
// 		}
// 	}
// 	return fm, nil
// }

// func (fms *fileMetaService) DeleteFileMeta(ctx context.Context, id string, owner *models.User) error {
// 	dbCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
// 	defer cancel()
// 	if err := fms.db.DeleteFileMeta(dbCtx, id, owner); err != nil {
// 		return err
// 	}
// 	return nil
// }
//
