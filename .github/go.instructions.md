---
applyTo: "**/*.go"
---

- **Internal API defensive checks**: Do not suggest adding nil/empty guards on internal (unexported package) functions that are only called with known-safe arguments. The project trusts internal call sites and only validates at system boundaries.
- **tmux format string output parsing**: Do not flag missing length checks when parsing tmux output produced by a controlled format string (e.g., `list-panes -F "#{pane_dead} #{pane_dead_status}"`). The format string guarantees the output shape. Guards are applied selectively where tmux command semantics warrant them.
- **Functional option validation**: Do not suggest validating functional options (e.g., `WithTimeout`, `WithPollInterval`) at configuration time in `Open`. Validation happens at the point of use in wait methods, consistent with the project convention of trusting test authors over defensive input checking. Terminal-level defaults are intentionally not clamped or validated; only per-call wait overrides (`WithinTimeout`, `WithWaitPollInterval`) are validated/clamped.
- **Best-effort capture helpers**: Do not suggest surfacing or wrapping errors from `captureScreenRaw`. This function is intentionally best-effort, returning nil on failure so callers can handle the nil case explicitly. The nil return is a deliberate API contract, not a lost error.
- **fmt.Stringer on pointer receivers**: Do not flag `%s` formatting with pointer types that implement `String() string` via a pointer receiver. Go's `fmt` package correctly invokes the `Stringer` interface on pointer receivers (e.g., `*Screen` with `func (s *Screen) String() string`).
