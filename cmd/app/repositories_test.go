package main

import (
	"atad-project/internal/jobs"
	"encoding/json"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestRepositoryReports(t *testing.T) {
	call := testClient(t)
	call("PUT", "/identities", `{"rules":[{"name":"Sam Smith","email":"sam@test","aliases":["sam@example.test"]}]}`, 204)
	for i, policy := range []string{"count", "ignore"} {
		response := call("POST", "/repositories", `{"path":".","merge_policy":"`+policy+`"}`, 202)
		var job jobs.Job
		if err := json.Unmarshal(response.Body.Bytes(), &job); err != nil || job.ID != i+1 || job.State != "analysing" {
			t.Fatal(response.Body)
		}
		base := "/repositories/" + strconv.Itoa(job.ID)
		deadline := time.Now().Add(15 * time.Second)
		for job.State == "analysing" && time.Now().Before(deadline) {
			rr := call("GET", base, "", 200)
			if err := json.Unmarshal(rr.Body.Bytes(), &job); err != nil {
				t.Fatal(err)
			}
			if job.State == "analysing" {
				time.Sleep(10 * time.Millisecond)
			}
		}
		if job.State != "ready" || job.Commits != 3-i {
			t.Fatalf("job %+v", job)
		}
		rr := call("GET", base+"/contributors", "", 200)
		if !strings.Contains(rr.Body.String(), "Sam Smith") || !strings.Contains(rr.Body.String(), policy) {
			t.Fatal(rr.Body)
		}
		call("GET", base+"/activity?period=day", "", 200)
		rr = call("GET", base+"/churn?path=missing", "", 200)
		if !strings.Contains(rr.Body.String(), `"churn":[]`) {
			t.Fatal(rr.Body)
		}
		rr = call("GET", base+"/churn", "", 200)
		if !strings.Contains(rr.Body.String(), "hello.txt") {
			t.Fatal(rr.Body)
		}
	}
	call("POST", "/reset", "", 204)
	call("GET", "/repositories/1", "", 404)
	rr := call("POST", "/repositories", `{"path":"."}`, 202)
	if !strings.Contains(rr.Body.String(), `"id":1`) {
		t.Fatal(rr.Body)
	}
	call("POST", "/repositories", `{"url":"file:///tmp/repo"}`, 400)
}
