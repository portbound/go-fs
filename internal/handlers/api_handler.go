// Package handlers
package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
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
	fs  services.FileService
	fms services.FileMetaService
	us  services.UserService
}

func NewAPIHandler(fs services.FileService, fms services.FileMetaService, us services.UserService) *APIHandler {
	return &APIHandler{fs: fs, fms: fms, us: us}
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

	user, ok := h.getUserFromContext(w, r.Context())
	if !ok {
		return
	}

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
			contentType := part.Header.Get("Content-Type")
			metaType := strings.Split(contentType, "/")[0]
			if metaType != "image" && metaType != "video" {
				response.WriteJSONError(w, http.StatusBadRequest, fmt.Sprintf("file type not allowed: %s", metaType))
				return
			}
			id := uuid.New().String()
			path, _, err := h.fs.StageFileToDisk(r.Context(), id, part)
			if err != nil {
				errs = append(errs, fmt.Errorf("[handleUploadFile] failed to stage %s to disk (skipping): %v", part.FileName(), err))
				continue
			}
			defer os.Remove(path)

			batch = append(batch, &models.FileMeta{
				ID:          id,
				Name:        filepath.Base(part.FileName()),
				ContentType: contentType,
				Owner:       user.Email,
				TmpFilePath: path,
			})
		}
	}

	batchErrs := h.fs.ProcessBatch(r.Context(), batch, user)
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
	user, ok := h.getUserFromContext(w, r.Context())
	if !ok {
		return
	}

	dbCtx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	fm, err := h.fms.LookupFileMeta(dbCtx, r.PathValue("id"))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			response.WriteJSONError(w, http.StatusNotFound, err.Error())
			return
		}
		response.WriteJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if user.Email != fm.Owner {
		response.WriteJSONError(w, http.StatusForbidden, fmt.Sprintf("Reqested action denied for %s", user.Email))
		return
	}

	gcsReader, err := h.fs.DownloadFile(r.Context(), fm.ID, user)
	if err != nil {
		response.WriteJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer gcsReader.Close()

	w.Header().Set("Content-Type", fm.ContentType)
	w.WriteHeader(http.StatusOK)

	if _, err := io.Copy(w, gcsReader); err != nil {
		response.WriteJSONError(w, http.StatusInternalServerError, fmt.Sprintf("failed to stream file to client: %v", err))
		return
	}
}

func (h *APIHandler) handleGetFileMeta(w http.ResponseWriter, r *http.Request) {
	user, ok := h.getUserFromContext(w, r.Context())
	if !ok {
		return
	}

	dbCtx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	afm, err := h.fms.LookupAllFileMeta(dbCtx, user)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return
		}
		response.WriteJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	response.WriteJSON(w, http.StatusOK, afm)
}

func (h *APIHandler) handleDeleteFile(w http.ResponseWriter, r *http.Request) {
	user, ok := h.getUserFromContext(w, r.Context())
	if !ok {
		return
	}

	dbCtx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	id := r.PathValue("id")
	fm, err := h.fms.LookupFileMeta(dbCtx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			response.WriteJSONError(w, http.StatusNotFound, fmt.Sprintf("[handleDeleteFile] file not found for id: %s", id))
			return
		}
		response.WriteJSONError(w, http.StatusNotFound, err.Error())
		return
	}

	if user.Email != fm.Owner {
		response.WriteJSONError(w, http.StatusForbidden, fmt.Sprintf("Reqested action denied for %s", user.Email))
		return
	}

	var errs []error
	if fm.ThumbID != "" {
		if err := h.fs.DeleteFile(r.Context(), fm.ThumbID, user); err != nil {
			errs = append(errs, fmt.Errorf("[services.DeleteFile] failed to delete thumbnail for %s: %v", fm.ID, err))
		}
		if err := h.fms.DeleteFileMeta(r.Context(), fm.ThumbID); err != nil {
			errs = append(errs, fmt.Errorf("[services.DeleteFileMeta] failed to delete file meta for %s: %v", fm.ID, err))
		}
	}

	if err := h.fs.DeleteFile(r.Context(), fm.ID, user); err != nil {
		errs = append(errs, fmt.Errorf("[services.DeleteFile] failed to delete file %s: %v", fm.ID, err))
	}
	if err := h.fms.DeleteFileMeta(r.Context(), fm.ID); err != nil {
		errs = append(errs, fmt.Errorf("[services.DeleteFileMeta] failed to delete file meta for %s: %v", fm.ID, err))
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
	user, ok := h.getUserFromContext(w, r.Context())
	if !ok {
		return
	}

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
			fm, err := h.fms.LookupFileMeta(ctx, id)
			if err != nil {
				ch <- fmt.Errorf("[services.DeleteFileMeta] file not found for id %s: %w", id, err)
				return
			}

			if fm.ThumbID != "" {
				if err := h.fs.DeleteFile(ctx, fm.ThumbID, user); err != nil {
					ch <- fmt.Errorf("[services.DeleteFile] failed to delete thumbnail for %s: %v", fm.ID, err)
				}
				if err := h.fms.DeleteFileMeta(ctx, fm.ThumbID); err != nil {
					ch <- fmt.Errorf("[services.DeleteFileMeta] failed to delete file meta for %s: %v", fm.ID, err)
				}
			}

			if err := h.fs.DeleteFile(ctx, fm.ID, user); err != nil {
				ch <- fmt.Errorf("[services.DeleteFile] failed to delete file %s: %v", fm.ID, err)
			}
			if err := h.fms.DeleteFileMeta(ctx, fm.ID); err != nil {
				ch <- fmt.Errorf("[services.DeleteFileMeta] failed to delete file meta for %s: %v", fm.ID, err)
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

func (h *APIHandler) getUserFromContext(w http.ResponseWriter, ctx context.Context) (*models.User, bool) {
	requester, ok := ctx.Value(middleware.RequesterEmailKey).(string)
	if !ok {
		response.WriteJSONError(w, http.StatusUnauthorized, "[getUserFromContext] access denied: no requester in context")
		return nil, false
	}

	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	user, err := h.us.LookupUser(dbCtx, requester)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			response.WriteJSONError(w, http.StatusNotFound, fmt.Sprintf("[getUserFromContext] user not found: %s", requester))
			return nil, false
		}
		response.WriteJSONError(w, http.StatusInternalServerError, fmt.Sprintf("[getUserFromContext] failed to lookup user: %v", err))
		return nil, false
	}

	return user, true
}
