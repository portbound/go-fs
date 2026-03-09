package fs

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"mime/multipart"
	"net/http"
	"path/filepath"

	"github.com/portbound/go-fs/internal/middleware"
	"github.com/portbound/go-fs/internal/user"
	"github.com/portbound/go-fs/pkg/response"
)

type Handler struct {
	fs     *Service
	logger *slog.Logger
}

func NewHandler(f *Service, logger *slog.Logger) *Handler {
	return &Handler{fs: f, logger: logger}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /files", h.handleUploadFile)
	mux.HandleFunc("GET /files", h.handleGetMetadata)
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
	results := h.fs.Upload(r.Context(), requests)

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
			Filename:    filepath.Base(part.FileName()),
			ContentType: part.Header.Get("Content-Type"),
			Reader:      part,
			UserId:      requester.Id,
			Bucket:      requester.Bucket,
		}
	}

	var resultErrs error
	for result := range results {
		if result.Err != nil {
			resultErrs = errors.Join(resultErrs, err)
		}
	}

	if resultErrs != nil {
		response.WriteJSON(w, http.StatusMultiStatus, resultErrs.Error())
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

	fileId := r.PathValue("id")
	if fileId == "" {
		response.WriteJSONError(w, http.StatusBadRequest, "file id missing from request")
		return
	}

	request := DownloadRequest{
		FileId: fileId,
		UserId: requester.Id,
		Bucket: requester.Bucket,
	}

	result, err := h.fs.Download(r.Context(), request)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// TODO: log error
			response.WriteJSONError(w, http.StatusNotFound, fmt.Sprintf("file not found for id: %q", fileId))
			return
		}

		if errors.Is(err, ErrMediaCorrupted) {
			// TODO: log error
			// orphaned data needs cleanup
			response.WriteJSONError(w, http.StatusInternalServerError, fmt.Sprintf("%v: %q", err.Error(), fileId))
			return
		}

		response.WriteJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	defer result.Reader.Close()

	w.Header().Set("Content-Type", result.ContentType)
	w.WriteHeader(http.StatusOK)

	if _, err := io.Copy(w, result.Reader); err != nil {
		// TODO: log err
		// failed to stream file to client
		response.WriteJSONError(w, http.StatusInternalServerError, fmt.Sprintf("failed to download file for id: '%s'", fileId))
	}
}

func (h *Handler) handleGetMetadata(w http.ResponseWriter, r *http.Request) {
	logger := h.logger.With("handler", "handleFetchFileMeta")
	requester, ok := r.Context().Value(middleware.RequesterKey).(*user.User)
	if !ok {
		logger.Warn("unauthorized request: user not found in context")
		response.WriteJSONError(w, http.StatusUnauthorized, "unauthorized: user is missing from request")
		return
	}
	logger = logger.With("requester", requester.Email)

	metadata, err := h.fs.GetMetadata(r.Context(), requester.Id)
	if err != nil {
		// TODO: log err
		// failed to retrieve metadata for requester
		response.WriteJSONError(w, http.StatusInternalServerError, fmt.Sprintf("failed to fetch metadata for user %q", requester.Id))
		return
	}

	response.WriteJSON(w, http.StatusOK, metadata)
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

	fileId := r.PathValue("id")
	if fileId == "" {
		response.WriteJSONError(w, http.StatusBadRequest, "file id missing from request")
		return
	}

	request := DeleteRequest{
		FileId: fileId,
		UserId: requester.Id,
		Bucket: requester.Bucket,
	}

	if err := h.fs.Delete(r.Context(), request); err != nil {
		// TODO: log error
		// failed to delete file
		response.WriteJSONError(w, http.StatusInternalServerError, fmt.Sprintf("failed to delete file %q", request.FileId))
		return
	}
}
