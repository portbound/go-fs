package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/api/idtoken"
)

const RequesterKey contextKey = "user"
const Prod string = "production"
const DefaultCookieName string = "gofs_session"
const Issuer string = "yerboi"

var ErrFailedLogin = errors.New("login attempt failed")

type Authenticator struct {
	jwtSecret      string
	googleClientID string
	environment    string
}

func New(secret string, id string, e string) *Authenticator {
	return &Authenticator{jwtSecret: secret, googleClientID: id, environment: e}
}

func (a *Authenticator) validateOAuth(idToken string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	validator, err := idtoken.NewValidator(ctx)
	if err != nil {
		return "", fmt.Errorf("new validator: %w", err)
	}

	payload, err := validator.Validate(ctx, idToken, a.googleClientID)
	if err != nil {
		return "", fmt.Errorf("validate: %w", err)
	}

	data := payload.Claims["email"]
	email, ok := data.(string)
	if !ok {
		return "", fmt.Errorf("parse claims: %w", err)
	}

	return email, nil
}

func (a *Authenticator) generateJWT(expirationDate time.Time, subject string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer:    Issuer,
		IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
		ExpiresAt: jwt.NewNumericDate(expirationDate),
		Subject:   subject,
	})

	signedToken, err := token.SignedString([]byte(a.jwtSecret))
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}

	return signedToken, nil
}

func (a *Authenticator) validateJWT(tokenString string) (*jwt.Token, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(t *jwt.Token) (any, error) { return []byte(a.jwtSecret), nil })
	if err != nil {
		return nil, fmt.Errorf("parse with claims: %w", err)
	}

	return token, nil
}
