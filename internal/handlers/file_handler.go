package handlers

import (
	"net/http"

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
	mux.HandleFunc("PUT /files/{id}", h.handleUpdateFile)
	mux.HandleFunc("DELETE /files/{id}", h.handleDeleteFile)
}

func (h *FileHandler) handleFileUpload(w http.ResponseWriter, r *http.Request)  {}
func (h *FileHandler) handleGetFile(w http.ResponseWriter, r *http.Request)     {}
func (h *FileHandler) handleGetAllFiles(w http.ResponseWriter, r *http.Request) {}
func (h *FileHandler) handleUpdateFile(w http.ResponseWriter, r *http.Request)  {}
func (h *FileHandler) handleDeleteFile(w http.ResponseWriter, r *http.Request)  {}
