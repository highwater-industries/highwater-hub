// internal/user/postgres_store.go
package user

import (
    "context"
    "database/sql"
)

type PostgresStore struct {
    db *sql.DB
}

func NewPostgresStore(db *sql.DB) *PostgresStore {
    return &PostgresStore{db: db}
}

func (s *PostgresStore) Get(ctx context.Context, id string) (User, error) {
    // TODO: implement
    return User{}, nil
}

func (s *PostgresStore) List(ctx context.Context, offset, limit int) ([]User, int, error) {
    // TODO: implement
    return nil, 0, nil
}

func (s *PostgresStore) Create(ctx context.Context, name, email string) (User, error) {
    // TODO: implement
    return User{}, nil
}
