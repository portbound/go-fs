package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/portbound/go-fs/internal/auth"
	"github.com/portbound/go-fs/internal/response"
	"github.com/portbound/go-fs/internal/services"
)

type LoginRequest struct {
	Token string `json:"token"`
}

type LoginResponse struct {
	JWT string `json:"jwt"`
}

type WebHandler struct {
	userService   *services.UserService
	authenticator *auth.Authenticator
}

func NewWebHandler(a *auth.Authenticator, us *services.UserService) *WebHandler {
	return &WebHandler{authenticator: a, userService: us}
}

func (h *WebHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /login", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./web/public/login.html")
	})
	mux.HandleFunc("POST /login", h.HandleLogin)

}

func (h *WebHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != "POST" {
		response.WriteJSON(w, http.StatusMethodNotAllowed, fmt.Sprintf("Method not allowed: %s", r.Method))
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error decoding request: %v", err)
		response.WriteJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	requesterEmail, err := auth.ValidateOAuth(req.Token)
	if err != nil {
		response.WriteJSONError(w, http.StatusUnauthorized, err.Error())
		return
	}

	_, err = h.userService.GetUser(r.Context(), requesterEmail)
	if err != nil {
		response.WriteJSONError(w, http.StatusForbidden, fmt.Sprintf("Access denied for %s", requesterEmail))
		return
	}

	jwt, err := h.authenticator.GenerateJWT(requesterEmail)
	if err != nil {
		response.WriteJSONError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to provision JWT for %s: %s", requesterEmail, err.Error()))
		return
	}

	response.WriteJSON(w, http.StatusOK, LoginResponse{JWT: jwt})
}
