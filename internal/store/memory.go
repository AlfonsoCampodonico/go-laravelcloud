// internal/store/memory.go
package store

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/alfonso/go-laravelcloud/internal/models"
)

type MemoryStore struct {
	mu     sync.RWMutex
	users  map[int]models.User
	nextID int
}

func NewMemoryStore() *MemoryStore {
	now := time.Now()
	s := &MemoryStore{users: make(map[int]models.User), nextID: 1}
	seed := []models.User{
		{Name: "Ada Lovelace", Email: "ada@example.com", CreatedAt: now.Add(-72 * time.Hour)},
		{Name: "Alan Turing", Email: "alan@example.com", CreatedAt: now.Add(-48 * time.Hour)},
		{Name: "Grace Hopper", Email: "grace@example.com", CreatedAt: now.Add(-24 * time.Hour)},
	}
	for _, u := range seed {
		u.ID = s.nextID
		s.users[u.ID] = u
		s.nextID++
	}
	return s
}

func (s *MemoryStore) List(_ context.Context, q string, limit, offset int) ([]models.User, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	matches := make([]models.User, 0, len(s.users))
	q = strings.ToLower(strings.TrimSpace(q))
	for _, u := range s.users {
		if q == "" || strings.Contains(strings.ToLower(u.Name), q) || strings.Contains(strings.ToLower(u.Email), q) {
			matches = append(matches, u)
		}
	}
	sort.Slice(matches, func(i, j int) bool { return matches[i].ID < matches[j].ID })
	total := len(matches)

	if offset > total {
		offset = total
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return matches[offset:end], total, nil
}

func (s *MemoryStore) Get(_ context.Context, id int) (models.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.users[id]
	if !ok {
		return models.User{}, ErrNotFound
	}
	return u, nil
}

func (s *MemoryStore) Create(_ context.Context, u models.User) (models.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	u.ID = s.nextID
	u.CreatedAt = time.Now()
	s.users[u.ID] = u
	s.nextID++
	return u, nil
}

func (s *MemoryStore) Update(_ context.Context, id int, u models.User) (models.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.users[id]
	if !ok {
		return models.User{}, ErrNotFound
	}
	existing.Name = u.Name
	existing.Email = u.Email
	s.users[id] = existing
	return existing, nil
}

func (s *MemoryStore) Delete(_ context.Context, id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.users[id]; !ok {
		return ErrNotFound
	}
	delete(s.users, id)
	return nil
}
