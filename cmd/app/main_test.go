package main

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"atad-project/internal/version"
)

func TestVersionReportsContractFields(t *testing.T) {
	var b strings.Builder
	if err := version.WriteJSON(&b); err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}
	for _, field := range []string{`"app"`, `"version"`, `"commit"`, `"built_at"`} {
		if !strings.Contains(b.String(), field) {
			t.Errorf("version output is missing %s: %s", field, b.String())
		}
	}
}

func TestContractRoutes(t *testing.T) {
	mux := contractRoutes(time.Now().Add(-2*time.Second), func() {})
	for _, test := range []struct {
		method, path, content string
		status                int
	}{
		{"GET", "/", "text/html", 200},
		{"GET", "/health", "application/json", 200},
		{"GET", "/version", "application/json", 200},
		{"POST", "/reset", "", 204},
		{"GET", "/reset", "", 405},
		{"GET", "/missing", "", 404},
	} {
		t.Run(test.method+" "+test.path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, httptest.NewRequest(test.method, test.path, nil))
			if rec.Code != test.status || !strings.HasPrefix(rec.Header().Get("Content-Type"), test.content) {
				t.Fatalf("status %d, content type %q", rec.Code, rec.Header().Get("Content-Type"))
			}
			switch test.path {
			case "/":
				for _, text := range []string{version.App, "Gotiu Bettina Catalina", "Master Cloud Technologies", "Project 10"} {
					if !strings.Contains(rec.Body.String(), text) {
						t.Errorf("identity page missing %q", text)
					}
				}
			case "/health":
				var health struct {
					Status string `json:"status"`
					Uptime int    `json:"uptime_seconds"`
				}
				if err := json.Unmarshal(rec.Body.Bytes(), &health); err != nil || health.Status != "ok" || health.Uptime < 2 {
					t.Fatalf("invalid health response %q: %v", rec.Body, err)
				}
			case "/version":
				var info version.Info
				if err := json.Unmarshal(rec.Body.Bytes(), &info); err != nil || info != version.Current() {
					t.Fatalf("invalid version response %q: %v", rec.Body, err)
				}
			case "/reset":
				if test.status == 204 && rec.Body.Len() != 0 {
					t.Errorf("reset response should be empty, got %q", rec.Body)
				}
			}
		})
	}
}
