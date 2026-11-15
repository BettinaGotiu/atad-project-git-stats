package stats

import (
	"testing"
	"time"

	"atad-project/internal/gitlog"
)

func TestCountsAndMergePolicy(t *testing.T) {
	s := NewCounter("week")
	sun := time.Date(2026, 11, 15, 22, 0, 0, 0, time.UTC)
	c := gitlog.Commit{Name: "Jo", Email: "jo@example.test", AuthoredAt: sun.Add(2 * time.Hour), Files: []gitlog.File{{Path: "b", Added: 3, Removed: 1}, {Path: "a"}}}
	s.Add(c)
	c.AuthoredAt = sun
	c.Merge = true
	s.Add(c)
	r := s.Report(1)
	a := r.Contributors[0]
	if r.Total != 2 || a.Commits != 2 || a.Added != 3 || a.Removed != 1 || !a.First.Equal(sun) || !a.Last.Equal(sun.Add(2*time.Hour)) {
		t.Fatalf("contributor %+v, total %d", a, r.Total)
	}
	if len(r.Activity) != 2 || r.Activity[0] != (Bucket{"2026-11-09", 1}) || r.Activity[1] != (Bucket{"2026-11-16", 1}) || len(r.Churn) != 1 || r.Churn[0] != (Churn{"a", 1}) {
		t.Fatalf("report %+v", r)
	}
	if bucket(sun, "day") != "2026-11-15" || bucket(sun, "month") != "2026-11" {
		t.Fatal("wrong calendar bucket")
	}
}
