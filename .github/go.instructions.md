---
applyTo: "**/*.go"
---

- **Internal API defensive checks**: Do not suggest adding nil/empty guards on internal (unexported package) functions that are only called with known-safe arguments. The project trusts internal call sites and only validates at system boundaries.
- **tmux format string output parsing**: Do not flag missing length checks when parsing tmux output produced by a controlled format string (e.g., `list-panes -F "#{pane_dead} #{pane_dead_status}"`). The format string guarantees the output shape. Guards are applied selectively where tmux command semantics warrant them.
- **Functional option validation**: Do not suggest validating functional options (e.g., `WithTimeout`, `WithPollInterval`) at configuration time in `Open`. Validation happens at the point of use in wait methods, consistent with the project convention of trusting test authors over defensive input checking.
