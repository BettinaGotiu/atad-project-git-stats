# Architecture

The service listens on port 8080. Go handles HTTP, validation and aggregation. The installed Git executable reads repository history.

```text
GET /stats              → local Git history → filters → aliases → counts → JSON or text
POST /repositories      → background reader → in-memory snapshot under an integer ID
GET /repositories/ID/…  → snapshot          → filters → aliases → counts → JSON
```

## Data and lifetime

A live request resolves the revision once. The Git reader emits a commit with its author, UTC timestamp, merge flag and file changes. The counter keeps one entry per contributor, active time period and changed path. A live report releases those maps when it finishes. Background jobs retain parsed commits so repeated filtered reports can reuse the same history. A mutex protects job IDs, states and the current alias map. Completed histories are immutable. Reset cancels workers, clears jobs, restores the startup aliases and restarts ID allocation at 1. Restarting the process also clears runtime state.

## Three decisions

1. **Stream one Git log per scan.** Git handles object storage, packs and history traversal. Launching a diff process for each commit would add over 12,000 process launches to the reference scan. NUL-delimited fields preserve unusual filenames.
2. **Separate activity from line attribution.** Users can count or ignore merge events. In either mode, branch authors retain their own changes. Diffing a merge against its first parent would credit the integrator with the incoming branch's churn.
3. **Keep named snapshots in memory.** The course routes need stable IDs and repeatable queries. Keeping parsed records allows a later filter or identity rule to take effect without a second Git scan. Memory use grows with retained histories until reset. The live endpoint suits one-off queries.

## Source map

| File | Responsibility |
| --- | --- |
| `cmd/app/main.go` | Reads startup configuration, handles `--version`, starts and shuts down the server. |
| `cmd/app/contract.go` | Serves the identity, health, version and reset routes and encodes JSON. |
| `cmd/app/service.go` | Registers routes, resolves allowed checkout paths and writes API errors. |
| `cmd/app/stats.go` | Validates report options, runs live analysis and formats plain text. |
| `cmd/app/repositories.go` | Starts jobs and serves status, contributor, activity and churn responses. |
| `cmd/app/identities.go` | Decodes JSON request bodies and applies identity-rule updates. |
| `internal/gitlog/gitlog.go` | Resolves revisions, runs read-only Git commands and decodes the NUL-delimited log. |
| `internal/gitlog/remote.go` | Reads public GitHub repositories through a temporary bare clone. |
| `internal/stats/stats.go` | Counts commits, lines, activity and file touches, then sorts results. |
| `internal/stats/aliases.go` | Loads and validates rules and resolves contributor identities. |
| `internal/stats/filter.go` | Selects commits by author date and files by path or glob. |
| `internal/jobs/store.go` | Assigns IDs, publishes job snapshots and implements cancellation and reset. |
| `internal/version/version.go` | Supplies build identity to both HTTP and CLI clients. |
| `cmd/app/main_test.go` | Checks the common routes and version contract. |
| `cmd/app/stats_test.go` | Builds a small merge history and checks both live output formats. |
| `cmd/app/repositories_test.go` | Exercises both merge policies, identity updates, polling and reset. |
| `cmd/app/benchmark_test.go` | Measures live reports on the pinned Prettier revision. |
| `internal/gitlog/gitlog_test.go` | Checks the log parser, fixture history and optional Prettier scan. |
| `internal/gitlog/remote_test.go` | Checks allowed repository URLs and bare checkouts. |
| `internal/stats/stats_test.go` | Checks merge attribution, calendar buckets and churn ties. |
| `internal/stats/aliases_test.go` | Groups contributors by canonical address, alternate address and name. |
| `internal/stats/filter_test.go` | Checks inclusive date bounds, combined file filters and input errors. |
| `internal/jobs/store_test.go` | Checks worker cancellation and snapshot removal on reset. |

The submission metadata and personal reflection in the course report are pending student input.
