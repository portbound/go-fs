package handlers

import (
	"net/http"
)

type WebHandler struct{}

func NewWebHandler() *WebHandler {
	return &WebHandler{}
}

func (h *WebHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.Handle("/", http.FileServer(http.Dir("./web/public")))
	mux.HandleFunc("/login", h.handleLogin)
}

func (h *WebHandler) handleLogin(w http.ResponseWriter, r *http.Request) {}
