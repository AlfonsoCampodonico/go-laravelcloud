# OpenAPI Showcase API Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Turn the minimal stdlib HTTP API into a polished, OpenAPI-documented REST API (with embedded Swagger UI) that showcases Go on Laravel Cloud.

**Architecture:** Stdlib `net/http` only. A `Store` interface (in-memory implementation) backs a complete `users` resource. A hand-written OpenAPI 3.1 spec (`api/openapi.json`) and the Swagger UI assets are compiled into the binary with `go:embed` and served at `/openapi.json` and `/docs`. A single source-of-truth route list drives both `http.ServeMux` registration and a spec-drift contract test.

**Tech Stack:** Go 1.26, standard library only (net/http, embed, encoding/json, net/mail). Swagger UI dist (static assets, vendored). Docker (distroless).

## Global Constraints

- **Zero third-party Go dependencies.** `go.mod` must have no `require` directives. The contract test parses the spec with stdlib `encoding/json` (this is why the spec is JSON, not YAML).
- **Go 1.22+ routing** via `http.ServeMux` method+path patterns (e.g. `POST /users`, `GET /users/{id}`). No third-party router.
- **Single self-contained binary.** Spec + Swagger UI embedded via `go:embed` — no CDN, no runtime file dependencies.
- **Module path:** `github.com/alfonso/go-laravelcloud`.
- **Structured errors everywhere:** every error response is a `models.APIError` JSON body.
- **Every functional endpoint appears in `api/openapi.json`.** Only meta endpoints (`/`, `/openapi.json`, `/docs/`) are exempt (marked `Undocumented`).
- `make test` and `make vet` stay green after every task.

---

### Task 1: User model, input validation, and APIError type

**Files:**
- Modify: `internal/models/user.go`
- Test: `internal/models/user_test.go`

**Interfaces:**
- Consumes: nothing.
- Produces:
  - `models.User` struct (unchanged fields: `ID int`, `Name string`, `Email string`, `CreatedAt time.Time`).
  - `models.UserInput struct { Name string; Email string }` with method `func (in UserInput) Validate() error`. Returns an error whose `.Error()` names the first problem; returns `nil` when valid. Trims `Name`; requires 1–100 chars; requires `Email` to parse via `net/mail.ParseAddress`.
  - `models.APIError struct { Error string; Detail string; Status int; RequestID string }` with JSON tags `error`, `detail`, `status`, `request_id` (`request_id` is `omitempty`).

- [ ] **Step 1: Write the failing test**

```go
// internal/models/user_test.go
package models

import "testing"

func TestUserInput_Validate(t *testing.T) {
	tests := []struct {
		name    string
		in      UserInput
		wantErr bool
	}{
		{"valid", UserInput{Name: "Ada", Email: "ada@example.com"}, false},
		{"trims and accepts", UserInput{Name: "  Ada  ", Email: "ada@example.com"}, false},
		{"empty name", UserInput{Name: "   ", Email: "ada@example.com"}, true},
		{"missing email", UserInput{Name: "Ada", Email: ""}, true},
		{"bad email", UserInput{Name: "Ada", Email: "not-an-email"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.in.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUserInput_ValidateRejectsLongName(t *testing.T) {
	long := make([]byte, 101)
	for i := range long {
		long[i] = 'a'
	}
	in := UserInput{Name: string(long), Email: "ada@example.com"}
	if err := in.Validate(); err == nil {
		t.Fatal("expected error for 101-char name, got nil")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/models/ -run TestUserInput -v`
Expected: FAIL — `undefined: UserInput`.

- [ ] **Step 3: Write minimal implementation**

```go
// internal/models/user.go
package models

import (
	"errors"
	"net/mail"
	"strings"
	"time"
)

type User struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

type UserInput struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

func (in UserInput) Validate() error {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return errors.New("name is required")
	}
	if len(name) > 100 {
		return errors.New("name must be at most 100 characters")
	}
	if in.Email == "" {
		return errors.New("email is required")
	}
	if _, err := mail.ParseAddress(in.Email); err != nil {
		return errors.New("email must be a valid address")
	}
	return nil
}

type APIError struct {
	Error     string `json:"error"`
	Detail    string `json:"detail"`
	Status    int    `json:"status"`
	RequestID string `json:"request_id,omitempty"`
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/models/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/models/user.go internal/models/user_test.go
git commit -m "Add user input validation and APIError model"
```

---

### Task 2: Store interface and in-memory implementation

**Files:**
- Create: `internal/store/store.go`
- Create: `internal/store/memory.go`
- Test: `internal/store/memory_test.go`

**Interfaces:**
- Consumes: `models.User` (Task 1).
- Produces:
  - `store.ErrNotFound` (`var ErrNotFound = errors.New("user not found")`).
  - `store.Store` interface:
    ```go
    List(ctx context.Context, q string, limit, offset int) ([]models.User, int, error)
    Get(ctx context.Context, id int) (models.User, error)
    Create(ctx context.Context, u models.User) (models.User, error)
    Update(ctx context.Context, id int, u models.User) (models.User, error)
    Delete(ctx context.Context, id int) error
    ```
  - `store.NewMemoryStore() *store.MemoryStore` — seeded with 3 users, implements `Store`.
  - Semantics: `List` filters by case-insensitive substring `q` over name+email, returns results sorted by `ID` ascending, `total` = match count before pagination, then applies `offset`/`limit`. `Create` assigns the next ID and sets `CreatedAt = time.Now()`. `Update` preserves the existing `ID` and `CreatedAt`, replaces `Name`/`Email`, returns `ErrNotFound` if absent. `Get`/`Delete` return `ErrNotFound` if absent.

- [ ] **Step 1: Write the failing test**

```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/store/ -v`
Expected: FAIL — `undefined: NewMemoryStore`.

- [ ] **Step 3: Write the interface**

```go
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
```

- [ ] **Step 4: Write the in-memory implementation**

```go
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
```

Note: a `PostgresStore` implementing the same `Store` interface (backed by Laravel Cloud managed Postgres via injected `DB_*` env vars) would drop in here without touching handlers. Not implemented in this pass.

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/store/ -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/store/
git commit -m "Add Store interface and in-memory implementation"
```

---

### Task 3: Structured error responder with request ID

**Files:**
- Modify: `internal/respond/respond.go`
- Test: `internal/respond/respond_test.go`

**Interfaces:**
- Consumes: `models.APIError` (Task 1), `middleware.RequestIDFrom` (existing).
- Produces:
  - `respond.JSON(w http.ResponseWriter, status int, body any)` — unchanged.
  - `respond.Error(w http.ResponseWriter, r *http.Request, status int, detail string)` — **signature changed** to take `*http.Request`. Writes a `models.APIError` with `Error = http.StatusText(status)`, `Detail = detail`, `Status = status`, `RequestID = middleware.RequestIDFrom(r.Context())`.

Note: this changes every existing `respond.Error(w, status, msg)` call site. Those are fixed in Task 5 (users) and Task 6 (registry/utility handlers); until then the build in `internal/handlers` is red, which is expected mid-refactor. Keep commits per-task; `go test ./internal/respond/...` is the gate for this task.

- [ ] **Step 1: Write the failing test**

```go
// internal/respond/respond_test.go
package respond

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alfonso/go-laravelcloud/internal/models"
)

func TestError_StructuredBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	rec := httptest.NewRecorder()

	Error(rec, req, http.StatusNotFound, "user not found")

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	var body models.APIError
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Error != "Not Found" || body.Detail != "user not found" || body.Status != 404 {
		t.Fatalf("unexpected body: %+v", body)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/respond/ -v`
Expected: FAIL — `too many arguments in call to Error` (old signature).

- [ ] **Step 3: Update the implementation**

```go
// internal/respond/respond.go
package respond

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/alfonso/go-laravelcloud/internal/middleware"
	"github.com/alfonso/go-laravelcloud/internal/models"
)

func JSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		log.Printf("respond: encode error: %v", err)
	}
}

func Error(w http.ResponseWriter, r *http.Request, status int, detail string) {
	JSON(w, status, models.APIError{
		Error:     http.StatusText(status),
		Detail:    detail,
		Status:    status,
		RequestID: middleware.RequestIDFrom(r.Context()),
	})
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/respond/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/respond/
git commit -m "Return structured APIError with request ID from responder"
```

---

### Task 4: CORS middleware

**Files:**
- Modify: `internal/middleware/middleware.go`
- Test: `internal/middleware/middleware_test.go`

**Interfaces:**
- Consumes: nothing new.
- Produces: `middleware.CORS(next http.Handler) http.Handler` — sets `Access-Control-Allow-Origin: *`, `Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS`, `Access-Control-Allow-Headers: Content-Type, X-Request-ID`. Answers `OPTIONS` preflight with `204` and does not call `next`.

- [ ] **Step 1: Write the failing test**

```go
// internal/middleware/middleware_test.go
package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORS_SetsHeaders(t *testing.T) {
	h := CORS(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("ACAO = %q, want *", got)
	}
}

func TestCORS_PreflightShortCircuits(t *testing.T) {
	called := false
	h := CORS(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true }))
	req := httptest.NewRequest(http.MethodOptions, "/x", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if called {
		t.Fatal("preflight should not call next handler")
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("preflight status = %d, want 204", rec.Code)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/middleware/ -run TestCORS -v`
Expected: FAIL — `undefined: CORS`.

- [ ] **Step 3: Add the middleware**

Append to `internal/middleware/middleware.go`:

```go
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Request-ID")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/middleware/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/middleware/
git commit -m "Add permissive CORS middleware"
```

---

### Task 5: Users resource handlers (store-backed, full CRUD)

**Files:**
- Rewrite: `internal/handlers/users.go`
- Test: `internal/handlers/users_test.go`

**Interfaces:**
- Consumes: `store.Store`, `store.ErrNotFound` (Task 2); `models.User`, `models.UserInput` (Task 1); `respond.JSON`, `respond.Error(w, r, status, detail)` (Task 3).
- Produces:
  - `handlers.UserHandler struct { Store store.Store }`.
  - Methods (all `func (h *UserHandler) X(w http.ResponseWriter, r *http.Request)`): `List`, `Get`, `Create`, `Update`, `Delete`.
  - `List` reads query params `q` (string), `limit` (default 20, clamped 1..100), `offset` (default 0, min 0); responds `200` with `{"users": [...], "count": <total>, "limit": <n>, "offset": <n>}`.
  - `Get` / `Delete` parse `{id}`; `400` on non-int, `404` (mapped from `ErrNotFound`) if absent; `Get` → `200` user, `Delete` → `204`.
  - `Create` decodes `UserInput`, validates, `201` with created user.
  - `Update` parses `{id}`, decodes+validates `UserInput`, `200` with updated user, `404` if absent.

- [ ] **Step 1: Write the failing tests**

```go
// internal/handlers/users_test.go
package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alfonso/go-laravelcloud/internal/models"
	"github.com/alfonso/go-laravelcloud/internal/store"
)

func newUserHandler() *UserHandler { return &UserHandler{Store: store.NewMemoryStore()} }

func TestUsers_List(t *testing.T) {
	h := newUserHandler()
	req := httptest.NewRequest(http.MethodGet, "/users?limit=2", nil)
	rec := httptest.NewRecorder()
	h.List(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var body struct {
		Users  []models.User `json:"users"`
		Count  int           `json:"count"`
		Limit  int           `json:"limit"`
		Offset int           `json:"offset"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Count != 3 || len(body.Users) != 2 || body.Limit != 2 {
		t.Fatalf("unexpected list body: %+v", body)
	}
}

func TestUsers_Get_NotFound(t *testing.T) {
	h := newUserHandler()
	req := httptest.NewRequest(http.MethodGet, "/users/9999", nil)
	req.SetPathValue("id", "9999")
	rec := httptest.NewRecorder()
	h.Get(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestUsers_Get_BadID(t *testing.T) {
	h := newUserHandler()
	req := httptest.NewRequest(http.MethodGet, "/users/abc", nil)
	req.SetPathValue("id", "abc")
	rec := httptest.NewRecorder()
	h.Get(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestUsers_Create(t *testing.T) {
	h := newUserHandler()
	req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(`{"name":"Linus","email":"linus@example.com"}`))
	rec := httptest.NewRecorder()
	h.Create(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201", rec.Code)
	}
	var u models.User
	_ = json.NewDecoder(rec.Body).Decode(&u)
	if u.ID == 0 || u.Name != "Linus" {
		t.Fatalf("unexpected created user: %+v", u)
	}
}

func TestUsers_Create_Invalid(t *testing.T) {
	h := newUserHandler()
	req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(`{"name":"","email":"nope"}`))
	rec := httptest.NewRecorder()
	h.Create(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestUsers_Update(t *testing.T) {
	h := newUserHandler()
	req := httptest.NewRequest(http.MethodPut, "/users/1", strings.NewReader(`{"name":"Ada L","email":"ada2@example.com"}`))
	req.SetPathValue("id", "1")
	rec := httptest.NewRecorder()
	h.Update(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var u models.User
	_ = json.NewDecoder(rec.Body).Decode(&u)
	if u.Name != "Ada L" || u.ID != 1 {
		t.Fatalf("unexpected updated user: %+v", u)
	}
}

func TestUsers_Delete(t *testing.T) {
	h := newUserHandler()
	req := httptest.NewRequest(http.MethodDelete, "/users/1", nil)
	req.SetPathValue("id", "1")
	rec := httptest.NewRecorder()
	h.Delete(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/handlers/ -run TestUsers -v`
Expected: FAIL — `undefined: UserHandler` (and, at this point, the package may not compile because `handlers.go` still calls the old `respond.Error`; that is fixed in Task 6).

- [ ] **Step 3: Write the implementation**

```go
// internal/handlers/users.go
package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/alfonso/go-laravelcloud/internal/models"
	"github.com/alfonso/go-laravelcloud/internal/respond"
	"github.com/alfonso/go-laravelcloud/internal/store"
)

type UserHandler struct {
	Store store.Store
}

func (h *UserHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	limit := clampQueryInt(r, "limit", 20, 1, 100)
	offset := clampQueryInt(r, "offset", 0, 0, 1<<31-1)

	users, total, err := h.Store.List(r.Context(), q, limit, offset)
	if err != nil {
		respond.Error(w, r, http.StatusInternalServerError, "failed to list users")
		return
	}
	respond.JSON(w, http.StatusOK, map[string]any{
		"users":  users,
		"count":  total,
		"limit":  limit,
		"offset": offset,
	})
}

func (h *UserHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	u, err := h.Store.Get(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		respond.Error(w, r, http.StatusNotFound, "user not found")
		return
	}
	if err != nil {
		respond.Error(w, r, http.StatusInternalServerError, "failed to fetch user")
		return
	}
	respond.JSON(w, http.StatusOK, u)
}

func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
	in, ok := decodeUserInput(w, r)
	if !ok {
		return
	}
	u, err := h.Store.Create(r.Context(), models.User{Name: in.Name, Email: in.Email})
	if err != nil {
		respond.Error(w, r, http.StatusInternalServerError, "failed to create user")
		return
	}
	respond.JSON(w, http.StatusCreated, u)
}

func (h *UserHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	in, ok := decodeUserInput(w, r)
	if !ok {
		return
	}
	u, err := h.Store.Update(r.Context(), id, models.User{Name: in.Name, Email: in.Email})
	if errors.Is(err, store.ErrNotFound) {
		respond.Error(w, r, http.StatusNotFound, "user not found")
		return
	}
	if err != nil {
		respond.Error(w, r, http.StatusInternalServerError, "failed to update user")
		return
	}
	respond.JSON(w, http.StatusOK, u)
}

func (h *UserHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	err := h.Store.Delete(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		respond.Error(w, r, http.StatusNotFound, "user not found")
		return
	}
	if err != nil {
		respond.Error(w, r, http.StatusInternalServerError, "failed to delete user")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func parseID(w http.ResponseWriter, r *http.Request) (int, bool) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		respond.Error(w, r, http.StatusBadRequest, "invalid id")
		return 0, false
	}
	return id, true
}

func decodeUserInput(w http.ResponseWriter, r *http.Request) (models.UserInput, bool) {
	var in models.UserInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		respond.Error(w, r, http.StatusBadRequest, "invalid json body")
		return in, false
	}
	if err := in.Validate(); err != nil {
		respond.Error(w, r, http.StatusBadRequest, err.Error())
		return in, false
	}
	return in, true
}

func clampQueryInt(r *http.Request, key string, def, min, max int) int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	if n < min {
		return min
	}
	if n > max {
		return max
	}
	return n
}
```

- [ ] **Step 4: Note**

Tests won't pass yet — `internal/handlers` still contains the old `Register`/utility handlers calling `respond.Error(w, status, msg)`. Proceed to Task 6, which fixes the package, then run the users tests at the end of Task 6.

- [ ] **Step 5: Commit**

```bash
git add internal/handlers/users.go internal/handlers/users_test.go
git commit -m "Rewrite users resource as store-backed CRUD handler"
```

---

### Task 6: Route registry, utility-handler fixups, and simplified root

**Files:**
- Rewrite: `internal/handlers/handlers.go`
- Modify: `internal/handlers/hello.go` (simplify `Root`)
- Modify: `internal/handlers/random.go`, `internal/handlers/echo.go` (fix `respond.Error` call sites)

**Interfaces:**
- Consumes: `store.Store` (Task 2); `UserHandler` (Task 5); `DocsSpec`, `DocsUI` handlers (Task 7 — declared there, referenced here).
- Produces:
  - `handlers.Route struct { Method string; Pattern string; Handler http.HandlerFunc; Undocumented bool }`.
  - `handlers.Routes(st store.Store) []Route` — the single source of truth for all routes.
  - `handlers.Register(mux *http.ServeMux, st store.Store)` — iterates `Routes` and calls `mux.HandleFunc(method+" "+pattern, handler)`; the docs UI subtree (`/docs/`) is registered via `mux.Handle` because it is a subtree pattern.

**Important:** Task 7 must be completed alongside this task before the package compiles, because `Routes` references `DocsSpec` and `DocsUI`. Implement Task 7's `docs.go` first, or stub the two handlers, then run tests. The commit for this task comes after Task 7's file exists.

- [ ] **Step 1: Rewrite the registry**

```go
// internal/handlers/handlers.go
package handlers

import (
	"net/http"

	"github.com/alfonso/go-laravelcloud/internal/store"
)

type Route struct {
	Method       string
	Pattern      string
	Handler      http.HandlerFunc
	Undocumented bool // excluded from the OpenAPI contract test
}

// Routes is the single source of truth for the API surface. The contract test
// (contract_test.go) asserts every documented route appears in api/openapi.json.
func Routes(st store.Store) []Route {
	u := &UserHandler{Store: st}
	return []Route{
		{Method: "GET", Pattern: "/", Handler: Root, Undocumented: true},
		{Method: "GET", Pattern: "/hello", Handler: Hello},
		{Method: "GET", Pattern: "/health", Handler: Health},
		{Method: "GET", Pattern: "/version", Handler: Version},
		{Method: "GET", Pattern: "/time", Handler: Time},
		{Method: "GET", Pattern: "/random", Handler: Random},
		{Method: "GET", Pattern: "/quote", Handler: Quote},
		{Method: "POST", Pattern: "/echo", Handler: Echo},

		{Method: "GET", Pattern: "/users", Handler: u.List},
		{Method: "GET", Pattern: "/users/{id}", Handler: u.Get},
		{Method: "POST", Pattern: "/users", Handler: u.Create},
		{Method: "PUT", Pattern: "/users/{id}", Handler: u.Update},
		{Method: "DELETE", Pattern: "/users/{id}", Handler: u.Delete},

		{Method: "GET", Pattern: "/panic", Handler: Panic},
		{Method: "GET", Pattern: "/slow", Handler: Slow},

		{Method: "GET", Pattern: "/openapi.json", Handler: DocsSpec, Undocumented: true},
	}
}

func Register(mux *http.ServeMux, st store.Store) {
	for _, rt := range Routes(st) {
		mux.HandleFunc(rt.Method+" "+rt.Pattern, rt.Handler)
	}
	// Swagger UI is a subtree (serves index.html + static assets); register
	// separately. ServeMux auto-redirects GET /docs -> GET /docs/.
	mux.Handle("GET /docs/", DocsUI())
}
```

- [ ] **Step 2: Simplify `Root` and fix its route list**

Replace the `Root` function in `internal/handlers/hello.go` with:

```go
func Root(w http.ResponseWriter, r *http.Request) {
	respond.JSON(w, http.StatusOK, map[string]any{
		"name":    "go-laravelcloud",
		"message": "A Go API showcasing Go on Laravel Cloud.",
		"docs":    "/docs",
		"openapi": "/openapi.json",
	})
}
```

(Leave `Hello` unchanged.)

- [ ] **Step 3: Fix utility-handler `respond.Error` call sites**

In `internal/handlers/random.go`, change:

```go
respond.Error(w, http.StatusInternalServerError, "failed to read random bytes")
```
to:
```go
respond.Error(w, r, http.StatusInternalServerError, "failed to read random bytes")
```

In `internal/handlers/echo.go`, change:

```go
respond.Error(w, http.StatusBadRequest, "could not read body")
```
to:
```go
respond.Error(w, r, http.StatusBadRequest, "could not read body")
```

- [ ] **Step 4: Build check (after Task 7's docs.go exists)**

Run: `go build ./...`
Expected: success. If `DocsSpec`/`DocsUI` are undefined, complete Task 7 first.

- [ ] **Step 5: Run the handler tests**

Run: `go test ./internal/handlers/ -run TestUsers -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/handlers/handlers.go internal/handlers/hello.go internal/handlers/random.go internal/handlers/echo.go
git commit -m "Add route registry and inject store into handlers"
```

---

### Task 7: OpenAPI spec, embedded Swagger UI, and docs handlers

**Files:**
- Create: `api/openapi.json`
- Create: `api/embed.go`
- Create: `api/swagger-ui/index.html`
- Download: `api/swagger-ui/swagger-ui-bundle.js`, `api/swagger-ui/swagger-ui.css`
- Create: `internal/handlers/docs.go`

**Interfaces:**
- Consumes: `api.OpenAPISpec []byte`, `api.SwaggerUI embed.FS`.
- Produces:
  - `handlers.DocsSpec(w http.ResponseWriter, r *http.Request)` — serves `api.OpenAPISpec` with `Content-Type: application/json`.
  - `handlers.DocsUI() http.Handler` — serves the embedded Swagger UI subtree under `/docs/`.

- [ ] **Step 1: Download pinned Swagger UI assets**

Run:
```bash
mkdir -p api/swagger-ui
curl -fsSL https://unpkg.com/swagger-ui-dist@5.17.14/swagger-ui-bundle.js -o api/swagger-ui/swagger-ui-bundle.js
curl -fsSL https://unpkg.com/swagger-ui-dist@5.17.14/swagger-ui.css -o api/swagger-ui/swagger-ui.css
```
Verify both files are non-trivial:
```bash
wc -c api/swagger-ui/swagger-ui-bundle.js api/swagger-ui/swagger-ui.css
```
Expected: bundle > 1,000,000 bytes; css > 100,000 bytes.

- [ ] **Step 2: Write the Swagger UI page**

```html
<!-- api/swagger-ui/index.html -->
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <title>go-laravelcloud API docs</title>
  <link rel="stylesheet" href="./swagger-ui.css" />
  <style>body { margin: 0; }</style>
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="./swagger-ui-bundle.js"></script>
  <script>
    window.onload = function () {
      window.ui = SwaggerUIBundle({
        url: "/openapi.json",
        dom_id: "#swagger-ui",
        presets: [SwaggerUIBundle.presets.apis],
        layout: "BaseLayout",
      });
    };
  </script>
</body>
</html>
```

- [ ] **Step 3: Write the embed file**

```go
// api/embed.go
package api

import "embed"

//go:embed openapi.json
var OpenAPISpec []byte

//go:embed all:swagger-ui
var SwaggerUI embed.FS
```

- [ ] **Step 4: Write the docs handlers**

```go
// internal/handlers/docs.go
package handlers

import (
	"io/fs"
	"net/http"

	"github.com/alfonso/go-laravelcloud/api"
)

func DocsSpec(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, _ = w.Write(api.OpenAPISpec)
}

func DocsUI() http.Handler {
	sub, err := fs.Sub(api.SwaggerUI, "swagger-ui")
	if err != nil {
		panic(err) // embedded FS is known at build time; a failure here is a build bug
	}
	return http.StripPrefix("/docs/", http.FileServer(http.FS(sub)))
}
```

- [ ] **Step 5: Write the OpenAPI 3.1 spec**

```json
// api/openapi.json
{
  "openapi": "3.1.0",
  "info": {
    "title": "go-laravelcloud API",
    "version": "1.0.0",
    "description": "A stdlib-only Go REST API showcasing Go on Laravel Cloud. Spec and Swagger UI are embedded in the binary."
  },
  "servers": [{ "url": "/", "description": "This server" }],
  "paths": {
    "/hello": {
      "get": {
        "summary": "Greet a name",
        "parameters": [
          { "name": "name", "in": "query", "required": false, "schema": { "type": "string" } }
        ],
        "responses": { "200": { "description": "Greeting", "content": { "application/json": { "schema": { "type": "object", "properties": { "message": { "type": "string" } } } } } } }
      }
    },
    "/health": {
      "get": {
        "summary": "Health check",
        "responses": { "200": { "description": "Service healthy", "content": { "application/json": { "schema": { "type": "object", "properties": { "status": { "type": "string" }, "uptime_sec": { "type": "integer" } } } } } } }
      }
    },
    "/version": {
      "get": {
        "summary": "Build and runtime version info",
        "responses": { "200": { "description": "Version info", "content": { "application/json": { "schema": { "type": "object" } } } } }
      }
    },
    "/time": {
      "get": {
        "summary": "Current server time",
        "responses": { "200": { "description": "Time info", "content": { "application/json": { "schema": { "type": "object" } } } } }
      }
    },
    "/random": {
      "get": {
        "summary": "Random values",
        "parameters": [
          { "name": "bytes", "in": "query", "required": false, "schema": { "type": "integer", "minimum": 1, "maximum": 1024 } }
        ],
        "responses": { "200": { "description": "Random data", "content": { "application/json": { "schema": { "type": "object" } } } } }
      }
    },
    "/quote": {
      "get": {
        "summary": "Random programming quote",
        "responses": { "200": { "description": "A quote", "content": { "application/json": { "schema": { "type": "object", "properties": { "quote": { "type": "string" } } } } } } }
      }
    },
    "/echo": {
      "post": {
        "summary": "Echo the request back",
        "requestBody": { "required": false, "content": { "application/json": { "schema": { "type": "object" } } } },
        "responses": { "200": { "description": "Echoed request", "content": { "application/json": { "schema": { "type": "object" } } } } }
      }
    },
    "/users": {
      "get": {
        "summary": "List users",
        "parameters": [
          { "name": "q", "in": "query", "required": false, "schema": { "type": "string" }, "description": "Case-insensitive filter over name and email" },
          { "name": "limit", "in": "query", "required": false, "schema": { "type": "integer", "minimum": 1, "maximum": 100, "default": 20 } },
          { "name": "offset", "in": "query", "required": false, "schema": { "type": "integer", "minimum": 0, "default": 0 } }
        ],
        "responses": {
          "200": {
            "description": "A page of users",
            "content": { "application/json": { "schema": {
              "type": "object",
              "properties": {
                "users": { "type": "array", "items": { "$ref": "#/components/schemas/User" } },
                "count": { "type": "integer" },
                "limit": { "type": "integer" },
                "offset": { "type": "integer" }
              }
            } } }
          }
        }
      },
      "post": {
        "summary": "Create a user",
        "requestBody": { "required": true, "content": { "application/json": { "schema": { "$ref": "#/components/schemas/UserInput" } } } },
        "responses": {
          "201": { "description": "Created", "content": { "application/json": { "schema": { "$ref": "#/components/schemas/User" } } } },
          "400": { "$ref": "#/components/responses/BadRequest" }
        }
      }
    },
    "/users/{id}": {
      "get": {
        "summary": "Get a user by ID",
        "parameters": [{ "name": "id", "in": "path", "required": true, "schema": { "type": "integer" } }],
        "responses": {
          "200": { "description": "The user", "content": { "application/json": { "schema": { "$ref": "#/components/schemas/User" } } } },
          "400": { "$ref": "#/components/responses/BadRequest" },
          "404": { "$ref": "#/components/responses/NotFound" }
        }
      },
      "put": {
        "summary": "Update a user",
        "parameters": [{ "name": "id", "in": "path", "required": true, "schema": { "type": "integer" } }],
        "requestBody": { "required": true, "content": { "application/json": { "schema": { "$ref": "#/components/schemas/UserInput" } } } },
        "responses": {
          "200": { "description": "Updated", "content": { "application/json": { "schema": { "$ref": "#/components/schemas/User" } } } },
          "400": { "$ref": "#/components/responses/BadRequest" },
          "404": { "$ref": "#/components/responses/NotFound" }
        }
      },
      "delete": {
        "summary": "Delete a user",
        "parameters": [{ "name": "id", "in": "path", "required": true, "schema": { "type": "integer" } }],
        "responses": {
          "204": { "description": "Deleted" },
          "400": { "$ref": "#/components/responses/BadRequest" },
          "404": { "$ref": "#/components/responses/NotFound" }
        }
      }
    },
    "/panic": {
      "get": {
        "summary": "Trigger a panic (demonstrates recovery middleware)",
        "responses": { "500": { "$ref": "#/components/responses/ServerError" } }
      }
    },
    "/slow": {
      "get": {
        "summary": "Slow response (demonstrates timeouts / cancellation)",
        "parameters": [{ "name": "ms", "in": "query", "required": false, "schema": { "type": "integer" } }],
        "responses": { "200": { "description": "Slept", "content": { "application/json": { "schema": { "type": "object", "properties": { "slept_ms": { "type": "integer" } } } } } } }
      }
    }
  },
  "components": {
    "schemas": {
      "User": {
        "type": "object",
        "properties": {
          "id": { "type": "integer" },
          "name": { "type": "string" },
          "email": { "type": "string", "format": "email" },
          "created_at": { "type": "string", "format": "date-time" }
        },
        "required": ["id", "name", "email", "created_at"]
      },
      "UserInput": {
        "type": "object",
        "properties": {
          "name": { "type": "string", "minLength": 1, "maxLength": 100 },
          "email": { "type": "string", "format": "email" }
        },
        "required": ["name", "email"]
      },
      "APIError": {
        "type": "object",
        "properties": {
          "error": { "type": "string" },
          "detail": { "type": "string" },
          "status": { "type": "integer" },
          "request_id": { "type": "string" }
        },
        "required": ["error", "detail", "status"]
      }
    },
    "responses": {
      "BadRequest": { "description": "Invalid request", "content": { "application/json": { "schema": { "$ref": "#/components/schemas/APIError" } } } },
      "NotFound": { "description": "Resource not found", "content": { "application/json": { "schema": { "$ref": "#/components/schemas/APIError" } } } },
      "ServerError": { "description": "Internal server error", "content": { "application/json": { "schema": { "$ref": "#/components/schemas/APIError" } } } }
    }
  }
}
```

- [ ] **Step 6: Validate the JSON parses**

Run: `go run -tags=none ./... >/dev/null 2>&1; python3 -c "import json;json.load(open('api/openapi.json'));print('valid json')"`
Expected: `valid json` (the first command is a no-op guard; the JSON check is what matters).

- [ ] **Step 7: Build check**

Run: `go build ./...`
Expected: success (this compiles the embed + docs handlers + Task 6 registry together).

- [ ] **Step 8: Commit**

```bash
git add api/ internal/handlers/docs.go
git commit -m "Add embedded OpenAPI spec and Swagger UI docs endpoints"
```

---

### Task 8: Spec-drift contract test

**Files:**
- Create: `internal/handlers/contract_test.go`

**Interfaces:**
- Consumes: `handlers.Routes` (Task 6), `api.OpenAPISpec` (Task 7), `store.NewMemoryStore` (Task 2).
- Produces: a test asserting the set of documented `{method, path}` routes exactly equals the set of `{method, path}` operations in the spec.

- [ ] **Step 1: Write the test**

```go
// internal/handlers/contract_test.go
package handlers

import (
	"encoding/json"
	"sort"
	"strings"
	"testing"

	"github.com/alfonso/go-laravelcloud/api"
	"github.com/alfonso/go-laravelcloud/internal/store"
)

func TestOpenAPISpecMatchesRoutes(t *testing.T) {
	// Documented routes registered by the app.
	want := map[string]bool{}
	for _, rt := range Routes(store.NewMemoryStore()) {
		if rt.Undocumented {
			continue
		}
		want[strings.ToUpper(rt.Method)+" "+rt.Pattern] = true
	}

	// Operations declared in the spec.
	var spec struct {
		Paths map[string]map[string]json.RawMessage `json:"paths"`
	}
	if err := json.Unmarshal(api.OpenAPISpec, &spec); err != nil {
		t.Fatalf("parse openapi.json: %v", err)
	}
	got := map[string]bool{}
	for path, ops := range spec.Paths {
		for method := range ops {
			got[strings.ToUpper(method)+" "+path] = true
		}
	}

	if missing := diff(want, got); len(missing) > 0 {
		t.Errorf("routes registered but MISSING from openapi.json: %v", missing)
	}
	if extra := diff(got, want); len(extra) > 0 {
		t.Errorf("operations in openapi.json with NO registered route: %v", extra)
	}
}

func diff(a, b map[string]bool) []string {
	var out []string
	for k := range a {
		if !b[k] {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}
```

- [ ] **Step 2: Run the test**

Run: `go test ./internal/handlers/ -run TestOpenAPISpecMatchesRoutes -v`
Expected: PASS. If it fails, the failure message lists exactly which routes/operations are out of sync — fix `api/openapi.json` or `Routes` until they match.

- [ ] **Step 3: Commit**

```bash
git add internal/handlers/contract_test.go
git commit -m "Add contract test asserting routes match OpenAPI spec"
```

---

### Task 9: Wire everything in main.go (PORT-aware bootstrap)

**Files:**
- Modify: `main.go`

**Interfaces:**
- Consumes: `store.NewMemoryStore` (Task 2), `handlers.Register(mux, st)` (Task 6), `middleware.CORS` (Task 4).
- Produces: a server that constructs the store, registers routes, applies the middleware chain (`Recovery` → `CORS` → `Logging` → `RequestID`), and binds to `:$PORT` (falling back to `ADDR`, then `:9114`).

- [ ] **Step 1: Update main.go**

```go
// main.go
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alfonso/go-laravelcloud/internal/handlers"
	"github.com/alfonso/go-laravelcloud/internal/middleware"
	"github.com/alfonso/go-laravelcloud/internal/store"
)

func main() {
	mux := http.NewServeMux()
	handlers.Register(mux, store.NewMemoryStore())

	chain := middleware.Chain(
		middleware.Recovery,
		middleware.CORS,
		middleware.Logging,
		middleware.RequestID,
	)

	addr := resolveAddr()
	srv := &http.Server{
		Addr:              addr,
		Handler:           chain(mux),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		log.Printf("listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server failed: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	log.Println("shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("shutdown error: %v", err)
	}
}

// resolveAddr prefers the platform-injected PORT, then ADDR, then a default.
func resolveAddr() string {
	if p := os.Getenv("PORT"); p != "" {
		return ":" + p
	}
	if a := os.Getenv("ADDR"); a != "" {
		return a
	}
	return ":9114"
}
```

Note on `WriteTimeout`: `/slow` accepts up to 30s but `WriteTimeout` is 15s, so long `/slow` calls are cut off by design — this is fine for a demo endpoint and left as-is.

- [ ] **Step 2: Build and run the full suite**

Run: `go build ./... && go vet ./... && go test ./...`
Expected: build success, no vet issues, all tests PASS.

- [ ] **Step 3: Smoke-test the running server**

Run:
```bash
PORT=9114 go run . &
sleep 1
curl -s localhost:9114/users | head -c 200; echo
curl -s -o /dev/null -w "%{http_code}\n" localhost:9114/openapi.json
curl -s -o /dev/null -w "%{http_code}\n" localhost:9114/docs/
curl -s -X POST localhost:9114/users -d '{"name":"Test","email":"t@example.com"}' -o /dev/null -w "%{http_code}\n"
kill %1
```
Expected: users JSON printed; `/openapi.json` → `200`; `/docs/` → `200`; POST → `201`.

- [ ] **Step 4: Commit**

```bash
git add main.go
git commit -m "Wire store, CORS, and PORT-aware bootstrap into main"
```

---

### Task 10: Documentation (README, Makefile, Dockerfile)

**Files:**
- Rewrite: `README.md`
- Modify: `Makefile` (add `curl`-based smoke target)
- Modify: `Dockerfile` (comment on PORT)

**Interfaces:** none (docs only).

- [ ] **Step 1: Write the README**

```markdown
# go-laravelcloud

A small, **dependency-free** Go REST API that showcases running Go on **Laravel Cloud**.

- Standard library only — `go.mod` has zero `require` directives.
- Go 1.22+ `net/http` routing (method + path patterns), no framework.
- OpenAPI 3.1 spec and Swagger UI **embedded into the binary** (`go:embed`) — the whole thing ships as one static file.
- Graceful shutdown, structured JSON errors, request IDs, CORS, panic recovery.

## Endpoints

Interactive docs: **`/docs`** · Raw spec: **`/openapi.json`**

| Method | Path          | Description                       |
|--------|---------------|-----------------------------------|
| GET    | /users        | List users (`?q`, `?limit`, `?offset`) |
| GET    | /users/{id}   | Get a user                        |
| POST   | /users        | Create a user                     |
| PUT    | /users/{id}   | Update a user                     |
| DELETE | /users/{id}   | Delete a user                     |
| GET    | /hello, /health, /version, /time, /random, /quote | Utility/demo endpoints |
| POST   | /echo         | Echo the request                  |
| GET    | /panic, /slow | Middleware demos                  |

## Run locally

```bash
make run                 # go run .  (listens on :9114)
make test                # go test ./...
open http://localhost:9114/docs
```

Example:

```bash
curl -s localhost:9114/users | jq
curl -s -X POST localhost:9114/users -d '{"name":"Ada","email":"ada@example.com"}' | jq
```

## Deploy to Laravel Cloud

This app deploys as a Docker image and reads its listen port from `PORT`
(falling back to `ADDR`, then `:9114`).

1. Push this repo to GitHub and create a new application in Laravel Cloud,
   pointing at your repository/branch.
2. The included `Dockerfile` builds a distroless static binary — no runtime
   language configuration needed.
3. Laravel Cloud injects a `PORT` for the web process; the app binds to it
   automatically. Confirm the exposed port in the environment's network
   settings matches.
4. Deploy. Hit `/docs` on your assigned domain to see the live interactive API.

> Persistence is in-memory behind a `store.Store` interface. To back it with
> Laravel Cloud managed Postgres, add a `PostgresStore` implementing the same
> interface (using the injected `DB_*` env vars) and construct it in `main.go` —
> no handler changes required.

## Build the container

```bash
make docker
docker run --rm -e PORT=8080 -p 8080:8080 go-laravelcloud:dev
```
```

- [ ] **Step 2: Add a Makefile smoke target**

Append to `Makefile`:

```makefile
smoke:
	@PORT=9114 go run . & echo $$! > /tmp/golc.pid; sleep 1; \
	curl -s -o /dev/null -w "GET /openapi.json -> %{http_code}\n" localhost:9114/openapi.json; \
	curl -s -o /dev/null -w "GET /docs/ -> %{http_code}\n" localhost:9114/docs/; \
	kill `cat /tmp/golc.pid`
```

Add `smoke` to the `.PHONY` line:

```makefile
.PHONY: run build test fmt vet tidy docker clean smoke
```

- [ ] **Step 3: Comment the Dockerfile PORT expectation**

In `Dockerfile`, replace the `EXPOSE 9114` line with:

```dockerfile
# App binds to $PORT if set (Laravel Cloud injects one), else :9114.
EXPOSE 9114
```

- [ ] **Step 4: Verify docs build and suite is green**

Run: `go build ./... && go test ./...`
Expected: success, all PASS.

- [ ] **Step 5: Commit**

```bash
git add README.md Makefile Dockerfile
git commit -m "Document API, deploy steps, and add smoke target"
```

---

## Self-Review (completed against the spec)

- **Spec coverage:** OpenAPI approach (Tasks 7, 8) · users resource with pagination/filter/PUT/validation (Tasks 1, 5) · in-memory `Store` interface + Postgres seam (Task 2, README) · structured errors with request_id (Tasks 1, 3) · CORS (Task 4) · PORT-aware bootstrap (Task 9) · embedded spec + Swagger UI (Task 7) · contract test (Task 8) · README/deploy docs (Task 10) · utility endpoints documented (Task 7). All spec sections map to a task.
- **Placeholder scan:** none — every code and command step is complete.
- **Type consistency:** `respond.Error(w, r, status, detail)` signature is used consistently across Tasks 3, 5, 6. `Store` method signatures (with `context.Context`) match between Tasks 2 and 5. `Route` struct fields match between Tasks 6 and 8. `DocsSpec`/`DocsUI` referenced in Task 6 are defined in Task 7 (cross-task dependency called out explicitly).
- **Cross-task build ordering:** Tasks 5 and 6 leave `internal/handlers` non-compiling until Task 7 adds `docs.go`; this is called out in Tasks 5, 6, and 7 with the correct gate commands.
```
