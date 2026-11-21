package main

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"atad-project/internal/jobs"
)

func sampleRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", repo, "-c", "user.name=Sam", "-c", "user.email=sam@example.test"}, args...)...)
		if b, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %s (%v)", args, b, err)
		}
	}
	git("init", "-q", "-b", "main")
	if err := os.WriteFile(filepath.Join(repo, "hello.txt"), []byte("hello\n"), 0600); err != nil {
		t.Fatal(err)
	}
	git("add", "hello.txt")
	git("commit", "-qm", "Add greeting")
	git("checkout", "-qb", "topic")
	git("commit", "--allow-empty", "-qm", "Record review")
	git("checkout", "-q", "main")
	git("merge", "--no-ff", "-qm", "Merge topic", "topic")
	return repo
}

func testClient(t *testing.T) func(string, string, string, int) *httptest.ResponseRecorder {
	s := service{root: sampleRepo(t), jobs: jobs.New(nil)}
	t.Cleanup(s.jobs.Reset)
	handler := s.routes()
	return func(method, path, body string, code int) *httptest.ResponseRecorder {
		t.Helper()
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, httptest.NewRequest(method, path, strings.NewReader(body)))
		if rr.Code != code {
			t.Fatalf("%s %s returned %d: %s", method, path, rr.Code, rr.Body)
		}
		return rr
	}
}

func TestLiveStats(t *testing.T) {
	call := testClient(t)
	var report result
	response := call("GET", "/stats?period=week", "", 200)
	if err := json.Unmarshal(response.Body.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if report.Total != 3 || len(report.Churn) != 1 || report.Contributors[0].Added != 1 || report.Policy != "count" {
		t.Fatalf("unexpected report %+v", report)
	}
	call("GET", "/stats?period=year", "", 400)
	call("GET", "/stats?repo=missing", "", 422)
	txt := call("GET", "/stats?format=text&glob=*.txt", "", 200)
	if !strings.Contains(txt.Body.String(), "Commits: 1") {
		t.Fatal(txt.Body)
	}
}
