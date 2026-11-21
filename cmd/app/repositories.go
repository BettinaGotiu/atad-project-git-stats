package main

import (
	"context"
	"net/http"
	"net/url"
	"strconv"

	"atad-project/internal/gitlog"
)

func (s service) createJob(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Path   string `json:"path"`
		URL    string `json:"url"`
		Policy string `json:"merge_policy"`
	}
	if err := decode(w, r, &body); err != nil {
		problem(w, 400, "validation", err.Error())
		return
	}
	if (body.Path == "") == (body.URL == "") {
		problem(w, 400, "validation", "provide one path or url")
		return
	}
	if body.Policy == "" {
		body.Policy = "count"
	}
	if body.Policy != "count" && body.Policy != "ignore" {
		problem(w, 400, "validation", "merge_policy must be count or ignore")
		return
	}
	var read func(context.Context, func(gitlog.Commit)) error
	if body.Path != "" {
		repo, err := s.repository(body.Path)
		if err != nil {
			problem(w, 422, "unprocessable", err.Error())
			return
		}
		read = func(ctx context.Context, visit func(gitlog.Commit)) error { return gitlog.Read(ctx, repo, "", visit) }
	} else {
		if !gitlog.ValidURL(body.URL) {
			problem(w, 400, "validation", "url must name a public HTTPS GitHub repository")
			return
		}
		read = func(ctx context.Context, visit func(gitlog.Commit)) error {
			return gitlog.ReadURL(ctx, body.URL, visit)
		}
	}
	job := s.jobs.Start(body.Policy, read)
	writeJSON(w, 202, map[string]any{"id": job.ID, "state": job.State})
}

func (s service) jobReport(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id < 1 {
		problem(w, 400, "validation", "id must be a positive integer")
		return
	}
	job, ok := s.jobs.Get(id)
	if !ok {
		problem(w, 404, "not_found", "unknown repository ID")
		return
	}
	view := r.PathValue("view")
	if view != "" && view != "contributors" && view != "activity" && view != "churn" {
		problem(w, 404, "not_found", "unknown report")
		return
	}
	if view == "" {
		writeJSON(w, 200, job)
		return
	}
	switch job.State {
	case "analysing":
		problem(w, 409, "conflict", "analysis in progress")
		return
	case "failed":
		problem(w, 422, "unprocessable", job.Message)
		return
	}
	q, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		problem(w, 400, "validation", "invalid query string")
		return
	}
	opts, err := reportOptions(q, job.Policy, s.jobs.Aliases())
	if err != nil {
		problem(w, 400, "validation", err.Error())
		return
	}
	for _, c := range job.History {
		opts.add(c)
	}
	report := opts.counts.Report(opts.limit)
	out := map[string]any{"merge_policy": job.Policy, "merge_note": mergeNote}
	switch view {
	case "contributors":
		out[view] = report.Contributors
	case "activity":
		out[view] = report.Activity
	case "churn":
		out[view] = report.Churn
	}
	writeJSON(w, 200, out)
}
