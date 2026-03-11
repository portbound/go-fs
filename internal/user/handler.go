package user

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/portbound/go-fs/pkg/auth"
	 "github.com/portbound/go-fs/internal/platform/http/response"
	"github.com/portbound/portlog"
)

type LoginRequest struct {
	Token string `json:"token"`
}

type LoginResponse struct {
	JWT string `json:"jwt"`
}

type Handler struct {
	service       *Service
	authenticator Authenticator
	logger        *portlog.PortLog
}

func NewHandler(a Authenticator, s *Service, l *portlog.PortLog) *Handler {
	return &Handler{authenticator: a, service: s, logger: l}
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

	requesterEmail, err := h.authenticator.ValidateOAuth(req.Token)
	if err != nil {
		h.logger.Error("oAuth validation", err)
		response.Error(w, http.StatusInternalServerError, ErrFailedLogin)
		return
	}

	_, err = h.service.Get(r.Context(), requesterEmail)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			h.logger.Error("unauthorized login attempt", err, "requester", requesterEmail)
			response.Error(w, http.StatusForbidden, fmt.Errorf("permission denied for user: '%s'", requesterEmail))
			return
		}
		h.logger.Error("user lookup", err)
		response.Error(w, http.StatusInternalServerError, ErrFailedLogin)
		return
	}

	expirationDate := time.Now().UTC().AddDate(0, 30, 0)
	jwt, err := h.authenticator.GenerateJWT(expirationDate, requesterEmail)
	if err != nil {
		h.logger.Error("failed to provision JWT", err, "requester", requesterEmail)
		response.Error(w, http.StatusInternalServerError, ErrFailedLogin)
		return
	}

	cookie := h.authenticator.GenerateCookie(expirationDate, jwt)
	http.SetCookie(w, cookie)

	response.JSON(w, http.StatusOK, LoginResponse{JWT: jwt})
}
