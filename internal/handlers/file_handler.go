package handlers

import (
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"

	"github.com/google/uuid"
	"github.com/portbound/go-fs/internal/models"
	"github.com/portbound/go-fs/internal/services"
)

type FileHandler struct {
	fileService *services.FileService
}

func NewFileHandler(fs *services.FileService) *FileHandler {
	return &FileHandler{fileService: fs}
}

func (h *FileHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /files", h.handleFileUpload)
	mux.HandleFunc("GET /files/{id}", h.handleGetFile)
	mux.HandleFunc("GET /files", h.handleGetAllFiles)
	mux.HandleFunc("DELETE /files/{id}", h.handleDeleteFile)
}

func (h *FileHandler) handleFileUpload(w http.ResponseWriter, r *http.Request) {
	_, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil {
		WriteJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	reader := multipart.NewReader(r.Body, params["boundary"])
	batch := []*models.FileMeta{}
	for {
		mp, err := reader.NextPart()
		if err != nil {
			if err == io.EOF {
				break
			}
			WriteJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}

		if mp.FileName() != "" {
			metadata := models.FileMeta{
				Name: mp.FileName(),
				Type: mp.Header.Get("Content-Type"),
			}

			if err := h.fileService.StageFileToDisk(r.Context(), &metadata, mp); err != nil {
				WriteJSONError(w, http.StatusInternalServerError, err.Error())
				return
			}

			batch = append(batch, &metadata)
		}
	}

	errs := h.fileService.ProcessBatch(r.Context(), batch)
	if len(errs) > 0 {
		errMsgs := []string{}
		errMsgs = append(errMsgs, fmt.Sprintf("failed to upload %d file(s)", len(errs)))

		for _, e := range errs {
			errMsgs = append(errMsgs, e.Error())
		}

		WriteJSON(w, http.StatusInternalServerError, errMsgs)
		return
	}

	WriteJSON(w, http.StatusCreated, nil)
}

func (h *FileHandler) handleGetFile(w http.ResponseWriter, r *http.Request)     {}
func (h *FileHandler) handleGetAllFiles(w http.ResponseWriter, r *http.Request) {}
func (h *FileHandler) handleDeleteFile(w http.ResponseWriter, r *http.Request) {
func (h *FileHandler) handleGetFile(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		WriteJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	fm, gcsReader, err := h.fileService.GetFile(r.Context(), id)
	if err != nil {
		WriteJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer gcsReader.Close()

	w.Header().Set("Content-Type", fm.Type)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fm.Name))
	w.WriteHeader(http.StatusOK)

	if _, err := io.Copy(w, gcsReader); err != nil {
		log.Printf("failed to stream file to client: %v", err)
	}
}

		return
	}

	if err := h.fileService.DeleteFileMeta(r.Context(), id); err != nil {
		WriteJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	WriteJSON(w, http.StatusNoContent, nil)
}
