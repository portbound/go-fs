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
	requestingUser := []byte{}

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
			requestingUser, err = io.ReadAll(mp)
			if err != nil {
				WriteJSONError(w, http.StatusInternalServerError, err.Error())
				return
			}
		}
	}

	for _, fm := range allFileMeta {
		// NOTE: I think what needs to happen here is that we fire off a go routine for each potential file. We will need a channel with a max capacity of say 5, and then we can write the results (make a struct containing the id and err) to it. When all of the goroutines have finished, i.e. use a wait group, we can then check to see if any of the processes returned an error, and then Write our JSON err with that information
		fm.Owner = string(requestingUser)
		if err := h.fileService.SaveFileMeta(r.Context(), fm); err != nil {
			// WriteJSONError(w, http.StatusInternalServerError, err.Error())
			// return
		}
		// Upload to Cloud
		if err := h.fileService.UploadToCloud(r.Context(), fm); err != nil {

		}

		// if successful, delete from local storage
		os.Remove(fm.TmpDir)
	}

	WriteJSON(w, http.StatusCreated, allFileMeta)
}

func (h *FileHandler) handleGetFile(w http.ResponseWriter, r *http.Request)     {}
func (h *FileHandler) handleGetAllFiles(w http.ResponseWriter, r *http.Request) {}
func (h *FileHandler) handleDeleteFile(w http.ResponseWriter, r *http.Request)  {}
