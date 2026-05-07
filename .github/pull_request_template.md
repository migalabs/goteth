<!--
Thanks for opening a pull request. Please fill every section below.
PRs with empty Motivation, Description, Tasks, or Proof of Success sections will be
asked for changes before review starts. See CONTRIBUTING.md for the full guidelines.
-->

## Motivation
<!--
Why this change. Link the issue with `Fixes #NNN` or `Closes #NNN` so it auto-closes.
For bug fixes, summarize the failure mode in one sentence and the impact (rows, validators,
epochs, networks affected).
For features, link the design discussion or feature-request issue.
-->

_Related links:_

## Description
<!--
How the change works. Major structural changes, new packages or tables, new flags.
Call out any non-obvious decisions and the alternatives you rejected.
-->

## Type of change
<!-- Check all that apply -->
- [ ] Bug fix (non-breaking)
- [ ] New feature (non-breaking)
- [ ] Breaking change (CLI flag rename, schema change, behavior change)
- [ ] Documentation only
- [ ] Refactor / internal cleanup
- [ ] Performance improvement

## Tasks
<!-- Checklist of in-PR tasks. Tick as you complete them. -->
- [ ]

## Testing
<!--
Show what you ran. At minimum:
- `go test ./...` output (or the specific package(s) touched)
- Any new test you added and what it covers
- For DB changes: the Python integration test under `tests/` you ran

Reward-math changes will not be merged without a unit test that drives `EpochReward`
with a known synthetic state. See CONTRIBUTING.md → Testing Requirements.
-->

```text

```

## Reproduction steps (for bug fixes)
<!-- Minimal steps to reproduce the bug on `master` before this PR. Delete if N/A. -->

## Mitigation options considered
<!--
Other approaches you weighed and why this one won. Especially important when the change
affects:
  - Concurrency / locking around `processerBook` or `ChainCache`
  - Reward math in `pkg/spec/metrics/`
  - ClickHouse table schema or mutation patterns
  - SSE event handling / reorg processing
-->

## Proof of Success
<!--
Evidence that the change works. At least one of:
  - Test output (preferred for any logic change)
  - Before/after database queries (when persisted data is affected)
  - Logs from a real run with timestamps (for runtime-only changes)
  - Profiling / benchmark output (for performance work)

Synthetic mock-only proof is acceptable for pure refactors. For any behavior change in
the indexer pipeline, include at least one run against a live beacon endpoint.
-->

## Documentation
- [ ] `README.md` updated (if user-facing flag, install, or run change)
- [ ] `docs/tables.md` updated (if persisted schema change)
- [ ] Inline comments added where the *why* is non-obvious

## Backwards compatibility
<!--
Does this change break existing deployments, dashboards, or downstream consumers?
If yes, describe the migration path. If a DB migration is added, link the migration file.
-->

## Reviewer notes
<!-- Anything specific you want the reviewer to look at first, or known follow-ups. -->
