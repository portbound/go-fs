package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/portbound/go-fs/internal/response"
	"github.com/portbound/go-fs/internal/services"
	"google.golang.org/api/idtoken"
)

type LoginRequest struct {
	Token string `json:"token"`
}

type LoginResponse struct {
	Token string `json:"token"`
}

type WebHandler struct {
	jwtSecret   string
	userService *services.UserService
}

func NewWebHandler(jwt string, us *services.UserService) *WebHandler {
	return &WebHandler{jwtSecret: jwt, userService: us}
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
		response.WriteJSON(w, http.StatusMethodNotAllowed, "WebHandler.HandleLogin failed: Method not allowed")
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error decoding request: %v", err)
		response.WriteJSONError(w, http.StatusBadRequest, "WebHandler.HandleLogin failed: Invalid request body")
		return
	}

	googleClientID := os.Getenv("GOOGLE_CLIENT_ID")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	validator, err := idtoken.NewValidator(ctx)
	if err != nil {
		response.WriteJSONError(w, http.StatusInternalServerError, "WebHandler.HandleLogin failed: Internal server error")
		return
	}

	payload, err := validator.Validate(ctx, req.Token, googleClientID)
	if err != nil {
		response.WriteJSONError(w, http.StatusUnauthorized, "WebHandler.HandleLogin failed: Login failed: Invalid token")
		return
	}

	userEmail := payload.Claims["email"]
	e, ok := userEmail.(string)
	if !ok {
		response.WriteJSONError(w, http.StatusUnauthorized, "WebHandler.HandleLogin: login failed: Invalid token")
		return
	}
	_, err = h.userService.GetUser(r.Context(), e)
	if err != nil {
		log.Printf("Unauthorized login attempt from: %s", userEmail)
		response.WriteJSONError(w, http.StatusForbidden, fmt.Sprintf("WebHandler.HandleLogin: Access denied for %s", userEmail))
		return
	}

	claims := jwt.MapClaims{
		"email": e,
		"exp":   time.Now().Add(time.Hour * 3600).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(h.jwtSecret))
	if err != nil {
		response.WriteJSONError(w, http.StatusInternalServerError, "WebHandler.HandleLogin: failed to sign JWT")
		return
	}

	response.WriteJSON(w, http.StatusOK, LoginResponse{Token: signedToken})
}
