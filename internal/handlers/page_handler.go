package handlers

import (
	"net/http"

	"github.com/portbound/go-fs/internal/services"
	"github.com/portbound/go-fs/internal/templates"
)

type PageHandler struct {
	fs *services.FileService
}

func NewPageHandler(fs *services.FileService) *PageHandler {
	return &PageHandler{fs: fs}
}

func (h *PageHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /", h.renderHomepage)
	mux.HandleFunc("GET /upload", h.renderUploadpage)

	mux.HandleFunc("POST /upload", h.handleFileUpload)
}

func (h *PageHandler) renderHomepage(w http.ResponseWriter, r *http.Request) {
	templates.HomePage().Render(r.Context(), w)
}

func (h *PageHandler) renderUploadpage(w http.ResponseWriter, r *http.Request) {
	templates.UploadPage().Render(r.Context(), w)
}

func (h *PageHandler) renderListpage(w http.ResponseWriter, r *http.Request) {
	thumbnailsIDs, err := h.fs.GetThumbnailIDs(r.Context())
	if err != nil {
	}

}
func (h *PageHandler) handleFileUpload(w http.ResponseWriter, r *http.Request) {}
