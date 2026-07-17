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
