package main

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"atad-project/internal/version"
)

func contractRoutes(started time.Time, reset func()) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"status":         "ok",
			"uptime_seconds": int(time.Since(started).Seconds()),
		})
	})
	mux.HandleFunc("GET /version", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = version.WriteJSON(w)
	})
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = io.WriteString(w, `<!doctype html><html lang="en"><meta charset="utf-8">
<title>Git Repository Statistics</title><body>
<h1>Git Repository Statistics</h1>
<p>Gotiu Bettina Catalina, group Master Cloud Technologies and Internet of Things (Master Year II)</p>
<p>Project 10</p>
</body></html>`)
	})
	mux.HandleFunc("POST /reset", func(w http.ResponseWriter, r *http.Request) {
		reset()
		w.WriteHeader(http.StatusNoContent)
	})
	return mux
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
