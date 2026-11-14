package gitlog

import (
	"bufio"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestParsePreservesPaths(t *testing.T) {
	data := "\x00Name\x00mail@example.test\x001\x00\x00\n1\t2\ta\tb\nc\x00-\t-\timage.png\x00"
	var commits []Commit
	err := parse(bufio.NewReader(strings.NewReader(data)), func(c Commit) { commits = append(commits, c) })
	if err != nil || len(commits) != 1 || len(commits[0].Files) != 2 {
		t.Fatalf("commits = %+v, error = %v", commits, err)
	}
	if got := commits[0].Files[0]; got != (File{Path: "a\tb\nc", Added: 1, Removed: 2}) {
		t.Errorf("text file = %+v", got)
	}
	if got := commits[0].Files[1]; got != (File{Path: "image.png"}) {
		t.Errorf("binary file = %+v", got)
	}
}

func TestReadHistory(t *testing.T) {
	dir := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir,
			"-c", "user.name=Fixture", "-c", "user.email=fixture@example.test"}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	git("init", "-q", "-b", "main")
	err := Read(t.Context(), dir, "", func(c Commit) { t.Errorf("empty repository produced %+v", c) })
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("hello\n"), 0600); err != nil {
		t.Fatal(err)
	}
	git("add", "hello.txt")
	git("commit", "-qm", "root")
	git("checkout", "-qb", "feature")
	git("commit", "--allow-empty", "-qm", "empty feature")
	git("checkout", "-q", "main")
	git("merge", "--no-ff", "-qm", "merge", "feature")

	var commits []Commit
	err = Read(t.Context(), dir, "", func(c Commit) { commits = append(commits, c) })
	if err != nil || len(commits) != 3 {
		t.Fatalf("commits = %+v, error = %v", commits, err)
	}
	if !commits[0].Merge || len(commits[0].Files) != 0 || len(commits[1].Files) != 0 {
		t.Errorf("merge and empty commit should have no diff: %+v", commits[:2])
	}
	if files := commits[2].Files; len(files) != 1 || files[0] != (File{Path: "hello.txt", Added: 1}) {
		t.Errorf("root diff = %+v", files)
	}
	if err := Read(t.Context(), dir, "--all", func(Commit) {}); !errors.Is(err, ErrRevision) {
		t.Errorf("invalid revision = %v", err)
	}
	if err := Read(t.Context(), t.TempDir(), "", func(Commit) {}); !errors.Is(err, ErrRepository) {
		t.Errorf("invalid repository = %v", err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := Read(ctx, dir, "", func(Commit) {}); !errors.Is(err, context.Canceled) {
		t.Errorf("canceled read = %v", err)
	}
}

func TestPrettierHistory(t *testing.T) {
	repo := os.Getenv("PRETTIER_REPO")
	if repo == "" {
		t.Skip("set PRETTIER_REPO to run the reference-repository check")
	}
	count := 0
	err := Read(t.Context(), repo, "6ba1dee64a6cacabd237746f9c62060b10747aed", func(Commit) { count++ })
	if err != nil || count != 12148 {
		t.Fatalf("read %d commits, error = %v", count, err)
	}
}
