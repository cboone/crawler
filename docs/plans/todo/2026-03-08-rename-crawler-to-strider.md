# Rename crawler to strider

## Context

The library name "crawler" sounds too much like "web crawler," which carries
negative connotations. After brainstorming names evocative of Dungeon Crawler
Carl, the decision is to rename to **strider** (library) and **stride** (paired
CLI tool in `cboone/crawl`). The names evoke confident movement through
dangerous territory, with a Tolkien resonance that complements the DCC flavor.

This plan covers renaming everything in this repo and creating a GitHub issue in
`cboone/crawl` for the corresponding rename.

## Steps

### 1. Rename the GitHub repo

```sh
gh repo rename strider
```

This changes `cboone/crawler` to `cboone/strider`. GitHub automatically
redirects the old URL.

### 2. Rename Go source files

- `crawler.go` -> `strider.go`
- `crawler_test.go` -> `strider_test.go`

Use `git mv` for both to preserve history.

### 3. Update go.mod

- `module github.com/cboone/crawler` -> `module github.com/cboone/strider`

### 4. Update package declarations (10 files)

Change `package crawler` to `package strider` in all public `.go` files:

- `strider.go` (formerly crawler.go)
- `doc.go`
- `keys.go`
- `match.go`
- `options.go`
- `screen.go`
- `snapshot.go`
- `tmux.go`

Change `package crawler_test` to `package strider_test` in:

- `strider_test.go` (formerly crawler_test.go)
- `example_test.go`

### 5. Update import paths

Replace `"github.com/cboone/crawler"` with `"github.com/cboone/strider"` and
`"github.com/cboone/crawler/internal/tmuxcli"` with
`"github.com/cboone/strider/internal/tmuxcli"` in:

- `strider.go`
- `tmux.go`
- `strider_test.go`
- `example_test.go`
- `internal/tmuxcli/tmuxcli_test.go`

### 6. Update error message prefixes

Replace `"crawler:` with `"strider:` in all error format strings. These appear
in approximately 30+ locations across:

- `strider.go` (was `crawler.go`)
- `snapshot.go`
- `tmux.go`

### 7. Rename environment variables

| Old                            | New                            | Files                                  |
| ------------------------------ | ------------------------------ | -------------------------------------- |
| `CRAWLER_UPDATE`               | `STRIDER_UPDATE`               | `snapshot.go`, `strider_test.go`       |
| `CRAWLER_TMUX`                 | `STRIDER_TMUX`                 | `tmux.go`, `options.go`               |
| `CRAWLER_WAITFOR_TIMEOUT_HELPER` | `STRIDER_WAITFOR_TIMEOUT_HELPER` | `strider_test.go`                    |
| `CRAWLER_WAITEXIT_TIMEOUT_HELPER` | `STRIDER_WAITEXIT_TIMEOUT_HELPER` | `strider_test.go`                  |
| `CRAWLER_TEST_VAR`             | `STRIDER_TEST_VAR`             | `strider_test.go`                      |

Also update all comments and documentation referencing these env vars.

### 8. Update socket path prefix

In `tmux.go`, change the socket name format from `"crawler-%s-%s.sock"` to
`"strider-%s-%s.sock"` (2 locations).

### 9. Update temp directory prefix

In `strider_test.go`, change `"crawler-testbin-*"` to `"strider-testbin-*"`.

### 10. Update godoc and comments

- `doc.go`: Update package doc comment (lines 1-3, 11-12, 25, 61, 69, 86)
- `internal/testbin/main.go`: Update comment on line 1
- All other comments referencing "crawler" in code files

### 11. Update API references in tests

In `strider_test.go` and `example_test.go`, all `crawler.Open`, `crawler.Text`,
`crawler.Regexp`, etc. calls become `strider.Open`, `strider.Text`,
`strider.Regexp`, etc. (~150+ occurrences).

### 12. Update documentation files

All markdown files need `crawler` -> `strider` and `CRAWLER_` -> `STRIDER_`
replacements:

- `README.md`
- `CLAUDE.md`
- `docs/ARCHITECTURE.md`
- `docs/GETTING-STARTED.md`
- `docs/MATCHERS.md`
- `docs/PATTERNS.md`
- `docs/SNAPSHOTS.md`
- `docs/TROUBLESHOOTING.md`
- `docs/plans/done/PLAN.md`
- `docs/plans/todo/add-comprehensive-docs.md`

Also update `pkg.go.dev` links from
`pkg.go.dev/github.com/cboone/crawler` to
`pkg.go.dev/github.com/cboone/strider`.

### 13. Update git remote URL

```sh
git remote set-url origin https://github.com/cboone/strider.git
```

### 14. Create GitHub issue in cboone/crawl

Create an issue titled "Rename crawl to stride" explaining:

- The library has been renamed from `crawler` to `strider`
- This CLI tool should be renamed from `crawl` to `stride` to match
- The module path will change from `github.com/cboone/crawl` to
  `github.com/cboone/stride`
- The dependency import path changes from `github.com/cboone/crawler` to
  `github.com/cboone/strider`

### 15. Update auto-memory

Update the memory directory path reference if needed. The directory path
`/Users/ctm/.claude/projects/-Users-ctm-Development-crawler/` is derived from
the filesystem path, so it will change when the local directory is renamed. No
action needed in this plan (that happens outside this repo).

## Verification

1. `go build ./...` compiles without errors
2. `go vet ./...` passes
3. `go test ./...` passes (requires tmux in PATH)
4. `grep -r "crawler" --include="*.go" .` returns zero matches
5. `grep -r "CRAWLER" --include="*.go" .` returns zero matches
6. `grep -r "crawler" --include="*.md" .` returns zero matches (or only
   historical references in plan files, which are acceptable)
7. Verify the GitHub issue was created in `cboone/crawl`

## Commit strategy

Use frequent small commits at logical boundaries:

1. `git mv` the source files
2. Update `go.mod` and package declarations
3. Update import paths
4. Update error message prefixes
5. Rename environment variables
6. Update socket/temp prefixes
7. Update tests (API references)
8. Update documentation
9. Rename GitHub repo and update remote URL
