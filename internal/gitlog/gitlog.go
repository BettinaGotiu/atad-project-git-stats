package gitlog

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var (
	ErrRepository = errors.New("path must be the root of a Git repository")
	ErrRevision   = errors.New("revision must identify a commit in the repository")
)

type Commit struct {
	Name, Email string
	AuthoredAt  time.Time
	Merge       bool
	Files       []File
}

type File struct {
	Path           string
	Added, Removed int64
}

// Read streams commits reachable from revision. A blank revision uses HEAD.
func Read(ctx context.Context, repo, revision string, visit func(Commit)) error {
	if revision != "" && !isObjectID(revision) {
		return ErrRevision
	}
	kind, err := command(ctx, repo, "rev-parse", "--is-bare-repository", "--is-inside-work-tree", "--show-prefix").Output()
	if ctx.Err() != nil {
		return ctx.Err()
	}
	var exit *exec.ExitError
	switch {
	case errors.As(err, &exit):
		return ErrRepository
	case err != nil:
		return err
	}
	switch strings.ReplaceAll(string(kind), "\r\n", "\n") {
	case "false\ntrue\n\n", "true\nfalse\n\n":
	default:
		return ErrRepository
	}

	target := revision
	if target == "" {
		target = "HEAD"
	}
	resolved, err := command(ctx, repo, "rev-parse", "--verify", "--quiet", target+"^{commit}").Output()
	if ctx.Err() != nil {
		return ctx.Err()
	}
	switch {
	case errors.As(err, &exit) && exit.ExitCode() == 1 && revision == "":
		return nil // An unborn HEAD has no history.
	case errors.As(err, &exit):
		return ErrRevision
	case err != nil:
		return err
	}

	cmd := command(ctx, repo, "log", "--topo-order", "--format=%x00%an%x00%ae%x00%at%x00%P",
		"--numstat", "-z", "--root", "--diff-merges=off", "--no-renames",
		"--no-ext-diff", "--no-textconv", "--no-notes", "--no-show-signature", "--no-color",
		strings.TrimSpace(string(resolved)), "--")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	parseErr := parse(bufio.NewReader(stdout), visit)
	if parseErr != nil {
		_ = cmd.Process.Kill()
	}
	waitErr := cmd.Wait()
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if waitErr != nil && stderr.Len() != 0 {
		return fmt.Errorf("git log: %s", strings.TrimSpace(stderr.String()))
	}
	if parseErr != nil {
		return parseErr
	}
	return waitErr
}

// command keeps inherited Git settings from redirecting history or starting a fetch.
func command(ctx context.Context, repo string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "git", append([]string{
		"--no-pager", "--no-replace-objects", "-c", "safe.directory=" + filepath.ToSlash(repo), "-C", repo,
	}, args...)...)
	cmd.Env = []string{
		"GIT_OPTIONAL_LOCKS=0", "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=" + os.DevNull,
		"GIT_TERMINAL_PROMPT=0", "GIT_NO_LAZY_FETCH=1", "GIT_ALLOW_PROTOCOL=",
	}
	for _, setting := range os.Environ() {
		if !strings.HasPrefix(strings.ToUpper(setting), "GIT_") {
			cmd.Env = append(cmd.Env, setting)
		}
	}
	return cmd
}

func isObjectID(value string) bool {
	if len(value) != 40 && len(value) != 64 {
		return false
	}
	for _, c := range value {
		if !strings.ContainsRune("0123456789abcdefABCDEF", c) {
			return false
		}
	}
	return true
}

// parse uses NUL delimiters so tabs and newlines in Git paths stay intact.
func parse(r *bufio.Reader, visit func(Commit)) error {
	var current *Commit
	for {
		token, err := field(r)
		if err == io.EOF {
			if current != nil {
				visit(*current)
			}
			return nil
		}
		if err != nil {
			return err
		}
		if token == "" {
			if current != nil {
				visit(*current)
			}
			current, err = readCommit(r)
		} else if current == nil {
			return errors.New("git log started without a commit")
		} else {
			var file File
			file, err = readFile(strings.TrimPrefix(token, "\n"))
			current.Files = append(current.Files, file)
		}
		if err != nil {
			return err
		}
	}
}

func field(r *bufio.Reader) (string, error) {
	token, err := r.ReadString(0)
	if err == io.EOF && token != "" {
		return "", io.ErrUnexpectedEOF
	}
	return strings.TrimSuffix(token, "\x00"), err
}

func readCommit(r *bufio.Reader) (*Commit, error) {
	var fields [4]string
	for i := range fields {
		value, err := field(r)
		if err != nil {
			return nil, fmt.Errorf("read Git commit: %w", err)
		}
		fields[i] = value
	}
	stamp, err := strconv.ParseInt(fields[2], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("read author timestamp: %w", err)
	}
	return &Commit{
		Name: fields[0], Email: fields[1], AuthoredAt: time.Unix(stamp, 0).UTC(),
		Merge: strings.Contains(fields[3], " "),
	}, nil
}

func readFile(token string) (File, error) {
	parts := strings.SplitN(token, "\t", 3)
	if len(parts) != 3 {
		return File{}, errors.New("invalid Git numstat record")
	}
	file := File{Path: parts[2]}
	if parts[0] == "-" && parts[1] == "-" {
		return file, nil
	}
	added, addErr := strconv.ParseInt(parts[0], 10, 64)
	removed, removeErr := strconv.ParseInt(parts[1], 10, 64)
	if addErr != nil || removeErr != nil {
		return File{}, errors.New("invalid Git line counts")
	}
	file.Added, file.Removed = added, removed
	return file, nil
}
