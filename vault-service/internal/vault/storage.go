package vault

import (
	"fmt"
	"sync"
	"time"
)

// Secret represents a stored secret
type Secret struct {
	Key       string            `json:"key"`
	Value     string            `json:"value"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

// Storage defines the interface for secret storage
type Storage interface {
	Get(key string) (*Secret, error)
	Set(key string, value string, metadata map[string]string) error
	Delete(key string) error
	List(prefix string) ([]*Secret, error)
}

// MemoryStorage is an in-memory secret storage
type MemoryStorage struct {
	secrets map[string]*Secret
	mu      sync.RWMutex
}

// NewMemoryStorage creates a new in-memory storage
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		secrets: make(map[string]*Secret),
	}
}

// Get retrieves a secret by key
func (s *MemoryStorage) Get(key string) (*Secret, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	secret, exists := s.secrets[key]
	if !exists {
		return nil, fmt.Errorf("secret not found: %s", key)
	}
	return secret, nil
}

// Set stores a secret
func (s *MemoryStorage) Set(key, value string, metadata map[string]string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	existing, exists := s.secrets[key]

	if exists {
		existing.Value = value
		existing.Metadata = metadata
		existing.UpdatedAt = now
	} else {
		s.secrets[key] = &Secret{
			Key:       key,
			Value:     value,
			Metadata:  metadata,
			CreatedAt: now,
			UpdatedAt: now,
		}
	}

	return nil
}

// Delete removes a secret
func (s *MemoryStorage) Delete(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.secrets[key]; !exists {
		return fmt.Errorf("secret not found: %s", key)
	}

	delete(s.secrets, key)
	return nil
}

// List returns all secrets with the given prefix
func (s *MemoryStorage) List(prefix string) ([]*Secret, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []*Secret
	for key, secret := range s.secrets {
		if prefix == "" || len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			results = append(results, secret)
		}
	}

	return results, nil
}
