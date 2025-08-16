// Package handlers
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/google/uuid"
	"github.com/portbound/go-fs/internal/models"
	"github.com/portbound/go-fs/internal/services"
	"github.com/portbound/go-fs/internal/utils"
)

type FileHandler struct {
	fileService *services.FileService
}

func NewFileHandler(fs *services.FileService) *FileHandler {
	return &FileHandler{fileService: fs}
}

func (fh *FileHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /files", fh.handleUploadFile)
	mux.HandleFunc("GET /files/{id}", fh.handleDownloadFile)
	mux.HandleFunc("GET /files", fh.handleGetThumbnailIDs)
	mux.HandleFunc("DELETE /files/{id}", fh.handleDeleteFile)
	mux.HandleFunc("POST /files/delete-batch", fh.handleDeleteBatch)
}

func (fh *FileHandler) handleUploadFile(w http.ResponseWriter, r *http.Request) {
	var errs []error
	var batch []*models.FileMeta

	_, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil {
		WriteJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	reader := multipart.NewReader(r.Body, params["boundary"])
	for {
		part, err := reader.NextPart()
		if err != nil {
			if err == io.EOF {
				break
			}
			WriteJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}
		defer part.Close()

		if part.FileName() != "" {
			id := uuid.New().String()
			path, err := utils.StageFileToDisk(r.Context(), fh.fileService.TmpDir, id, part)
			if err != nil {
				errs = append(errs, fmt.Errorf("handleUploadFile: StageFileToDisk() failed: %v - skipping: %s", err, part.FileName()))
				continue
			}
			defer os.Remove(path)

			batch = append(batch, &models.FileMeta{
				ID:          id,
				ParentID:    "",
				ThumbID:     "",
				Name:        filepath.Base(part.FileName()),
				Owner:       "me",
				ContentType: part.Header.Get("Content-Type"),
				TmpFilePath: path,
			})
		}
	}

	batchErrs := fh.fileService.ProcessBatch(r.Context(), batch)
	if batchErrs != nil {
		errs = append(errs, batchErrs...)
	}

	if len(errs) > 0 {
		var errMessages []string
		errMessages = append(errMessages, fmt.Sprintf("failed to upload %d file(s)", len(errs)))
		for _, err := range errs {
			errMessages = append(errMessages, err.Error())
		}

		WriteJSON(w, http.StatusMultiStatus, errMessages)
		return
	}

	WriteJSON(w, http.StatusCreated, nil)
}

func (fh *FileHandler) handleDownloadFile(w http.ResponseWriter, r *http.Request) {
	fm, err := fh.fileService.LookupFileMeta(r.Context(), r.PathValue("id"))
	if err != nil {
		WriteJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	gcsReader, err := fh.fileService.GetFile(r.Context(), fm.ID)
	if err != nil {
		WriteJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer gcsReader.Close()

	w.Header().Set("Content-Type", fm.ContentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fm.Name))
	w.WriteHeader(http.StatusOK)

	if _, err := io.Copy(w, gcsReader); err != nil {
		log.Printf("failed to stream file to client: %v", err)
	}
}

func (fh *FileHandler) handleGetThumbnailIDs(w http.ResponseWriter, r *http.Request) {
	fileNames, err := fh.fileService.GetThumbnails(r.Context())
	if err != nil {
		WriteJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, fileNames)
}

func (fh *FileHandler) handleDeleteFile(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	fm, err := fh.fileService.LookupFileMeta(r.Context(), id)
	if err != nil {
		WriteJSONError(w, http.StatusNotFound, err.Error())
		return
	}

	var errs []error
	if err := fh.fileService.DeleteFile(r.Context(), fm.ID); err != nil {
		errs = append(errs, fmt.Errorf("services.DeleteFile: failed to delete file %s: %v", fm.ID, err))
	}

	if fm.ThumbID != "" {
		if err := fh.fileService.DeleteFile(r.Context(), fm.ThumbID); err != nil {
			errs = append(errs, fmt.Errorf("services.DeleteFile: failed to delete thumbnail for %s: %v", fm.ID, err))
		}

		if err := fh.fileService.DeleteFileMeta(r.Context(), fm.ThumbID); err != nil {
			errs = append(errs, fmt.Errorf("services.DeleteFileMeta: failed to delete file meta for %s: %v", fm.ID, err))
		}

	}

	if err := fh.fileService.DeleteFileMeta(r.Context(), fm.ID); err != nil {
		errs = append(errs, fmt.Errorf("services.DeleteFileMeta: failed to delete file meta for %s: %v", fm.ID, err))
	}

	if len(errs) > 0 {
		var errMessages []string
		for _, err := range errs {
			errMessages = append(errMessages, err.Error())
		}
		WriteJSON(w, http.StatusMultiStatus, errMessages)
		return
	}

	WriteJSON(w, http.StatusNoContent, nil)
}

func (fh *FileHandler) handleDeleteBatch(w http.ResponseWriter, r *http.Request) {
	var ids []string
	if err := json.NewDecoder(r.Body).Decode(&ids); err != nil {
		WriteJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	var wg sync.WaitGroup
	ch := make(chan error, len(ids))
	ctx := context.Background()
	for _, id := range ids {
		wg.Add(1)
		go func(id string) {
			fm, err := fh.fileService.LookupFileMeta(ctx, id)
			if err != nil {
				ch <- fmt.Errorf("services.LookupFileMeta: file not found for id %s: %w", id, err)
				return
			}

			if err := fh.fileService.DeleteFile(ctx, fm.ID); err != nil {
				ch <- fmt.Errorf("services.DeleteFile: failed to delete file %s: %v", fm.ID, err)
			}

			if fm.ThumbID != "" {
				if err := fh.fileService.DeleteFile(ctx, fm.ThumbID); err != nil {
					ch <- fmt.Errorf("services.DeleteFile: failed to delete thumbnail for %s: %v", fm.ID, err)
				}

				if err := fh.fileService.DeleteFileMeta(ctx, fm.ThumbID); err != nil {
					ch <- fmt.Errorf("services.DeleteFileMeta: failed to delete file meta for %s: %v", fm.ID, err)
				}
			}

			if err := fh.fileService.DeleteFileMeta(ctx, fm.ID); err != nil {
				ch <- fmt.Errorf("services.DeleteFileMeta: failed to delete file meta for %s: %v", fm.ID, err)
			}
		}(id)
	}

	go func() {
		wg.Wait()
		if len(ch) > 0 {
			var errMessages []string
			for err := range ch {
				errMessages = append(errMessages, err.Error())
			}
			WriteJSON(w, http.StatusMultiStatus, errMessages)
			return
		}
	}()

	WriteJSON(w, http.StatusNoContent, nil)
}
