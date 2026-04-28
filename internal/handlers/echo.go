package handlers

import (
	"io"
	"net/http"
	"time"

	"github.com/alfonso/go-laravelcloud/internal/respond"
)

func Echo(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "could not read body")
		return
	}
	defer r.Body.Close()

	headers := make(map[string]string, len(r.Header))
	for k, v := range r.Header {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}

	respond.JSON(w, http.StatusOK, map[string]any{
		"method":  r.Method,
		"path":    r.URL.Path,
		"query":   r.URL.RawQuery,
		"headers": headers,
		"body":    string(body),
		"length":  len(body),
	})
}

func Slow(w http.ResponseWriter, r *http.Request) {
	d := 2 * time.Second
	if v := r.URL.Query().Get("ms"); v != "" {
		if parsed, err := time.ParseDuration(v + "ms"); err == nil && parsed <= 30*time.Second {
			d = parsed
		}
	}
	select {
	case <-time.After(d):
		respond.JSON(w, http.StatusOK, map[string]any{"slept_ms": d.Milliseconds()})
	case <-r.Context().Done():
		return
	}
}

func Panic(w http.ResponseWriter, r *http.Request) {
	panic("intentional panic from /panic — recovery middleware should catch this")
}
