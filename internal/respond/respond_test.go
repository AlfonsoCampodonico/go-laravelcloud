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
