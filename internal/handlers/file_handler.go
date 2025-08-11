// Package handlers
package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime"
	"mime/multipart"
	"net/http"
	"path/filepath"

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

func (h *FileHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /files", h.handleUploadFile)
	mux.HandleFunc("GET /files/{id}", h.handleDownloadFile)
	mux.HandleFunc("GET /files", h.handleGetFileIds)
	mux.HandleFunc("DELETE /files/{id}", h.handleDeleteFile)
	mux.HandleFunc("POST /files/delete-batch", h.handleDeleteBatch)
}

func (h *FileHandler) handleUploadFile(w http.ResponseWriter, r *http.Request) {
	_, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil {
		WriteJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	reader := multipart.NewReader(r.Body, params["boundary"])
	batch := []*models.FileMeta{}
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
			fm := models.FileMeta{
				ID:          uuid.New().String(),
				Name:        filepath.Base(part.FileName()),
				ContentType: part.Header.Get("Content-Type"),
			}

			path, err := utils.StageFileToDisk(r.Context(), h.fileService.TmpStorage, fm.ID, part)
			if err != nil {
				WriteJSONError(w, http.StatusInternalServerError, err.Error())
				return
			}

			fm.TmpFilePath = path
			batch = append(batch, &fm)
		}
	}

	errs := h.fileService.ProcessBatch(r.Context(), batch)
	if len(errs) > 0 {
		errMsgs := []string{}
		errMsgs = append(errMsgs, fmt.Sprintf("failed to upload %d file(s)", len(errs)))

		for _, err := range errs {
			errMsgs = append(errMsgs, err.Error())
		}

		WriteJSON(w, http.StatusOK, errMsgs)
		return
	}

	WriteJSON(w, http.StatusOK, nil)
}

func (h *FileHandler) handleDownloadFile(w http.ResponseWriter, r *http.Request) {
	fm, err := h.fileService.LookupFileMeta(r.Context(), r.PathValue("id"))
	if err != nil {
		WriteJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	gcsReader, err := h.fileService.GetFile(r.Context(), fm.ID)
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

func (h *FileHandler) handleGetFileIds(w http.ResponseWriter, r *http.Request) {
	fileNames, err := h.fileService.GetThumbnails(r.Context())
	if err != nil {
		WriteJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, fileNames)
}

func (h *FileHandler) handleDeleteFile(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		WriteJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.fileService.DeleteFile(r.Context(), id.String()); err != nil {
		WriteJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusNoContent, nil)
}

func (h *FileHandler) handleDeleteBatch(w http.ResponseWriter, r *http.Request) {
	var batch struct {
		IDs []uuid.UUID `json:"ids"`
	}

	if err := json.NewDecoder(r.Body).Decode(&batch); err != nil {
		WriteJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	errs := h.fileService.DeleteBatch(r.Context(), &batch.IDs)
	if len(errs) > 0 {
		errMsgs := []string{}
		errMsgs = append(errMsgs, fmt.Sprintf("failed to delete %d file(s)", len(errs)))

		for _, err := range errs {
			errMsgs = append(errMsgs, err.Error())
		}

		WriteJSON(w, http.StatusInternalServerError, errMsgs)
		return
	}
	WriteJSON(w, http.StatusOK, nil)
}
