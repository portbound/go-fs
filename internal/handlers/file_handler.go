package handlers

import (
	"bytes"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"time"

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

	// TODO: extract to private function
	reader := multipart.NewReader(r.Body, params["boundary"])
	buf := new(bytes.Buffer)
	metadata := models.FileMetadata{}
	fields := make(map[string]string)

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
			_, err = io.Copy(buf, mp)
			if err != nil {
			}
			metadata.Name = mp.FileName()
			metadata.Type = mp.Header.Get("Content-Type")
		} else {
			fieldName := mp.FormName()
			if fieldName == "" {
				continue
			}

			fieldValue, err := io.ReadAll(mp)
			if err != nil {
				WriteJSONError(w, http.StatusInternalServerError, err.Error())
				return
			}

			fields[fieldName] = string(fieldValue)
		}
	}

	// TODO: make sure we verify that the metadata is complete,
	metadata.Owner = fields["owner"]
	metadata.UploadDate = time.Now()
	if err := h.fileService.UploadFile(buf, &metadata); err != nil {
		WriteJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// TODO: Delete file after pushing to cloud?
	WriteJSON(w, http.StatusCreated, metadata)
}
func (h *FileHandler) handleGetFile(w http.ResponseWriter, r *http.Request)     {}
func (h *FileHandler) handleGetAllFiles(w http.ResponseWriter, r *http.Request) {}
func (h *FileHandler) handleDeleteFile(w http.ResponseWriter, r *http.Request)  {}
