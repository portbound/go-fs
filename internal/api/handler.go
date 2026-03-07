// Package handlers
package api

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
	"strings"
	"sync"
	"time"

	"github.com/portbound/go-fs/internal/fs"
	"github.com/portbound/go-fs/internal/middleware"
	"github.com/portbound/go-fs/internal/user"
	"github.com/portbound/go-fs/pkg/response"
)

type Handler struct {
	fileService fs.Service
	userService user.Service
	logger      *slog.Logger
}

func New(f fs.Service, u user.Service, logger *slog.Logger) *Handler {
	return &Handler{fileService: f, userService: u, logger: logger}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /files", h.handleUploadFile)
	mux.HandleFunc("GET /files", h.handleFetchFileMeta)
	mux.HandleFunc("GET /files/{id}", h.handleDownloadFile)
	mux.HandleFunc("DELETE /files/{id}", h.handleDeleteFile)
}

func (h *Handler) handleUploadFile(w http.ResponseWriter, r *http.Request) {
	logger := h.logger.With("handler", "handleUploadFile")
	requester, ok := r.Context().Value(middleware.RequesterKey).(*user.User)
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

		contentType := part.Header.Get("Content-Type")
		metaType := strings.Split(contentType, "/")[0]
		if metaType != "image" && metaType != "video" {
			logger.Error("unsupported file type", "file_name", name)
			batchErrs = errors.Join(batchErrs, fmt.Errorf("unsupported file type: '%s'", name))
			continue
		}

		wg.Go(func() {
			if err := h.fileService.Upload(r.Context, part, metadata, part, requester.Email); err != nil {
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
		// err = errors.Join(err, batchErrs)
		response.WriteJSON(w, http.StatusMultiStatus, batchErrs.Error())
		return
	}

	response.WriteJSON(w, http.StatusCreated, nil)
}

func (h *Handler) handleDownloadFile(w http.ResponseWriter, r *http.Request) {
	logger := h.logger.With("handler", "handleDownloadFile")
	requester, ok := r.Context().Value(middleware.RequesterKey).(*user.User)
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

	fileReader, err := h.fileService.DownloadFile(r.Context(), fm.ID, requester)
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

func (h *Handler) handleFetchFileMeta(w http.ResponseWriter, r *http.Request) {
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

func (h *Handler) handleDeleteFile(w http.ResponseWriter, r *http.Request) {
	logger := h.logger.With("handler", "handleDeleteFile")
	requester, ok := r.Context().Value(middleware.RequesterKey).(*user.User)
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

	if err := h.fileService.Delete(r.Context(), id, requester); err != nil {
	}
}
