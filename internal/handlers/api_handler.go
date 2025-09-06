// Package handlers
package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/portbound/go-fs/internal/models"
	"github.com/portbound/go-fs/internal/services"
	"github.com/portbound/go-fs/pkg/response"
)

type APIHandler struct {
	fileService     services.FileService
	fileMetaService services.FileMetaService
}

func NewAPIHandler(fs services.FileService, fms services.FileMetaService) *APIHandler {
	return &APIHandler{fileService: fs, fileMetaService: fms}
}

func (h *APIHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /files", h.handleGetFileMeta)
	mux.HandleFunc("POST /files", h.handleUploadFile)
	mux.HandleFunc("GET /files/{id}", h.handleDownloadFile)
	mux.HandleFunc("DELETE /files/{id}", h.handleDeleteFile)
	mux.HandleFunc("POST /files/delete-batch", h.handleDeleteBatch)
}

func (h *APIHandler) handleUploadFile(w http.ResponseWriter, r *http.Request) {
	var errs []error
	var batch []*models.FileMeta

	_, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil {
		response.WriteJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	reader := multipart.NewReader(r.Body, params["boundary"])
	for {
		part, err := reader.NextPart()
		if err != nil {
			if err == io.EOF {
				break
			}
			response.WriteJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}
		defer part.Close()

		if part.FileName() != "" {
			id := uuid.New().String()
			path, bytesWritten, err := h.fileService.StageFileToDisk(r.Context(), id, part)
			if err != nil {
				errs = append(errs, fmt.Errorf("handleUploadFile: StageFileToDisk() failed: %v - skipping: %s", err, part.FileName()))
				continue
			}
			defer os.Remove(path)

			batch = append(batch, &models.FileMeta{
				ID:          id,
				Name:        filepath.Base(part.FileName()),
				ContentType: part.Header.Get("Content-Type"),
				Size:        bytesWritten,
				UploadDate:  time.Now(),
				Owner:       "me",
				TmpFilePath: path,
			})
		}
	}

	batchErrs := h.fileService.ProcessBatch(r.Context(), batch)
	if batchErrs != nil {
		errs = append(errs, batchErrs...)
	}

	if len(errs) > 0 {
		var errMessages []string
		errMessages = append(errMessages, fmt.Sprintf("failed to upload %d file(s)", len(errs)))
		for _, err := range errs {
			errMessages = append(errMessages, err.Error())
		}
		response.WriteJSON(w, http.StatusMultiStatus, errMessages)
		return
	}
	var fm []*models.FileMeta
	for _, item := range batch {
		fm = append(fm, item)
	}
	response.WriteJSON(w, http.StatusCreated, nil)
}

func (h *APIHandler) handleDownloadFile(w http.ResponseWriter, r *http.Request) {
	fm, err := h.fileMetaService.LookupFileMeta(r.Context(), r.PathValue("id"))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			response.WriteJSONError(w, http.StatusNotFound, err.Error())
			return
		}
		response.WriteJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	gcsReader, err := h.fileService.DownloadFile(r.Context(), fm.ID)
	if err != nil {
		response.WriteJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer gcsReader.Close()

	w.Header().Set("Content-Type", fm.ContentType)
	w.WriteHeader(http.StatusOK)

	if _, err := io.Copy(w, gcsReader); err != nil {
		log.Printf("failed to stream file to client: %v", err)
	}
}

func (h *APIHandler) handleGetFileMeta(w http.ResponseWriter, r *http.Request) {
	afm, err := h.fileMetaService.LookupAllFileMeta(r.Context())
	if err != nil {
		response.WriteJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.WriteJSON(w, http.StatusOK, afm)
}

func (h *APIHandler) handleDeleteFile(w http.ResponseWriter, r *http.Request) {
	var errs []error

	fm, err := h.fileMetaService.LookupFileMeta(r.Context(), r.PathValue("id"))
	if err != nil {
		response.WriteJSONError(w, http.StatusNotFound, err.Error())
		return
	}

	if fm.ThumbID != "" {
		if err := h.fileService.DeleteFile(r.Context(), fm.ThumbID); err != nil {
			errs = append(errs, fmt.Errorf("services.DeleteFile: failed to delete thumbnail for %s: %v", fm.ID, err))
		}
		if err := h.fileMetaService.DeleteFileMeta(r.Context(), fm.ThumbID); err != nil {
			errs = append(errs, fmt.Errorf("services.DeleteFileMeta: failed to delete file meta for %s: %v", fm.ID, err))
		}
	}

	if err := h.fileService.DeleteFile(r.Context(), fm.ID); err != nil {
		errs = append(errs, fmt.Errorf("services.DeleteFile: failed to delete file %s: %v", fm.ID, err))
	}
	if err := h.fileMetaService.DeleteFileMeta(r.Context(), fm.ID); err != nil {
		errs = append(errs, fmt.Errorf("services.DeleteFileMeta: failed to delete file meta for %s: %v", fm.ID, err))
	}

	if len(errs) > 0 {
		var errMessages []string
		for _, err := range errs {
			errMessages = append(errMessages, err.Error())
		}
		response.WriteJSON(w, http.StatusMultiStatus, errMessages)
		return
	}
	response.WriteJSON(w, http.StatusNoContent, nil)
}

func (h *APIHandler) handleDeleteBatch(w http.ResponseWriter, r *http.Request) {
	var ids []string
	if err := json.NewDecoder(r.Body).Decode(&ids); err != nil {
		response.WriteJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	var wg sync.WaitGroup
	ch := make(chan error, len(ids))
	ctx := context.Background()
	for _, id := range ids {
		wg.Add(1)
		go func(id string) {
			fm, err := h.fileMetaService.LookupFileMeta(ctx, id)
			if err != nil {
				ch <- fmt.Errorf("services.LookupFileMeta: file not found for id %s: %w", id, err)
				return
			}

			if fm.ThumbID != "" {
				if err := h.fileService.DeleteFile(ctx, fm.ThumbID); err != nil {
					ch <- fmt.Errorf("services.DeleteFile: failed to delete thumbnail for %s: %v", fm.ID, err)
				}
				if err := h.fileMetaService.DeleteFileMeta(ctx, fm.ThumbID); err != nil {
					ch <- fmt.Errorf("services.DeleteFileMeta: failed to delete file meta for %s: %v", fm.ID, err)
				}
			}

			if err := h.fileService.DeleteFile(ctx, fm.ID); err != nil {
				ch <- fmt.Errorf("services.DeleteFile: failed to delete file %s: %v", fm.ID, err)
			}
			if err := h.fileMetaService.DeleteFileMeta(ctx, fm.ID); err != nil {
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
			response.WriteJSON(w, http.StatusMultiStatus, errMessages)
			return
		}
	}()
	response.WriteJSON(w, http.StatusNoContent, nil)
}
