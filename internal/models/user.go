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
