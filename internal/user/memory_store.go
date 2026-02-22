package user

import (
    "context"
    "fmt"
    "sync"
)

type MemoryStore struct {
    mu     sync.Mutex
    users  map[string]User
    nextID int
}

func NewMemoryStore() *MemoryStore {
    return &MemoryStore{users: make(map[string]User)}
}

func (s *MemoryStore) Get(ctx context.Context, id string) (User, error) {
    s.mu.Lock()
    defer s.mu.Unlock()
    u, ok := s.users[id]
    if !ok {
        return User{}, fmt.Errorf("user not found: %s", id)
    }
    return u, nil
}

func (s *MemoryStore) List(ctx context.Context, offset, limit int) ([]User, int, error) {
    s.mu.Lock()
    defer s.mu.Unlock()
    all := make([]User, 0, len(s.users))
    for _, u := range s.users {
        all = append(all, u)
    }
    total := len(all)
    if offset > len(all) {
        return []User{}, total, nil
    }
    end := min(offset+limit, len(all))
    return all[offset:end], total, nil
}

func (s *MemoryStore) Create(ctx context.Context, name, email string) (User, error) {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.nextID++
    u := User{
        Id:    fmt.Sprintf("%d", s.nextID),
        Name:  name,
        Email: email,
    }
    s.users[u.Id] = u
    return u, nil
}
