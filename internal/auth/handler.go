package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/portbound/go-fs/internal/platform/http/response"
	"github.com/portbound/portlog"
)

type LoginRequest struct {
	Token string `json:"token"`
}

type LoginResponse struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
	JWT     string `json:"jwt"`
}

type Handler struct {
	service *Service
	logger  *portlog.PortLog
}

func NewHandler(s *Service, l *portlog.PortLog) *Handler {
	return &Handler{service: s, logger: l}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /login", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./web/public/login.html")
	})
	mux.HandleFunc("POST /login", h.HandleLogin)

}

func (h *Handler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != "POST" {
		response.JSON(w, http.StatusMethodNotAllowed, fmt.Sprintf("Method not allowed: %s", r.Method))
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("decode request data", err)
		response.Error(w, http.StatusInternalServerError, ErrFailedLogin)
		return
	}

	jwt, expiration, err := h.service.authenticateLoginRequest(r.Context(), req)
	if err != nil {
		response.JSON(w, http.StatusInternalServerError, LoginResponse{Status: false, JWT: "", Message: "Access Denied"})
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     DefaultCookieName,
		Value:    jwt,
		Path:     "/",
		MaxAge:   int(time.Until(expiration).Seconds()),
		HttpOnly: true,
		Secure:   h.service.isProd(),
		SameSite: http.SameSiteLaxMode,
	})

	response.JSON(w, http.StatusOK, LoginResponse{Status: true, JWT: jwt, Message: ""})
}
