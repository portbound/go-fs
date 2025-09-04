package auth

import (
	"context"
	"fmt"
	"os"
	"time"

	"google.golang.org/api/idtoken"
)

func ValidateOAuth(idToken string) (string, error) {
	googleClientID := os.Getenv("GOOGLE_CLIENT_ID")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	validator, err := idtoken.NewValidator(ctx)
	if err != nil {
		return "", fmt.Errorf("[auth.ValidateOAuth] failed: %w", err)
	}

	payload, err := validator.Validate(ctx, idToken, googleClientID)
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
