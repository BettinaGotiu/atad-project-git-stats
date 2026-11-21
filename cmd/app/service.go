package main

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"atad-project/internal/jobs"
)

type service struct {
	root string
	jobs *jobs.Store
}

func (s service) routes() *http.ServeMux {
	mux := contractRoutes(time.Now(), s.jobs.Reset)
	mux.HandleFunc("GET /stats", s.liveStats)
	mux.HandleFunc("POST /repositories", s.createJob)
	mux.HandleFunc("GET /repositories/{id}", s.jobReport)
	mux.HandleFunc("GET /repositories/{id}/{view}", s.jobReport)
	mux.HandleFunc("PUT /identities", s.identities)
	return mux
}

// repository resolves symlinks before checking that the requested checkout belongs to the configured root.
func (s service) repository(name string) (string, error) {
	root, err := filepath.Abs(s.root)
	if err != nil {
		return "", err
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return "", errors.New("repository root is unavailable")
	}
	if !filepath.IsAbs(name) {
		name = filepath.Join(root, name)
	}
	repo, err := filepath.EvalSymlinks(name)
	if err != nil {
		return "", errors.New("repository path does not exist")
	}
	relative, err := filepath.Rel(root, repo)
	if err != nil || !filepath.IsLocal(relative) {
		return "", errors.New("path is outside the repository root")
	}
	info, err := os.Stat(repo)
	if err != nil || !info.IsDir() {
		return "", errors.New("repository path must be a directory")
	}
	return repo, nil
}

func problem(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]string{"error": code, "message": message})
}
