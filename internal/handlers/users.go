package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/alfonso/go-laravelcloud/internal/models"
	"github.com/alfonso/go-laravelcloud/internal/respond"
)

var (
	userMu    sync.RWMutex
	userStore = map[int]models.User{
		1: {ID: 1, Name: "Ada Lovelace", Email: "ada@example.com", CreatedAt: time.Now().Add(-72 * time.Hour)},
		2: {ID: 2, Name: "Alan Turing", Email: "alan@example.com", CreatedAt: time.Now().Add(-48 * time.Hour)},
		3: {ID: 3, Name: "Grace Hopper", Email: "grace@example.com", CreatedAt: time.Now().Add(-24 * time.Hour)},
	}
	nextUserID = 4
)

func ListUsers(w http.ResponseWriter, r *http.Request) {
	userMu.RLock()
	defer userMu.RUnlock()
	users := make([]models.User, 0, len(userStore))
	for _, u := range userStore {
		users = append(users, u)
	}
	respond.JSON(w, http.StatusOK, map[string]any{"users": users, "count": len(users)})
}

func GetUser(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	userMu.RLock()
	defer userMu.RUnlock()
	u, ok := userStore[id]
	if !ok {
		respond.Error(w, http.StatusNotFound, "user not found")
		return
	}
	respond.JSON(w, http.StatusOK, u)
}

func CreateUser(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid json body")
		return
	}
	if input.Name == "" || input.Email == "" {
		respond.Error(w, http.StatusBadRequest, "name and email are required")
		return
	}
	userMu.Lock()
	defer userMu.Unlock()
	u := models.User{
		ID:        nextUserID,
		Name:      input.Name,
		Email:     input.Email,
		CreatedAt: time.Now(),
	}
	userStore[nextUserID] = u
	nextUserID++
	respond.JSON(w, http.StatusCreated, u)
}

func DeleteUser(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	userMu.Lock()
	defer userMu.Unlock()
	if _, ok := userStore[id]; !ok {
		respond.Error(w, http.StatusNotFound, "user not found")
		return
	}
	delete(userStore, id)
	w.WriteHeader(http.StatusNoContent)
}
