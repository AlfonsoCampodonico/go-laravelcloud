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
