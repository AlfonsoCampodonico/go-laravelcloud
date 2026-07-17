// internal/store/store.go
package store

import (
	"context"
	"errors"

	"github.com/alfonso/go-laravelcloud/internal/models"
)

var ErrNotFound = errors.New("user not found")

type Store interface {
	List(ctx context.Context, q string, limit, offset int) ([]models.User, int, error)
	Get(ctx context.Context, id int) (models.User, error)
	Create(ctx context.Context, u models.User) (models.User, error)
	Update(ctx context.Context, id int, u models.User) (models.User, error)
	Delete(ctx context.Context, id int) error
}
