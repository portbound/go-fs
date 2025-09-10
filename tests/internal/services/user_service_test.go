package services_test

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
	"testing"

	"github.com/portbound/go-fs/internal/models"
	"github.com/portbound/go-fs/internal/services"
)

type MockUserRepository struct {
	users map[string]*models.User
}

func NewMockUserRepository() *MockUserRepository {
	return &MockUserRepository{users: make(map[string]*models.User)}
}

func (r *MockUserRepository) GetUser(ctx context.Context, email string) (*models.User, error) {
	user, ok := r.users[email]
	if !ok {
		return nil, sql.ErrNoRows
	}
	return user, nil
}

func Test_userService_LookupUser(t *testing.T) {
	testUser := &models.User{
		ID:         "1",
		Email:      "test@gmail.com",
		BucketName: "testBucket",
	}

	testCases := []struct {
		name        string
		mockUsers   map[string]*models.User
		lookupEmail string
		want        *models.User
		wantErr     bool
		expectedErr error
	}{
		{
			name: "happy path",
			mockUsers: map[string]*models.User{
				"test@gmail.com": testUser,
			},
			lookupEmail: "test@gmail.com",
			want:        testUser,
			wantErr:     false,
			expectedErr: nil,
		},
		{
			name: "user does not exist",
			mockUsers: map[string]*models.User{
				"test@gmail.com": testUser,
			},
			lookupEmail: "nonexistentUser@gmail.com",
			want:        nil,
			wantErr:     true,
			expectedErr: sql.ErrNoRows,
		},
		{
			name:        "user not found in empty repository",
			mockUsers:   make(map[string]*models.User),
			lookupEmail: "test@gmail.com",
			want:        nil,
			wantErr:     true,
			expectedErr: sql.ErrNoRows,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			mockUserRepo := NewMockUserRepository()
			mockUserRepo.users = tt.mockUsers

			userService := services.NewUserService(mockUserRepo)

			got, err := userService.LookupUser(t.Context(), tt.lookupEmail)

			if tt.wantErr {
				if err == nil {
					t.Fatal("LookupUser() succeeded unexpectedly, wanted error")
				}
				if !errors.Is(err, tt.expectedErr) {
					t.Errorf("LookupUser() error = %v, want %v", err, tt.expectedErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("LookupUser() failed unexpectedly: %v", err)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("LookupUser() got = %v, want %v", got, tt.want)
			}
		})
	}
}
