package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const exp time.Duration = time.Duration(time.Hour * 3600)

type Authenticator struct {
	secret string
}

func NewAuthenticator(secret string) *Authenticator {
	return &Authenticator{secret: secret}
}

func (a *Authenticator) GenerateJWT(subject string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer:    "portbound",
		IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
		ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(exp)),
		Subject:   subject,
	})

	signedToken, err := token.SignedString([]byte(a.secret))
	if err != nil {
		return "", fmt.Errorf("[auth.GenerateJWT] failed to sign token: %w", err)
	}

	return signedToken, nil
}

func (a *Authenticator) ValidateJWT(token string) (*jwt.Token, error) {
	tok, err := jwt.ParseWithClaims(token, &jwt.RegisteredClaims{}, func(t *jwt.Token) (any, error) { return []byte(a.secret), nil })
	if err != nil {
		return nil, fmt.Errorf("[auth.ValidateJWT] failed to parse claims: %w", err)
	}

	return tok, nil
}
