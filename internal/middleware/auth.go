package middleware

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/portbound/go-fs/internal/handlers"
	"github.com/portbound/go-fs/internal/repositories"
	"google.golang.org/api/idtoken"
)

type contextKey string

const userEmailKey contextKey = "userEmail"

func GoogleAuthMiddleware(googleClientID string, userRepo repositories.UserRepository, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			handlers.WriteJSONError(w, http.StatusUnauthorized, "Authorization header is required")
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			handlers.WriteJSONError(w, http.StatusUnauthorized, "Authorization header is malformed")
			return
		}
		token := parts[1]

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		validator, err := idtoken.NewValidator(ctx)
		if err != nil {
			log.Printf("Error creating idtoken validator: %v", err)
			handlers.WriteJSONError(w, http.StatusInternalServerError, "Internal server error")
			return
		}

		payload, err := validator.Validate(ctx, token, googleClientID)
		if err != nil {
			log.Printf("ID token validation failed: %v", err)
			handlers.WriteJSONError(w, http.StatusUnauthorized, "Invalid or expired token")
			return
		}

		// Check if the user is in our approved list.
		userEmail := payload.Claims["email"].(string)
		_, err = userRepo.GetUser(ctx, userEmail)
		if err != nil {
			log.Printf("Unauthorized access attempt from: %s", userEmail)
			handlers.WriteJSONError(w, http.StatusForbidden, "Access denied")
			return
		}

		// Add the user's email to the request context for subsequent handlers.
		ctx = context.WithValue(r.Context(), userEmailKey, userEmail)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
