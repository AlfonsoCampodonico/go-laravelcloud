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
