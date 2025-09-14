// Package handlers
package handlers

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/portbound/go-fs/internal/middleware"
	"github.com/portbound/go-fs/internal/models"
	"github.com/portbound/go-fs/internal/services"
	"github.com/portbound/go-fs/pkg/response"
)

type APIHandler struct {
	fs     services.FileService
	fms    services.FileMetaService
	us     services.UserService
	logger *slog.Logger
}

func NewAPIHandler(fs services.FileService, fms services.FileMetaService, us services.UserService, logger *slog.Logger) *APIHandler {
	return &APIHandler{fs: fs, fms: fms, us: us, logger: logger}
}

func (h *APIHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /files", h.handleUploadFile)
	mux.HandleFunc("GET /files", h.handleFetchFileMeta)
	mux.HandleFunc("GET /files/{id}", h.handleDownloadFile)
	mux.HandleFunc("DELETE /files/{id}", h.handleDeleteFile)
}

func (h *APIHandler) handleUploadFile(w http.ResponseWriter, r *http.Request) {
	logger := h.logger.With("handler", "handleUploadFile")
	requester, ok := r.Context().Value(middleware.RequesterKey).(*models.User)
	if !ok {
		logger.Error("unauthorized request: user not found in context")
		response.WriteJSONError(w, http.StatusUnauthorized, "unauthorized: user is missing from request")
		return
	}
	logger = logger.With("requester", requester.Email)

	_, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil {
		response.WriteJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	var batchErrs error
	var wg sync.WaitGroup
	ch := make(chan error)
	reader := multipart.NewReader(r.Body, params["boundary"])
	for {
		part, err := reader.NextPart()
		if err != nil {
			if err == io.EOF {
				break
			}
			logger.Error("failed to parse incoming request", "error", err)
			response.WriteJSONError(w, http.StatusInternalServerError, "request failed")
			return
		}

		name := filepath.Base(part.FileName())
		contentType := part.Header.Get("Content-Type")
		metaType := strings.Split(contentType, "/")[0]
		if metaType != "image" && metaType != "video" {
			logger.Error("unsupported file type", "file_name", name)
			batchErrs = errors.Join(batchErrs, fmt.Errorf("unsupported file type: '%s'", name))
			continue
		}

		id := uuid.New().String()
		path, _, err := h.fs.StageFileToDisk(r.Context(), id, part)
		if err != nil {
			logger.Error("failed to stage file to disk", "error", err, "id", id, "file_name", name)
			batchErrs = errors.Join(batchErrs, fmt.Errorf("failed to upload file: '%s'", name))
			continue
		}
		part.Close()

		fm := models.FileMeta{
			ID:          id,
			Name:        name,
			ContentType: contentType,
			Owner:       requester.Email,
			TmpFilePath: path,
		}

		wg.Go(func() {
			if err := h.fs.ProcessFile(r.Context(), &fm, requester); err != nil {
				logger.Error("file processing failed", "error", err, "id", id, "file_name", fm.Name)
				select {
				case ch <- err:
				case <-r.Context().Done():
				}
			}
		})
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	for err := range ch {
		batchErrs = errors.Join(batchErrs, err)
	}

	if batchErrs != nil {
		err = errors.Join(err, batchErrs)
		response.WriteJSON(w, http.StatusMultiStatus, err)
		return
	}

	response.WriteJSON(w, http.StatusCreated, nil)
}

func (h *APIHandler) handleDownloadFile(w http.ResponseWriter, r *http.Request) {
	logger := h.logger.With("handler", "handleDownloadFile")
	requester, ok := r.Context().Value(middleware.RequesterKey).(*models.User)
	if !ok {
		logger.Warn("unauthorized request: user not found in context")
		response.WriteJSONError(w, http.StatusUnauthorized, "unauthorized: user is missing from request")
		return
	}
	logger = logger.With("requester", requester.Email)

	id := r.PathValue("id")
	if id == "" {
		response.WriteJSONError(w, http.StatusBadRequest, "file id missing from request")
		return
	}

	dbCtx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	fm, err := h.fms.LookupFileMeta(dbCtx, id, requester)
	if err != nil {
		logger.Error("failed to fetch metadata for file", "error", err, "id", id)
		if errors.Is(err, sql.ErrNoRows) {
			response.WriteJSONError(w, http.StatusNotFound, fmt.Sprintf("file not found for id: '%s'", id))
			return
		}
		response.WriteJSONError(w, http.StatusInternalServerError, fmt.Sprintf("failed to download file for id: '%s'", id))
		return
	}

	if requester.Email != fm.Owner {
		response.WriteJSONError(w, http.StatusForbidden, fmt.Sprintf("permission denied for user: '%s'", requester.Email))
		return
	}

	fileReader, err := h.fs.DownloadFile(r.Context(), fm.ID, requester)
	if err != nil {
		logger.Error("failed to download file from storage", "error", err, "id", id)
		response.WriteJSONError(w, http.StatusInternalServerError, fmt.Sprintf("failed to download file for id: '%s'", id))
		return
	}
	defer fileReader.Close()

	w.Header().Set("Content-Type", fm.ContentType)
	w.WriteHeader(http.StatusOK)

	if _, err := io.Copy(w, fileReader); err != nil {
		logger.Error("failed to stream file to client", "error", err, "id", id)
		response.WriteJSONError(w, http.StatusInternalServerError, fmt.Sprintf("failed to download file for id: '%s'", id))
	}
}

func (h *APIHandler) handleFetchFileMeta(w http.ResponseWriter, r *http.Request) {
	logger := h.logger.With("handler", "handleFetchFileMeta")
	requester, ok := r.Context().Value(middleware.RequesterKey).(*models.User)
	if !ok {
		logger.Warn("unauthorized request: user not found in context")
		response.WriteJSONError(w, http.StatusUnauthorized, "unauthorized: user is missing from request")
		return
	}
	logger = logger.With("requester", requester.Email)

	dbCtx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	afm, err := h.fms.LookupAllFileMeta(dbCtx, requester)
	if err != nil {
		logger.Error("failed to fetch metadata for requester", "error", err)
		if errors.Is(err, sql.ErrNoRows) {
			return
		}
		response.WriteJSONError(w, http.StatusInternalServerError, "failed to fetch file metadata")
		return
	}

	response.WriteJSON(w, http.StatusOK, afm)
}

func (h *APIHandler) handleDeleteFile(w http.ResponseWriter, r *http.Request) {
	logger := h.logger.With("handler", "handleDeleteFile")
	requester, ok := r.Context().Value(middleware.RequesterKey).(*models.User)
	if !ok {
		logger.Warn("unauthorized request: user not found in context")
		response.WriteJSONError(w, http.StatusUnauthorized, "unauthorized: user is missing from request")
		return
	}
	logger = logger.With("requester", requester.Email)

	id := r.PathValue("id")
	if id == "" {
		response.WriteJSONError(w, http.StatusBadRequest, "file id missing from request")
		return
	}

	dbCtx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	fm, err := h.fms.LookupFileMeta(dbCtx, id, requester)
	if err != nil {
		logger.Error("failed to fetch metadata for file", "error", err, "id", id)
		if errors.Is(err, sql.ErrNoRows) {
			response.WriteJSONError(w, http.StatusNotFound, fmt.Sprintf("file not found for id: '%s'", id))
			return
		}
		response.WriteJSONError(w, http.StatusInternalServerError, fmt.Sprintf("failed to delete file for id: '%s'", id))
		return
	}

	if requester.Email != fm.Owner {
		response.WriteJSONError(w, http.StatusForbidden, fmt.Sprintf("permission denied for user: '%s'", requester.Email))
		return
	}

	var errs error
	// TODO need to come up with a less brittle implementation to call here using the saga pattern or some sort of transaction
	if err := h.fs.DeleteFile(r.Context(), fm.ThumbID, requester); err != nil {
		errs = errors.Join(errs, err)
	}

	if err := h.fms.DeleteFileMeta(r.Context(), fm.ThumbID, requester); err != nil {
		errs = errors.Join(errs, err)
	}

	if err := h.fs.DeleteFile(r.Context(), fm.ID, requester); err != nil {
		errs = errors.Join(errs, err)
	}

	if err := h.fms.DeleteFileMeta(r.Context(), fm.ID, requester); err != nil {
		errs = errors.Join(errs, err)
	}

	if errs != nil {
		logger.Error("failed to fully delete file", "error", errs, "id", id)
		response.WriteJSONError(w, http.StatusMultiStatus, "failed to fully delete file for id: '%s'")
		return
	}

	response.WriteJSON(w, http.StatusNoContent, nil)
}
