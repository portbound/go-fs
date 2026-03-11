package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/portbound/go-fs/internal/user"
)

type contextKey string

type userProvider interface {
	Get(ctx context.Context, email string) (*user.User, error)
}

type Service struct {
	authenticator *Authenticator
	userProvider  userProvider
}

func NewService(a *Authenticator, u userProvider) *Service {
	return &Service{authenticator: a, userProvider: u}
}

func (s *Service) authenticateLoginRequest(ctx context.Context, req LoginRequest) (string, time.Time, error) {
	requesterEmail, err := s.authenticator.validateOAuth(req.Token)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("validate oAuth: %w", err)
	}

	_, err = s.userProvider.Get(ctx, requesterEmail)
	if err != nil {
		return "", time.Time{}, err
	}

	expiration := time.Now().UTC().AddDate(0, 30, 0)
	jwt, err := s.authenticator.generateJWT(expiration, requesterEmail)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("generate jwt: %w", err)
	}

	return jwt, expiration, nil
}

func (s *Service) authenticateToken(ctx context.Context, token string) (*user.User, error) {
	jwt, err := s.authenticator.validateJWT(token)
	if err != nil {
		return nil, fmt.Errorf("validate jwt: %w", err)
	}

	requesterEmail, err := jwt.Claims.GetSubject()
	if err != nil {
		return nil, fmt.Errorf("get subject: %w", err)
	}

	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	user, err := s.userProvider.Get(dbCtx, requesterEmail)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (s *Service) isProd() bool {
	return s.authenticator.environment == Prod
}
