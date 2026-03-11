package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/portbound/go-fs/internal/platform/http/response"
)

func (h *Handler) RequireWebAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(DefaultCookieName)
		if err != nil {
			http.Redirect(w, r, "/login", 303)
			return
		}

		requester, err := h.service.authenticateToken(r.Context(), cookie.Value)
		if err != nil {
			response.Error(w, http.StatusUnauthorized, err)
			return
		}

		ctx := context.WithValue(r.Context(), RequesterKey, requester)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (h *Handler) RequireAPIAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(r.Header.Get("Authorization"), " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			response.Error(w, http.StatusUnauthorized, errors.New("Authorization header is missing or malformed"))
			return
		}
		token := parts[1]

		requester, err := h.service.authenticateToken(r.Context(), token)
		if err != nil {
			response.Error(w, http.StatusUnauthorized, errors.New("failed to authenticate requester"))
			return
		}

		ctx := context.WithValue(r.Context(), RequesterKey, requester)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
