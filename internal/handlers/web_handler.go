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
	"github.com/portbound/go-fs/internal/templates"
	"github.com/portbound/go-fs/internal/templates/components"
	"github.com/portbound/go-fs/internal/utils"
)

type WebHandler struct {
	fs *services.FileService
}

func NewWebHandler(fs *services.FileService) *WebHandler {
	return &WebHandler{fs: fs}
}

func (h *WebHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /files", h.handleUploadFile)
	mux.HandleFunc("GET /files/{id}", h.handleDownloadFile)
	mux.HandleFunc("GET /files", h.handleRenderThumbnails)
	mux.HandleFunc("DELETE /files/{id}", h.handleDeleteFile)
	mux.HandleFunc("POST /files/delete-batch", h.handleDeleteBatch)
}

func (h *WebHandler) handleUploadFile(w http.ResponseWriter, r *http.Request) {
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
			path, err := utils.StageFileToDisk(r.Context(), h.fs.TmpDir, id, part)
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

	batchErrs := h.fs.ProcessBatch(r.Context(), batch)
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
	var ids []string
	for _, item := range batch {
		ids = append(ids, item.ID)
	}
	components.ShowGallery(ids).Render(r.Context(), w)
}

func (h *WebHandler) handleDownloadFile(w http.ResponseWriter, r *http.Request) {
	fm, err := h.fs.LookupFileMeta(r.Context(), r.PathValue("id"))
	if err != nil {
		WriteJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	gcsReader, err := h.fs.GetFile(r.Context(), fm.ID)
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

func (h *WebHandler) handleRenderThumbnails(w http.ResponseWriter, r *http.Request) {
	ids, err := h.fs.GetThumbnailIDs(r.Context())
	if err != nil {
		// render error toast
		return
	}
	templates.HomePage(ids).Render(r.Context(), w)
}

func (h *WebHandler) handleDeleteFile(w http.ResponseWriter, r *http.Request) {
	var errs []error

	fm, err := h.fs.LookupFileMeta(r.Context(), r.PathValue("id"))
	if err != nil {
		WriteJSONError(w, http.StatusNotFound, err.Error())
		return
	}

	if fm.ThumbID != "" {
		if err := h.fs.DeleteFile(r.Context(), fm.ThumbID); err != nil {
			errs = append(errs, fmt.Errorf("services.DeleteFile: failed to delete thumbnail for %s: %v", fm.ID, err))
		}
		if err := h.fs.DeleteFileMeta(r.Context(), fm.ThumbID); err != nil {
			errs = append(errs, fmt.Errorf("services.DeleteFileMeta: failed to delete file meta for %s: %v", fm.ID, err))
		}
	}

	if err := h.fs.DeleteFile(r.Context(), fm.ID); err != nil {
		errs = append(errs, fmt.Errorf("services.DeleteFile: failed to delete file %s: %v", fm.ID, err))
	}
	if err := h.fs.DeleteFileMeta(r.Context(), fm.ID); err != nil {
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

func (h *WebHandler) handleDeleteBatch(w http.ResponseWriter, r *http.Request) {
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
			fm, err := h.fs.LookupFileMeta(ctx, id)
			if err != nil {
				ch <- fmt.Errorf("services.LookupFileMeta: file not found for id %s: %w", id, err)
				return
			}

			if fm.ThumbID != "" {
				if err := h.fs.DeleteFile(ctx, fm.ThumbID); err != nil {
					ch <- fmt.Errorf("services.DeleteFile: failed to delete thumbnail for %s: %v", fm.ID, err)
				}
				if err := h.fs.DeleteFileMeta(ctx, fm.ThumbID); err != nil {
					ch <- fmt.Errorf("services.DeleteFileMeta: failed to delete file meta for %s: %v", fm.ID, err)
				}
			}

			if err := h.fs.DeleteFile(ctx, fm.ID); err != nil {
				ch <- fmt.Errorf("services.DeleteFile: failed to delete file %s: %v", fm.ID, err)
			}
			if err := h.fs.DeleteFileMeta(ctx, fm.ID); err != nil {
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
