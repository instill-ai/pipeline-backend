package memorystore

import (
	"context"
)

type MemoryStore interface {
	Set(ctx context.Context, key string, value string) (err error)
	Get(ctx context.Context, key string) (value string, err error)
}

type memoryStore struct {
	data map[string]string
}

func NewMemoryStore() MemoryStore {
	return &memoryStore{
		data: make(map[string]string),
	}
}

func (m *memoryStore) Set(ctx context.Context, key string, value string) error {

	m.data[key] = value
	return nil
}

func (m *memoryStore) Get(ctx context.Context, key string) (string, error) {

	return m.data[key], nil
}
