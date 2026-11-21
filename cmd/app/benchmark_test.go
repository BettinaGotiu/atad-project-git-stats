package main

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"testing"

	"atad-project/internal/jobs"
)

func BenchmarkPrettier(b *testing.B) {
	repo := os.Getenv("PRETTIER_REPO")
	if repo == "" {
		b.Skip("set PRETTIER_REPO to the reference checkout")
	}
	api := service{root: repo, jobs: jobs.New(nil)}.routes()
	const query = "/stats?revision=6ba1dee64a6cacabd237746f9c62060b10747aed"
	for _, tc := range []struct{ name, filter string }{{"full", ""}, {"src_js", "&dir=src&glob=*.js"}} {
		b.Run(tc.name, func(b *testing.B) {
			for b.Loop() {
				r := httptest.NewRecorder()
				api.ServeHTTP(r, httptest.NewRequest("GET", query+tc.filter, nil))
				var report result
				if r.Code != 200 {
					b.Fatal(r.Body)
				}
				if err := json.Unmarshal(r.Body.Bytes(), &report); err != nil {
					b.Fatal(err)
				}
				if tc.filter == "" && report.Total != 12148 {
					b.Fatalf("expected 12148 commits, got %d", report.Total)
				}
				b.ReportMetric(float64(report.Duration), "service-ms")
				b.ReportMetric(float64(report.Total), "commits")
			}
		})
	}
}
