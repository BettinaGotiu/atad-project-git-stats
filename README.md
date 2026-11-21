# Git Repository Statistics

A read-only Go service for exploring who contributed to a Git repository, when they were active, and which files changed most often.

- **Student:** Gotiu Bettina Catalina, group Master Cloud Technologies and Internet of Things (Master Year II)
- **Project:** 10 — Git Repository Statistics
- **Language:** Go 1.27+

## Run with Docker

From this project directory in PowerShell, build the image and mount the sibling Prettier clone read-only.

```powershell
docker build -t git-stats .
docker run --rm -p 8080:8080 -v "D:\prettier:/repos/prettier:ro" git-stats
```

For the release image, replace `git-stats` in the run command with `ghcr.io/bettinagotiu/atad-project-git-stats:v0.1`. The four common routes also work with a plain `docker run -p 8080:8080 git-stats`.

Open another terminal to check health and get a monthly report for JavaScript files under `src`.

```powershell
curl.exe -s http://localhost:8080/health
curl.exe -s "http://localhost:8080/stats?repo=prettier&dir=src&glob=*.js&period=month&format=text"
```

## Run directly

Install Go 1.27+ and Git. Set the local repository root before starting the service. With the root set to the checkout itself, omit `repo` from requests.

```powershell
$env:GIT_STATS_ROOT = 'D:\prettier'
go run ./cmd/app
```

From another terminal, run `curl.exe -s http://localhost:8080/stats`. Use `go run ./cmd/app --version` to print the build identity. CI stamps the release version, full commit SHA and UTC build time into the image. A local source build reports `unknown` for unavailable metadata.

## API

| Method and route | Result |
| --- | --- |
| `GET /`, `GET /health`, `GET /version` | The project identity page, process uptime, and build identity. |
| `GET /stats` | Reads the selected checkout and returns one timed report. Defaults to JSON. Use `format=text` for a human-readable report. |
| `POST /repositories` | Accepts one `path` or public GitHub HTTPS `url` and an optional `merge_policy`. Returns HTTP 202 with the new ID and `analysing`. |
| `GET /repositories/{id}` | Returns state, commit count, merge policy, and `analysed_in_ms`. Poll until `ready` or `failed`. |
| `GET /repositories/{id}/contributors` | Contributor totals, first and last activity, and the merge policy. |
| `GET /repositories/{id}/activity` | Commit counts for each nonempty UTC period. |
| `GET /repositories/{id}/churn` | File paths ordered by number of changes, highest first. |
| `PUT /identities` | Replaces runtime alias rules. Returns 204. Existing snapshots use the new rules on the next report. |
| `POST /reset` | Cancels active jobs, clears snapshots, restores startup rules and resets the next ID to 1. Returns 204. |

For example, in PowerShell with the Docker mount above:

```powershell
$base = 'http://localhost:8080'
$job = Invoke-RestMethod "$base/repositories" -Method Post -ContentType 'application/json' -Body '{"path":"prettier","merge_policy":"ignore"}'
Invoke-RestMethod "$base/repositories/$($job.id)"
# Once the state is ready:
Invoke-RestMethod "$base/repositories/$($job.id)/activity?period=week"
```

A live report reads `HEAD` by default. Pass `revision=<full commit SHA>` to reproduce a particular history. Repository paths resolve under `GIT_STATS_ROOT`, which defaults to the current directory locally and `/repos` in the image. A URL analysis clones the default branch into a temporary directory and deletes the clone after reading it.

## Reading the numbers

- **Who and when.** Commit authors receive credit. First and last activity use author timestamps. `period=day|week|month` defaults to month. Weeks begin on Monday in UTC.
- **Merges.** `count` includes merge commits once in activity. `ignore` skips them. Both policies exclude merge diffs from line totals and file touches, so integrating a branch cannot duplicate its churn.
- **Files.** Each changed path gets one touch per non-merge commit. Binary changes contribute a touch and zero measured lines. Renames count as a deletion at the old path and an addition at the new one. The default churn list has 20 entries. Set `limit` to request more.
- **Filters.** `from` and `to` are inclusive UTC dates in `YYYY-MM-DD` form. `dir` or `path` selects an exact file or directory subtree. `glob` uses Go path patterns. A pattern without a slash matches the basename at any depth. With a path filter, only commits with a matching file count, and only matching files contribute churn.
- **Aliases.** By default, email matching ignores case. Rules map specific names or addresses to a canonical name and email. Email rules take precedence over name rules. Set `GIT_STATS_ALIASES` to a JSON array file at startup, or replace the rules through HTTP:

```json
{"rules":[{"name":"Alex Smith","email":"alex@example.org","aliases":["A. Smith","alex@old.example"]}]}
```

Validation errors return 400, missing job IDs return 404, a report requested before its job finishes returns 409, and an unreadable repository returns 422. Errors have `error` and `message` fields.

## Verify and explore

```powershell
go vet ./...
go test ./...
$env:PRETTIER_REPO = 'D:\prettier'
go test ./internal/gitlog -run TestPrettierHistory -count=1
```

CI also runs `go test -race` and a container smoke test before publishing. See [architecture and source files](docs/architecture.md), [benchmark measurements](docs/benchmark.md), and the [draft course report](docs/report.docx).
