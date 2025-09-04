package middleware

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/portbound/go-fs/internal/response"
	"github.com/portbound/go-fs/internal/services"
	"google.golang.org/api/idtoken"
)

type contextKey string

const userEmailKey contextKey = "userEmail"

type AuthMiddleware struct {
	userService *services.UserService
}

func NewAuthMiddleware(us *services.UserService) *AuthMiddleware {
	return &AuthMiddleware{userService: us}
}

func (m *AuthMiddleware) RequireWebAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
}

func (m *AuthMiddleware) RequireAPIAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		googleClientID := os.Getenv("GOOGLE_CLIENT_ID")
		if googleClientID == "" {
			log.Fatal("GOOGLE_CLIENT_ID environment variable not set. Please set it to your Google OAuth client ID.")
		}

		authHeader := r.Header.Get("Authorization")
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			response.WriteJSONError(w, http.StatusUnauthorized, "Authorization header is malformed")
			return
		}
		token := parts[1]

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		validator, err := idtoken.NewValidator(ctx)
		if err != nil {
			log.Printf("Error creating idtoken validator: %v", err)
			response.WriteJSONError(w, http.StatusInternalServerError, "Internal server error")
			return
		}

		payload, err := validator.Validate(ctx, token, googleClientID)
		if err != nil {
			log.Printf("ID token validation failed: %v", err)
			response.WriteJSONError(w, http.StatusUnauthorized, "Invalid or expired token")
			return
		}

		userEmail := payload.Claims["email"].(string)
		_, err = m.userService.GetUser(ctx, userEmail)
		if err != nil {
			log.Printf("Unauthorized access attempt from: %s", userEmail)
			response.WriteJSONError(w, http.StatusForbidden, "Access denied")
			return
		}

		// Add the user's email to the request context for subsequent response.
		ctx = context.WithValue(r.Context(), userEmailKey, userEmail)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
