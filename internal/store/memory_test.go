// internal/store/memory_test.go
package store

import (
	"context"
	"errors"
	"testing"

	"github.com/alfonso/go-laravelcloud/internal/models"
)

func TestMemoryStore_CRUD(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()

	all, total, err := s.List(ctx, "", 20, 0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 3 || len(all) != 3 {
		t.Fatalf("seed: total=%d len=%d, want 3/3", total, len(all))
	}
	if all[0].ID > all[1].ID {
		t.Fatalf("List not sorted by ID ascending: %v", all)
	}

	created, err := s.Create(ctx, models.User{Name: "Linus", Email: "linus@example.com"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.ID == 0 || created.CreatedAt.IsZero() {
		t.Fatalf("Create did not assign ID/CreatedAt: %+v", created)
	}

	got, err := s.Get(ctx, created.ID)
	if err != nil || got.Name != "Linus" {
		t.Fatalf("Get: %+v err=%v", got, err)
	}

	updated, err := s.Update(ctx, created.ID, models.User{Name: "Linus T", Email: "lt@example.com"})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Name != "Linus T" || updated.CreatedAt != created.CreatedAt || updated.ID != created.ID {
		t.Fatalf("Update wrong: %+v", updated)
	}

	if err := s.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := s.Get(ctx, created.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get after delete = %v, want ErrNotFound", err)
	}
}

func TestMemoryStore_FilterAndPaginate(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()

	res, total, err := s.List(ctx, "ada", 20, 0)
	if err != nil {
		t.Fatalf("List filter: %v", err)
	}
	if total != 1 || len(res) != 1 {
		t.Fatalf("filter 'ada' total=%d len=%d, want 1/1", total, len(res))
	}

	page, total, err := s.List(ctx, "", 2, 0)
	if err != nil {
		t.Fatalf("List page: %v", err)
	}
	if total != 3 || len(page) != 2 {
		t.Fatalf("page total=%d len=%d, want 3/2", total, len(page))
	}

	page2, _, _ := s.List(ctx, "", 2, 2)
	if len(page2) != 1 {
		t.Fatalf("page2 len=%d, want 1", len(page2))
	}
}

func TestMemoryStore_UpdateMissing(t *testing.T) {
	s := NewMemoryStore()
	if _, err := s.Update(context.Background(), 9999, models.User{Name: "x", Email: "x@x.com"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Update missing = %v, want ErrNotFound", err)
	}
}
