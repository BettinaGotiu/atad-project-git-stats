package gitlog

import (
	"os/exec"
	"path/filepath"
	"testing"
)

func TestPublicRemoteAndBareHistory(t *testing.T) {
	for _, url := range []string{"file:///tmp/repo", "https://example.org/name/repo", "https://github.com/name/repo?x=1", "http://github.com/name/repo"} {
		if ValidURL(url) {
			t.Errorf("accepted %q", url)
		}
	}
	if !ValidURL("https://github.com/prettier/prettier.git") {
		t.Fatal("valid GitHub URL rejected")
	}
	repo := filepath.Join(t.TempDir(), "empty.git")
	if out, err := exec.Command("git", "init", "--bare", "-q", repo).CombinedOutput(); err != nil {
		t.Fatalf("%s: %v", out, err)
	}
	if err := Read(t.Context(), repo, "", func(Commit) { t.Fatal("unexpected commit") }); err != nil {
		t.Fatal(err)
	}
}
