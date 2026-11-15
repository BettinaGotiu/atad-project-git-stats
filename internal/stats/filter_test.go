package stats

import (
	"testing"
	"time"

	"atad-project/internal/gitlog"
)

func TestCombinedFilter(t *testing.T) {
	f, err := NewFilter("2026-11-14", "2026-11-15", "src/", "*.go")
	if err != nil {
		t.Fatal(err)
	}
	c := gitlog.Commit{AuthoredAt: time.Date(2026, 11, 15, 23, 59, 59, 0, time.UTC), Files: []gitlog.File{{Path: "src/nested/a.go"}, {Path: "src/b.md"}, {Path: "src2/a.go"}}}
	if !f.Apply(&c) || len(c.Files) != 1 || c.Files[0].Path != "src/nested/a.go" {
		t.Fatalf("files %+v", c.Files)
	}
	c.AuthoredAt = c.AuthoredAt.Add(time.Second)
	if f.Apply(&c) {
		t.Fatal("accepted next UTC day")
	}
	for _, args := range [][4]string{{"2026-02-30", "", "", ""}, {"2026-11-15", "2026-11-14", "", ""}, {"", "", "../src", ""}, {"", "", "", "["}} {
		if _, err := NewFilter(args[0], args[1], args[2], args[3]); err == nil {
			t.Errorf("accepted %q", args)
		}
	}
}
