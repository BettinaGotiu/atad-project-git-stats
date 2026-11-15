package stats

import (
	"fmt"
	"io/fs"
	"path"
	"strings"
	"time"

	"atad-project/internal/gitlog"
)

type Filter struct {
	start, end time.Time
	dir, glob  string
}

func NewFilter(from, to, dir, glob string) (Filter, error) {
	var f Filter
	var err error
	if from != "" {
		f.start, err = time.Parse(time.DateOnly, from)
		if err != nil {
			return f, fmt.Errorf("from must be YYYY-MM-DD")
		}
	}
	if to != "" {
		last, err := time.Parse(time.DateOnly, to)
		if err != nil {
			return f, fmt.Errorf("to must be YYYY-MM-DD")
		}
		if last.Before(f.start) {
			return f, fmt.Errorf("to precedes from")
		}
		f.end = last.AddDate(0, 0, 1)
	}
	if strings.HasPrefix(dir, "/") {
		return f, fmt.Errorf("dir must be relative")
	}
	dir = strings.TrimSuffix(dir, "/")
	if dir == "." {
		dir = ""
	}
	if dir != "" && (!fs.ValidPath(dir) || strings.Contains(dir, "\\") || strings.Contains(dir, ":")) {
		return f, fmt.Errorf("dir must be a relative repository directory")
	}
	if _, err := path.Match(glob, ""); err != nil {
		return f, fmt.Errorf("invalid glob")
	}
	f.dir, f.glob = dir, glob
	return f, nil
}

// Apply keeps matching file changes and drops commits without a match under a path filter.
func (f Filter) Apply(c *gitlog.Commit) bool {
	if (!f.start.IsZero() && c.AuthoredAt.Before(f.start)) || (!f.end.IsZero() && !c.AuthoredAt.Before(f.end)) {
		return false
	}
	if f.dir == "" && f.glob == "" {
		return true
	}
	if c.Merge {
		return false
	}
	var files []gitlog.File
	for _, file := range c.Files {
		if f.dir != "" && file.Path != f.dir && !strings.HasPrefix(file.Path, f.dir+"/") {
			continue
		}
		if f.glob != "" {
			name := file.Path
			if !strings.Contains(f.glob, "/") {
				name = path.Base(name)
			}
			if matched, _ := path.Match(f.glob, name); !matched {
				continue
			}
		}
		files = append(files, file)
	}
	c.Files = files
	return len(files) > 0
}
