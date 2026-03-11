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
	"github.com/portbound/portlog"
)

type Handler struct {
	fs     *Service
	logger *portlog.PortLog
}

func NewHandler(f *Service, l *portlog.PortLog) *Handler {
	return &Handler{fs: f, logger: l}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /files", h.handleUploadFile)
	mux.HandleFunc("GET /files", h.handleGetMetadata)
	mux.HandleFunc("GET /files/{id}", h.handleDownloadFile)
	mux.HandleFunc("DELETE /files/{id}", h.handleDeleteFile)
}

func (h *Handler) handleUploadFile(w http.ResponseWriter, r *http.Request) {
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
			msg := "failed to parse incoming multipart request"
			h.logger.Error(msg, err)
			response.WriteJSONError(w, http.StatusInternalServerError, errors.New(msg))
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
			h.logger.Error("file not found during download", err, "fileId", fileId, "userId", requester.Id)
			response.WriteJSONError(w, http.StatusNotFound, fmt.Errorf("file not found for id: %q", fileId))
			return
		}

		if errors.Is(err, ErrMediaCorrupted) {
			h.logger.Error("orphaned data needs cleanup", err, "fileId", fileId)
			response.WriteJSONError(w, http.StatusInternalServerError, fmt.Errorf("%v: %q", err.Error(), fileId))
			return
		}

		response.WriteJSONError(w, http.StatusBadRequest, err.Error())
		h.logger.Error("failed to download file", err, "fileId", fileId)
		return
	}
	defer result.Reader.Close()

	w.Header().Set("Content-Type", result.ContentType)
	w.WriteHeader(http.StatusOK)

	if _, err := io.Copy(w, result.Reader); err != nil {
		h.logger.Error("failed to stream file to client", err, "fileId", fileId)
		response.WriteJSONError(w, http.StatusInternalServerError, fmt.Errorf("failed to download file for id: '%s'", fileId))
	}
}

func (h *Handler) handleGetMetadata(w http.ResponseWriter, r *http.Request) {
	metadata, err := h.fs.GetMetadata(r.Context(), requester.Id)
	if err != nil {
		h.logger.Error("failed to retrieve metadata", err, "userId", requester.Id)
		response.WriteJSONError(w, http.StatusInternalServerError, fmt.Errorf("failed to fetch metadata for user %q", requester.Id))
		return
	}

	response.WriteJSON(w, http.StatusOK, metadata)
}

func (h *Handler) handleDeleteFile(w http.ResponseWriter, r *http.Request) {
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
		h.logger.Error("failed to delete file", err, "fileId", fileId, "userId", requester.Id)
		response.WriteJSONError(w, http.StatusInternalServerError, fmt.Errorf("failed to delete file %q", request.FileId))
		return
	}
}
