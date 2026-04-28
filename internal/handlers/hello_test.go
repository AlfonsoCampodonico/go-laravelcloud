package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHello_DefaultName(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/hello", nil)
	rec := httptest.NewRecorder()
	Hello(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["message"] != "hello world" {
		t.Errorf("message = %q, want 'hello world'", body["message"])
	}
}

func TestHello_WithName(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/hello?name=ada", nil)
	rec := httptest.NewRecorder()
	Hello(rec, req)

	var body map[string]string
	_ = json.NewDecoder(rec.Body).Decode(&body)
	if body["message"] != "hello ada" {
		t.Errorf("message = %q, want 'hello ada'", body["message"])
	}
}

func TestHealth(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	Health(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}
