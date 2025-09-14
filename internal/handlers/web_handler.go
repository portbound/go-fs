package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/portbound/go-fs/internal/services"
	"github.com/portbound/go-fs/pkg/auth"
	"github.com/portbound/go-fs/pkg/response"
)

type LoginRequest struct {
	Token string `json:"token"`
}

type LoginResponse struct {
	JWT string `json:"jwt"`
}

type WebHandler struct {
	userService   services.UserService
	authenticator *auth.Authenticator
	logger        *slog.Logger
}

func NewWebHandler(a *auth.Authenticator, us services.UserService, logger *slog.Logger) *WebHandler {
	return &WebHandler{authenticator: a, userService: us, logger: logger}
}

func (h *WebHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /login", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./web/public/login.html")
	})
	mux.HandleFunc("POST /login", h.HandleLogin)

}

func (h *WebHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	logger := h.logger.With("handler", "handleLogin")

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
		logger.Error("failed to decode request data", "error", err)
		response.WriteJSONError(w, http.StatusInternalServerError, "login attempt failed")
		return
	}

	requesterEmail, err := h.authenticator.ValidateOAuth(req.Token)
	if err != nil {
		logger.Error("oAuth validation failed", "error", err)
		response.WriteJSONError(w, http.StatusInternalServerError, "login attempt failed")
		return
	}

	_, err = h.userService.LookupUser(r.Context(), requesterEmail)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			logger.Error("unauthorized login attempt", "error", err, "requester", requesterEmail)
			response.WriteJSONError(w, http.StatusForbidden, fmt.Sprintf("permission denied for user: '%s'", requesterEmail))
		}
		logger.Error("user lookup failed", "error", err)
		response.WriteJSONError(w, http.StatusInternalServerError, "login attempt failed")
		return
	}

	expirationDate := time.Now().UTC().AddDate(0, 30, 0)
	jwt, err := h.authenticator.GenerateJWT(expirationDate, requesterEmail)
	if err != nil {
		logger.Error("failed to provision JWT", "error", err, "requester", requesterEmail)
		response.WriteJSONError(w, http.StatusInternalServerError, "login attempt failed")
		return
	}

	cookie := h.authenticator.GenerateCookie(expirationDate, jwt)
	http.SetCookie(w, cookie)

	response.WriteJSON(w, http.StatusOK, LoginResponse{JWT: jwt})
}
