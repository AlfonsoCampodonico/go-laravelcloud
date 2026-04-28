package handlers

import (
	"net/http"
	"runtime"
	"time"

	"github.com/alfonso/go-laravelcloud/internal/respond"
	"github.com/alfonso/go-laravelcloud/internal/version"
)

var startTime = time.Now()

func Health(w http.ResponseWriter, r *http.Request) {
	respond.JSON(w, http.StatusOK, map[string]any{
		"status":     "ok",
		"uptime_sec": int(time.Since(startTime).Seconds()),
	})
}

func Version(w http.ResponseWriter, r *http.Request) {
	respond.JSON(w, http.StatusOK, map[string]any{
		"version":   version.Version,
		"commit":    version.Commit,
		"buildDate": version.BuildDate,
		"goVersion": runtime.Version(),
		"goos":      runtime.GOOS,
		"goarch":    runtime.GOARCH,
	})
}

func Time(w http.ResponseWriter, r *http.Request) {
	now := time.Now().UTC()
	respond.JSON(w, http.StatusOK, map[string]any{
		"utc":     now.Format(time.RFC3339Nano),
		"unix":    now.Unix(),
		"unixMs":  now.UnixMilli(),
		"weekday": now.Weekday().String(),
	})
}
