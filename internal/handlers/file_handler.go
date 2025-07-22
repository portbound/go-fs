package handlers

import (
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"strings"

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
	type result struct {
		cloudPath string
		err       error
	}

	_, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil {
		WriteJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	reader := multipart.NewReader(r.Body, params["boundary"])
	allFileMeta := []*models.FileMeta{}
	owner := []byte{}

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

			allFileMeta = append(allFileMeta, &metadata)
		}

		if strings.ToLower(mp.FormName()) == "owner" {
			owner, err = io.ReadAll(mp)
			if err != nil {
				WriteJSONError(w, http.StatusInternalServerError, err.Error())
				return
			}
		}
	}

	for _, f := range allFileMeta {
		f.Owner = string(owner)
		if err := h.fileService.SaveFileMeta(r.Context(), f); err != nil {
			os.Remove(f.TmpDir)
			WriteJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	// Upload to Cloud

	WriteJSON(w, http.StatusCreated, allFileMeta)
}

func (h *FileHandler) handleGetFile(w http.ResponseWriter, r *http.Request)     {}
func (h *FileHandler) handleGetAllFiles(w http.ResponseWriter, r *http.Request) {}
func (h *FileHandler) handleDeleteFile(w http.ResponseWriter, r *http.Request)  {}
