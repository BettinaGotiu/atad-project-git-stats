package gitlog

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
)

var githubURL = regexp.MustCompile(`^https://github\.com/[A-Za-z0-9_-]+/[A-Za-z0-9_-][A-Za-z0-9_.-]*/?$`)

func ValidURL(url string) bool { return githubURL.MatchString(url) }

// ReadURL clones public GitHub history into a private temporary directory for this analysis.
func ReadURL(ctx context.Context, url string, visit func(Commit)) error {
	if !ValidURL(url) {
		return fmt.Errorf("url must name a public HTTPS GitHub repository")
	}
	dir, err := os.MkdirTemp("", "git-stats-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	repo := filepath.Join(dir, "repo.git")
	cmd := exec.CommandContext(ctx, "git", "-c", "http.followRedirects=false", "-c", "credential.helper=", "clone", "--bare", "--single-branch", "--no-tags", "--", url, repo)
	cmd.Env = append(command(ctx, repo).Env, "GIT_ALLOW_PROTOCOL=https")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("could not clone the public repository: %w", err)
	}
	return Read(ctx, repo, "", visit)
}
