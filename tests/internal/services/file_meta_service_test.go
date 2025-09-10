package services_test

import (
	"context"
	"database/sql"
	"reflect"
	"slices"
	"testing"

	"github.com/portbound/go-fs/internal/models"
	"github.com/portbound/go-fs/internal/services"
)

type MockFileMetaRepository struct {
	fileMeta map[string]*models.FileMeta
}

func NewMockFileMetaRepository() *MockFileMetaRepository {
	return &MockFileMetaRepository{fileMeta: make(map[string]*models.FileMeta)}
}

func (r *MockFileMetaRepository) CreateFileMeta(ctx context.Context, fm *models.FileMeta) error {
	r.fileMeta[fm.ID] = fm
	return nil
}

func (r *MockFileMetaRepository) GetFileMeta(ctx context.Context, id string, owner *models.User) (*models.FileMeta, error) {
	fm, ok := r.fileMeta[id]
	if !ok {
		return nil, sql.ErrNoRows
	}
	return fm, nil
}

func (r *MockFileMetaRepository) GetAllFileMeta(ctx context.Context, owner *models.User) ([]*models.FileMeta, error) {
	var afm []*models.FileMeta
	for _, fm := range r.fileMeta {
		afm = append(afm, fm)
	}
	return afm, nil
}
func (r *MockFileMetaRepository) DeleteFileMeta(ctx context.Context, id string) error {
	fm, ok := r.fileMeta[id]
	if !ok {
		return sql.ErrNoRows
	}

	delete(r.fileMeta, fm.ID)

	return nil
}

func Test_fileMetaService_SaveFileMeta(t *testing.T) {
	testFileMeta := models.FileMeta{
		ID:   "1",
		Name: "test.txt",
	}

	tests := []struct {
		name         string
		mockFileMeta map[string]*models.FileMeta
		want         *models.FileMeta
		wantErr      bool
		expectedErr  error
	}{
		{
			name:         "happy path",
			mockFileMeta: map[string]*models.FileMeta{},
			want:         &testFileMeta,
			wantErr:      false,
			expectedErr:  nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockFileMetaRepo := NewMockFileMetaRepository()
			mockFileMetaRepo.fileMeta = tt.mockFileMeta
			fileMetaService := services.NewFileMetaService(mockFileMetaRepo)

			gotErr := fileMetaService.SaveFileMeta(context.Background(), tt.want)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("SaveFileMeta() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("SaveFileMeta() succeeded unexpectedly")
			}
		})
	}
}

func Test_fileMetaService_LookupFileMeta(t *testing.T) {
	testFileMeta := models.FileMeta{
		ID:    "1",
		Name:  "test.txt",
		Owner: "test",
	}

	tests := []struct {
		name         string
		mockFileMeta map[string]*models.FileMeta
		mockOwner    *models.User
		want         *models.FileMeta
		wantErr      bool
		expectedErr  error
	}{
		{
			name: "happy path",
			mockFileMeta: map[string]*models.FileMeta{
				"1": &testFileMeta,
			},
			mockOwner: &models.User{
				ID:         "1",
				Email:      "test",
				BucketName: "testBucket",
			},
			want:        &testFileMeta,
			wantErr:     false,
			expectedErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockFileMetaRepository := NewMockFileMetaRepository()
			mockFileMetaRepository.fileMeta = tt.mockFileMeta
			fileMetaService := services.NewFileMetaService(mockFileMetaRepository)

			got, gotErr := fileMetaService.LookupFileMeta(context.Background(), tt.want.ID, tt.mockOwner)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("LookupFileMeta() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("LookupFileMeta() succeeded unexpectedly")
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("LookupFileMeta() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_fileMetaService_LookupAllFileMeta(t *testing.T) {
	testAllFileMeta := []*models.FileMeta{
		{
			ID:    "1",
			Name:  "test.txt",
			Owner: "test",
		},
		{
			ID:    "2",
			Name:  "anotherTest.txt",
			Owner: "test",
		},
	}

	tests := []struct {
		name         string
		mockFileMeta map[string]*models.FileMeta
		mockOwner    *models.User
		want         []*models.FileMeta
		wantErr      bool
		expectedErr  error
	}{
		{
			name: "happy path",
			mockFileMeta: map[string]*models.FileMeta{
				"1": {
					ID:    "1",
					Name:  "test.txt",
					Owner: "test",
				},
				"2": {
					ID:    "2",
					Name:  "anotherTest.txt",
					Owner: "test",
				},
			},
			mockOwner: &models.User{
				ID:         "1",
				Email:      "test",
				BucketName: "testBucket",
			},
			want:        testAllFileMeta,
			wantErr:     false,
			expectedErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockFileMetaRepository := NewMockFileMetaRepository()
			mockFileMetaRepository.fileMeta = tt.mockFileMeta

			fileMetaService := services.NewFileMetaService(mockFileMetaRepository)

			got, gotErr := fileMetaService.LookupAllFileMeta(context.Background(), tt.mockOwner)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("LookupAllFileMeta() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("LookupAllFileMeta() succeeded unexpectedly")
			}
			if slices.Equal(got, tt.want) {
				t.Errorf("LookupAllFileMeta() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_fileMetaService_DeleteFileMeta(t *testing.T) {
	testFileMeta := &models.FileMeta{
		ID:    "1",
		Name:  "test.txt",
		Owner: "test",
	}

	tests := []struct {
		name         string
		mockFileMeta map[string]*models.FileMeta
		mockOwner    *models.User
		id           string
		want         []*models.FileMeta
		wantErr      bool
		expectedErr  error
	}{
		{
			name: "happy path",
			mockFileMeta: map[string]*models.FileMeta{
				"1": testFileMeta,
			},
			mockOwner: &models.User{
				ID:         "1",
				Email:      "test",
				BucketName: "testBucket",
			},
			id:          "1",
			want:        nil,
			wantErr:     false,
			expectedErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockFileMetaRepository := NewMockFileMetaRepository()
			mockFileMetaRepository.fileMeta = tt.mockFileMeta

			fileMetaService := services.NewFileMetaService(mockFileMetaRepository)

			gotErr := fileMetaService.DeleteFileMeta(context.Background(), tt.id)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("DeleteFileMeta() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("DeleteFileMeta() succeeded unexpectedly")
			}
		})
	}
}
