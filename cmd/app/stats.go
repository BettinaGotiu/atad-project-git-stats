package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"atad-project/internal/gitlog"
	"atad-project/internal/stats"
)

const mergeNote = "Count includes merge commits in activity. Ignore skips them. Merge diffs never add lines or file touches. Path filters select non-merge commits with matching files."

type options struct {
	format, policy string
	limit          int
	filter         stats.Filter
	aliases        stats.Aliases
	counts         *stats.Counter
}

func reportOptions(q url.Values, policy string, aliases stats.Aliases) (options, error) {
	o := options{policy: policy, aliases: aliases, format: q.Get("format"), limit: 20}
	if policy != "count" && policy != "ignore" {
		return o, errors.New("merge_policy must be count or ignore")
	}
	if o.format != "" && o.format != "json" && o.format != "text" {
		return o, errors.New("format must be json or text")
	}
	period := q.Get("period")
	if period == "" {
		period = "month"
	}
	if period != "day" && period != "week" && period != "month" {
		return o, errors.New("period must be day, week or month")
	}
	if v := q.Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			return o, errors.New("limit must be a positive integer")
		}
		o.limit = n
	}
	dir := q.Get("dir")
	if p := q.Get("path"); p != "" {
		if dir != "" {
			return o, errors.New("use either path or dir")
		}
		dir = p
	}
	filter, err := stats.NewFilter(q.Get("from"), q.Get("to"), dir, q.Get("glob"))
	o.filter = filter
	o.counts = stats.NewCounter(period)
	return o, err
}

func (o options) add(c gitlog.Commit) {
	if (o.policy == "ignore" && c.Merge) || !o.filter.Apply(&c) {
		return
	}
	o.aliases.Apply(&c)
	o.counts.Add(c)
}

type result struct {
	Repository string `json:"repository"`
	Policy     string `json:"merge_policy"`
	MergeNote  string `json:"merge_note"`
	Duration   int64  `json:"duration_ms"`
	stats.Report
}

func (s service) liveStats(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	q, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		problem(w, 400, "validation", "invalid query string")
		return
	}
	policy := q.Get("merge_policy")
	if policy == "" {
		policy = "count"
	}
	o, err := reportOptions(q, policy, s.jobs.Aliases())
	if err != nil {
		problem(w, 400, "validation", err.Error())
		return
	}
	repo, err := s.repository(q.Get("repo"))
	if err != nil {
		problem(w, 422, "unprocessable", err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()
	err = gitlog.Read(ctx, repo, q.Get("revision"), o.add)
	if err != nil {
		if errors.Is(err, gitlog.ErrRevision) {
			problem(w, 400, "validation", err.Error())
			return
		}
		problem(w, 422, "unprocessable", err.Error())
		return
	}
	report := o.counts.Report(o.limit)
	out := result{repo, policy, mergeNote, time.Since(start).Milliseconds(), report}
	if o.format == "text" {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		writeText(w, out)
		return
	}
	writeJSON(w, 200, out)
}

func writeText(w http.ResponseWriter, r result) {
	fmt.Fprintf(w, "Repository: %s\nMerge policy: %s\n%s\nCommits: %d\nDuration: %d ms\n\nContributors\n", r.Repository, r.Policy, r.MergeNote, r.Total, r.Duration)
	for _, a := range r.Contributors {
		fmt.Fprintf(w, "%s <%s>  %d commits  +%d -%d  %s to %s\n", a.Name, a.Email, a.Commits, a.Added, a.Removed, a.First.Format(time.RFC3339), a.Last.Format(time.RFC3339))
	}
	fmt.Fprintln(w, "\nActivity")
	for _, b := range r.Activity {
		fmt.Fprintf(w, "%s  %d\n", b.Period, b.Commits)
	}
	fmt.Fprintln(w, "\nFile changes")
	for _, c := range r.Churn {
		fmt.Fprintf(w, "%d  %q\n", c.Modifications, c.Path)
	}
}
