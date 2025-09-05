package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/portbound/go-fs/internal/services"
	"github.com/portbound/go-fs/pkg/auth"
	"github.com/portbound/go-fs/pkg/response"
)

type contextKey string

const userEmailKey contextKey = "userEmail"

type AuthMiddleware struct {
	authenticator *auth.Authenticator
	userService   *services.UserService
}

func NewAuthMiddleware(a *auth.Authenticator, us *services.UserService) *AuthMiddleware {
	return &AuthMiddleware{authenticator: a, userService: us}
}

func (mw *AuthMiddleware) RequireWebAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("go-fs")
		if err != nil {
			if errors.Is(http.ErrNoCookie, err) {
				http.Redirect(w, r, "/login", 303)
				return
			}
			response.WriteJSONError(w, http.StatusUnauthorized, err.Error())
			return
		}

		jwt, err := mw.authenticator.ValidateJWT(cookie.Value)
		if err != nil {
			response.WriteJSONError(w, http.StatusUnauthorized, err.Error())
			return
		}

		userEmail, err := jwt.Claims.GetSubject()
		if err != nil {
			response.WriteJSONError(w, http.StatusUnauthorized, err.Error())
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		user, err := mw.userService.GetUser(ctx, userEmail)
		if err != nil {
			response.WriteJSONError(w, http.StatusForbidden, "")
			return
		}

		ctx = context.WithValue(r.Context(), userEmailKey, user.Email)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (mw *AuthMiddleware) RequireAPIAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(r.Header.Get("Authorization"), " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			response.WriteJSONError(w, http.StatusUnauthorized, "'Authorization' header is missing or malformed")
			return
		}
		token := parts[1]

		jwt, err := mw.authenticator.ValidateJWT(token)
		if err != nil {
			response.WriteJSONError(w, http.StatusUnauthorized, err.Error())
			return
		}

		userEmail, err := jwt.Claims.GetSubject()
		if err != nil {
			response.WriteJSONError(w, http.StatusUnauthorized, err.Error())
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		user, err := mw.userService.GetUser(ctx, userEmail)
		if err != nil {
			response.WriteJSONError(w, http.StatusForbidden, "")
			return
		}

		ctx = context.WithValue(r.Context(), userEmailKey, user.Email)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
