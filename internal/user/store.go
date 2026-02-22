package user

import "context"

type Store interface {
    Get(ctx context.Context, id string) (User, error)
    List(ctx context.Context, offset, limit int) ([]User, int, error)
    Create(ctx context.Context, name, email string) (User, error)
}
