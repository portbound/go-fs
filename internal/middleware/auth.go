package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/portbound/go-fs/internal/models"
	"github.com/portbound/go-fs/internal/services"
	"github.com/portbound/go-fs/pkg/auth"
	"github.com/portbound/go-fs/pkg/response"
)

type contextKey string

const RequesterKey contextKey = "user"

type AuthMiddleware struct {
	authenticator *auth.Authenticator
	userService   services.UserService
}

func NewAuthMiddleware(a *auth.Authenticator, us services.UserService) *AuthMiddleware {
	return &AuthMiddleware{authenticator: a, userService: us}
}

func (mw *AuthMiddleware) RequireWebAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("gofs_session")
		if err != nil {
			if errors.Is(http.ErrNoCookie, err) {
				http.Redirect(w, r, "/login", 303)
				return
			}
			response.WriteJSONError(w, http.StatusUnauthorized, err.Error())
			return
		}

		user, err := mw.authenticateUser(r.Context(), cookie.Value)
		if err != nil {
			response.WriteJSONError(w, http.StatusUnauthorized, err.Error())
			return
		}

		ctx := context.WithValue(r.Context(), RequesterKey, user)
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

		user, err := mw.authenticateUser(r.Context(), token)
		if err != nil {
			response.WriteJSONError(w, http.StatusUnauthorized, "failed to authenticate requester")
			return
		}

		ctx := context.WithValue(r.Context(), RequesterKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (mw *AuthMiddleware) authenticateUser(ctx context.Context, token string) (*models.User, error) {
	jwt, err := mw.authenticator.ValidateJWT(token)
	if err != nil {
		return nil, err
	}

	userEmail, err := jwt.Claims.GetSubject()
	if err != nil {
		return nil, err
	}

	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	user, err := mw.userService.LookupUser(dbCtx, userEmail)
	if err != nil {
		return nil, err
	}

	return user, nil
}
