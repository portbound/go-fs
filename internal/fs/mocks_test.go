package fs

import (
	"context"
	"io"

	"cloud.google.com/go/storage"
)

type MockMediaStore struct {
}

func NewMockMediaStore() *MockMediaStore {
	return &MockMediaStore{}
}

func (m *MockMediaStore) Upload(ctx context.Context, name, bucket string, src io.Reader) error {
	return nil
}

func (m *MockMediaStore) Download(ctx context.Context, name, bucket string) (*storage.ObjectAttrs, *storage.Reader, error) {
	return nil, nil, nil
}

func (m *MockMediaStore) Delete(ctx context.Context, name, bucket string) error {
	return nil
}

type MockMetaStore struct {
	store map[string]*Metadata
}

func NewMockMetaStore() *MockMetaStore {
	return &MockMetaStore{
		store: make(map[string]*Metadata),
	}
}

func (m *MockMetaStore) Save(ctx context.Context, meta *Metadata) error {
	m.store[meta.Id] = meta
	return nil
}

func (m *MockMetaStore) Get(ctx context.Context, fileId, userId string) (*Metadata, error) {
	meta, ok := m.store[fileId]
	if !ok {
		return nil, ErrMediaNotExist
	}
	return meta, nil
}

func (m *MockMetaStore) GetAll(ctx context.Context, userId string) ([]Metadata, error) {
	var all []Metadata
	for _, meta := range m.store {
		if meta.UserId == userId {
			all = append(all, *meta)
		}
	}
	return all, nil
}

func (m *MockMetaStore) Delete(ctx context.Context, fileId, userId string) error {
	delete(m.store, fileId)
	return nil
}
