# Prettier benchmark

The reference checkout was `D:\prettier` at `6ba1dee64a6cacabd237746f9c62060b10747aed`. Git counted 12,148 commits reachable from that revision. The native benchmark used Windows amd64, Go 1.27.4, Git 2.42.0.windows.2 and an Intel Core i7-1255U.

| Scan | Commits in report | Three measurements (s) | Median (s) |
| --- | ---: | --- | ---: |
| Full repository, count merges | 12,148 | 25.069, 26.601, 37.027 | 26.601 |
| Under `src`, matching `*.js` | 4,343 | 30.256, 14.970, 20.286 | 20.286 |
| Git metadata only | — | 0.187, 0.169, 0.170 | 0.170 |
| Git metadata and `--numstat` | — | 23.624, 31.451, 41.055 | 31.451 |

The first two rows measure `duration_ms` from the live HTTP handler, through Git traversal, aggregation and sorting. The other rows time a Git process with its output discarded. The checkout and OS cache were reused, and the machine also ran build and test work. These are wall-clock observations, not additive timing components. The Git-only comparison shows that computing file diffs took substantially longer than reading author and timestamp metadata. Filtered requests still read the diffs before selecting paths.

A separate Linux container scan on Docker Desktop completed the full report in **237.337 seconds** using the read-only Windows bind mount and UID 10001. It was a single observation with a different filesystem path and execution environment.

## Reproduce

Run the benchmark with the full reference repository available locally.

```powershell
$env:PRETTIER_REPO = 'D:\prettier'
go test ./cmd/app -run '^$' -bench BenchmarkPrettier -benchtime=1x -count=3 -timeout 20m
```

The benchmark pins the revision, issues each request through the actual HTTP handler, checks the status and verifies the full commit count. `service-ms` is the server measurement. `ns/op` additionally includes request construction, encoding and reading the result.

The Git-only runs used `git -C D:/prettier log --topo-order --format=%x00%an%x00%ae%x00%at%x00%P <revision> --`, then the same command with `--numstat -z --no-renames --diff-merges=off --root` before the revision. Standard output was directed to the null device for both.
