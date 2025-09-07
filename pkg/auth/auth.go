package auth

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/api/idtoken"
)

type Authenticator struct {
	jwtSecret      string
	googleClientID string
}

func NewAuthenticator(jwtSecret string, googleClientID string) *Authenticator {
	return &Authenticator{jwtSecret: jwtSecret, googleClientID: googleClientID}
}

func (a *Authenticator) GenerateJWT(expirationDate time.Time, subject string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer:    "portbound",
		IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
		ExpiresAt: jwt.NewNumericDate(expirationDate),
		Subject:   subject,
	})

	signedToken, err := token.SignedString([]byte(a.jwtSecret))
	if err != nil {
		return "", fmt.Errorf("[auth.GenerateJWT] failed to sign token: %w", err)
	}

	return signedToken, nil
}

func (a *Authenticator) ValidateJWT(tokenString string) (*jwt.Token, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(t *jwt.Token) (any, error) { return []byte(a.jwtSecret), nil })
	if err != nil {
		return nil, fmt.Errorf("[auth.ValidateJWT] failed to parse claims: %w", err)
	}

	return token, nil
}

func (a *Authenticator) GenerateCookie(expirationDate time.Time, jwt string) *http.Cookie {
	return &http.Cookie{
		Name:     "gofs_session",
		Value:    jwt,
		Path:     "/",
		MaxAge:   int(time.Until(expirationDate)),
		HttpOnly: true,
		Secure:   false, //TODO: Set to false for local HTTP development
		SameSite: http.SameSiteLaxMode,
	}
}

func (a *Authenticator) ValidateOAuth(idToken string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	validator, err := idtoken.NewValidator(ctx)
	if err != nil {
		return "", fmt.Errorf("[auth.ValidateOAuth] failed: %w", err)
	}

	payload, err := validator.Validate(ctx, idToken, a.googleClientID)
	if err != nil {
		return "", fmt.Errorf("[auth.ValidateOAuth] failed: %w", err)
	}

	data := payload.Claims["email"]
	email, ok := data.(string)
	if !ok {
		return "", fmt.Errorf("[auth.ValidateOAuth] failed: %w", err)
	}

	return email, nil
}
