package stats

import (
	"cmp"
	"slices"
	"strings"
	"time"

	"atad-project/internal/gitlog"
)

type Contributor struct {
	Name    string    `json:"name"`
	Email   string    `json:"email"`
	Commits int       `json:"commits"`
	Added   int64     `json:"lines_added"`
	Removed int64     `json:"lines_removed"`
	First   time.Time `json:"first_activity"`
	Last    time.Time `json:"last_activity"`
}

type Bucket struct {
	Period  string `json:"period"`
	Commits int    `json:"commits"`
}

type Churn struct {
	Path          string `json:"path"`
	Modifications int    `json:"modifications"`
}

type Report struct {
	Total        int           `json:"total_commits"`
	Contributors []Contributor `json:"contributors"`
	Activity     []Bucket      `json:"activity"`
	Churn        []Churn       `json:"code_churn"`
}

type Counter struct {
	period   string
	total    int
	authors  map[string]*Contributor
	activity map[string]int
	paths    map[string]int
}

func NewCounter(period string) *Counter {
	return &Counter{period: period, authors: make(map[string]*Contributor),
		activity: make(map[string]int), paths: make(map[string]int)}
}

// Add counts a merge as one event and skips its diff.
func (s *Counter) Add(c gitlog.Commit) {
	key := strings.ToLower(c.Email)
	if key == "" {
		key = "name:" + c.Name
	}
	t := c.AuthoredAt.UTC()
	author := s.authors[key]
	if author == nil {
		author = &Contributor{Name: c.Name, Email: c.Email, First: t, Last: t}
		s.authors[key] = author
	}
	if t.Before(author.First) {
		author.First = t
	}
	if t.After(author.Last) {
		author.Last = t
	}
	author.Commits++
	s.total++
	s.activity[bucket(t, s.period)]++
	if c.Merge {
		return
	}
	for _, file := range c.Files {
		author.Added += file.Added
		author.Removed += file.Removed
		s.paths[file.Path]++
	}
}

func bucket(t time.Time, period string) string {
	switch period {
	case "month":
		return t.Format("2006-01")
	case "week":
		daysSinceMonday := (int(t.Weekday()) + 6) % 7
		t = t.AddDate(0, 0, -daysSinceMonday)
	}
	return t.Format(time.DateOnly)
}

// Report sorts ties consistently and returns up to limit file paths.
func (s *Counter) Report(limit int) Report {
	r := Report{Total: s.total, Contributors: []Contributor{}, Activity: []Bucket{}, Churn: []Churn{}}
	for _, a := range s.authors {
		r.Contributors = append(r.Contributors, *a)
	}
	for period, n := range s.activity {
		r.Activity = append(r.Activity, Bucket{period, n})
	}
	for path, n := range s.paths {
		r.Churn = append(r.Churn, Churn{path, n})
	}
	slices.SortFunc(r.Contributors, func(a, b Contributor) int {
		return cmp.Or(cmp.Compare(b.Commits, a.Commits), cmp.Compare(a.Email, b.Email), cmp.Compare(a.Name, b.Name))
	})
	slices.SortFunc(r.Activity, func(a, b Bucket) int { return cmp.Compare(a.Period, b.Period) })
	slices.SortFunc(r.Churn, func(a, b Churn) int {
		return cmp.Or(cmp.Compare(b.Modifications, a.Modifications), cmp.Compare(a.Path, b.Path))
	})
	r.Churn = r.Churn[:min(limit, len(r.Churn))]
	return r
}
