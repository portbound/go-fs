package fs

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
	"time"

	"github.com/portbound/go-fs/internal/middleware"
	"github.com/portbound/go-fs/internal/user"
	"github.com/portbound/go-fs/pkg/response"
)

type Handler struct {
	fileService *Service
	logger      *slog.Logger
}

func NewHandler(f *Service, logger *slog.Logger) *Handler {
	return &Handler{fileService: f, logger: logger}
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
		// TODO: log
		// unauthorized, user not found in context
		response.WriteJSONError(w, http.StatusUnauthorized, "unauthorized: user is missing from request")
		return
	}
	logger = logger.With("requester", requester.Email)

	_, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil {
		response.WriteJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	reader := multipart.NewReader(r.Body, params["boundary"])
	requests := make(chan UploadRequest)
	results := h.fileService.Upload(r.Context(), requests, requester)

	for {
		part, err := reader.NextPart()
		if err != nil {
			if err == io.EOF {
				close(requests)
				break
			}
			// TODO: log
			// failed to parse incoming request
			response.WriteJSONError(w, http.StatusInternalServerError, "request failed")
			return
		}

		requests <- UploadRequest{
			path:        path,
			filename:    filepath.Base(part.FileName()),
			contentType: part.Header.Get("Content-Type"),
		}
	}

	var errs error
	for result := range results {
		if result.err != nil {
			errs = errors.Join(errs, err)
		}
	}

	if errs != nil {
		response.WriteJSON(w, http.StatusMultiStatus, errs.Error())
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

	result, err := h.fileService.Download(r.Context(), id, requester)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			response.WriteJSONError(w, http.StatusNotFound, fmt.Sprintf("file not found for id: '%s'", id))
			return
		}

		response.WriteJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	defer result.reader.Close()

	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(http.StatusOK)

	if _, err := io.Copy(w, reader); err != nil {
		logger.Error("stream file to client", "error", err, "id", id)
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
		logger.Info("unauthorized request: user not found in context")
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
		if errors.Is(err, ErrOrphanedData) {
			logger.Error("error", err)
		}
	}
}
