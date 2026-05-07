# Contributing to GotEth

Thank you for your interest in contributing to GotEth — the Go indexer that pulls
validator-, epoch- and block-level data from Ethereum's consensus layer (and optionally
the execution layer + MEV relays) into ClickHouse.

This document describes how to report a bug, propose a feature, prepare a pull request,
and what we expect before code is merged. It is intentionally detailed because this
project sits in a high-bug-density area (epoch reward math, reorg handling, ClickHouse
mutation pressure) where rushed contributions have caused production incidents in the
past.

If anything below is unclear, open a GitHub Discussion or ask in the issue you are
working on rather than guessing.

---

## Table of Contents

1. [Code of Conduct](#code-of-conduct)
2. [Before You Start](#before-you-start)
3. [Development Environment](#development-environment)
4. [Reporting a Bug](#reporting-a-bug)
5. [Proposing a Feature](#proposing-a-feature)
6. [Submitting a Pull Request](#submitting-a-pull-request)
7. [Commit and Branch Conventions](#commit-and-branch-conventions)
8. [Testing Requirements](#testing-requirements)
9. [Review Process](#review-process)
10. [Security Issues](#security-issues)

---

## Code of Conduct

Be respectful. Assume good intent. Keep discussions technical and on-topic. Personal
attacks, harassment, or off-topic political content will result in the contribution
being closed.

## Before You Start

1. Read [README](./README.md). They cover the build, the three-state sliding window mental model, the rewards pipeline, and the recurring
   gotchas — most "is this a bug?" questions are answered there.
2. Search [existing issues](https://github.com/migalabs/goteth/issues) and
   [pull requests](https://github.com/migalabs/goteth/pulls) (open and closed) for
   prior discussion. Several reward-math proposals have been raised and reverted
   multiple times — re-opening one without new evidence will be closed.
3. For non-trivial work (new fork support, schema changes, new metrics, new
   subcommands), open an issue first to align on scope before writing code. A merged
   PR that does not match the agreed scope is more painful for everyone than a
   thirty-minute design discussion.

## Development Environment

```bash
# Clone with submodules — go-relay-client is a git submodule referenced by go.mod
git clone --recurse-submodules https://github.com/migalabs/goteth.git
cd goteth

# Or, if already cloned without submodules
git submodule update --init --recursive

# Build
make build           # output: ./build/goteth
make install         # install to $GOBIN

# Run unit tests
go test ./...

# Run a single test
go test ./pkg/analyzer -run TestReorgDeadlock

# Run Python ClickHouse integration tests
pip install -r tests/requirements.txt
python -m pytest tests/db_blocks_test.py
```

Toolchain: **Go 1.25+** and **ClickHouse 24+**.

CLI flags bind to `ANALYZER_*` environment variables when running the binary directly.
The Docker entrypoint reads `GOTETH_*` and forwards them — do not mix the two.

## Reporting a Bug

Open a [bug report](https://github.com/migalabs/goteth/issues/new?template=bug_report.yml)
and fill every section. A bug report missing reproduction steps, version, or expected
vs. actual data is not actionable and will be asked for the missing information before
any investigation starts.

A high-quality bug report contains:

- **GotEth version / commit SHA**: the output of `goteth --version`, or the SHA the
  binary was built from. Bug reports against binaries built before the
  Electra consolidation rework (pre-2026-02-20) will be asked to retest on `master`.
- **Environment**: ClickHouse version, beacon node implementation and version,
  whether the execution layer is wired, network (mainnet, holesky, sepolia, gnosis,
  hoodi, fusaka-devnet, …), Docker vs. bare-metal.
- **Configuration**: the relevant CLI flags and env vars (redact secrets).
  Specifically include `--metrics`, `--init-slot`, `--final-slot`, `--workers-num`,
  `--rewards-aggregation-epochs`, and any `--db-*` values.
- **Reproduction steps**: minimal steps an outsider can run end-to-end. If the bug
  needs a specific epoch range, include it. If it needs a specific validator index
  or block, include it.
- **Expected behavior**: what you thought should happen, and the spec or doc passage
  that backs that up where applicable (a link to the consensus-specs file beats a
  paraphrase).
- **Actual behavior**: logs (with timestamps), database query results, screenshots
  of dashboards, Prometheus metric snapshots — whatever shows the discrepancy.
- **Scope**: how many rows, validators, epochs, or networks are affected. "All
  validators on mainnet for epochs 421000–425000" is more useful than "rewards
  look wrong".
- **Mitigation already applied or considered**: e.g. "we re-ran val-window with
  `--num-epochs=512`", "we re-indexed from the Electra fork", "we worked around
  this by querying the beacon node directly". This helps the reviewer skip
  suggestions you already ruled out.

## Proposing a Feature

Open a [feature request](https://github.com/migalabs/goteth/issues/new?template=feature_request.yml).
Include:

- **Use case**: what problem this solves and for whom (downstream dashboard, research,
  internal tooling). Features without a concrete consumer rarely justify the
  maintenance cost.
- **Proposed change**: a sketch of the API, schema additions, or CLI flag — enough
  for a maintainer to flag obvious issues before you start coding.
- **Alternatives considered**: equivalent SQL queries, beacon API calls, or
  off-the-shelf tools. If the data is already derivable, the answer may be a
  documentation patch rather than a feature.
- **Impact on disk and ingest cost**: new ClickHouse tables (especially
  per-validator-per-epoch ones) cost real money. Estimate row count per epoch and
  per month at full mainnet validator count.

## Submitting a Pull Request

1. **Fork** the repository and create a topic branch off `master` (or `dev` if you
   are coordinating with the maintainers).
2. **Keep the change focused**. One logical change per PR. Drive-by refactors and
   reformatting in the same PR as a bug fix make the diff hard to review and
   complicate any rollback. Open a separate PR.
3. **Match the existing style**. Run `gofmt`, keep package layout consistent, do
   not introduce new third-party dependencies without justification.
4. **Update documentation**. If you change a CLI flag, schema, or behavior, update
   `README.md` and `docs/tables.md` in the same PR.
5. **Add or update tests** (see [Testing Requirements](#testing-requirements)).
6. **Run the full suite locally** before pushing: `go test ./...` plus the relevant
   Python integration tests under `tests/`.
7. **Fill out the PR template completely**. The Motivation, Description, Tasks,
   and Proof of Success sections are required — empty sections will block review.
8. **Reference the issue** the PR addresses with `Fixes #NNN` or `Closes #NNN` so
   it auto-closes when merged.

### What "Proof of Success" means

This is the part most contributors under-deliver on. We expect at least one of:

- **Unit / integration test output** showing the new test passes and existing tests
  still pass.
- **Before/after database query results** when the change affects persisted data
  (e.g. a `SELECT … WHERE f_epoch IN (…)` showing the corrected `f_reward`).
- **Logs from a real run** for changes that only manifest at runtime (reorg handling,
  SSE listeners, retry logic). Include enough timestamps that a reviewer can match
  the log lines to your description.
- **Profiling or benchmark output** when the change is performance-driven.

Synthetic mock-only proof is acceptable for pure refactors; for any behavior change
in the indexer pipeline we expect at least one run against a live beacon endpoint.

### What is in scope for a single PR

| Change | One PR | Multiple PRs |
|---|---|---|
| Bug fix + the test that reproduces it | ✅ | |
| New persisted table + the analyzer wiring + migration | ✅ | |
| Refactor + bug fix in the same area | | ✅ split |
| Rewards math change + unrelated metrics flag | | ✅ split |
| New fork support spans many files | ✅ if cohesive | |

When in doubt, smaller is better. A 200-line PR that lands in two days beats a
2000-line PR that sits open for a month.

## Commit and Branch Conventions

- **Branch name**: `<type>/<short-description>`, e.g. `fix/reorg-deadlock`,
  `feat/fusaka-blob-support`, `docs/contributing-guide`. Allowed types: `feat`,
  `fix`, `chore`, `docs`, `refactor`, `test`, `perf`.
- **Commit message**: short imperative subject (≤ 72 chars), then a blank line, then
  a body that explains *why*. Match the type prefix used on the branch:
  ```
  fix(analyzer): serialize AdvanceFinalized to prevent concurrent CleanUpTo race

  CleanUpTo was being invoked from two finalized-checkpoint handlers in
  parallel after a reorg, which led to a partial cache wipe and rewards
  rows being persisted with stale state references. Serializing through
  the existing processerBook removes the race without adding a new lock.
  ```
- **Squash strategy**: small PRs merge as a single commit; large PRs may keep a
  reviewable commit-by-commit history if each commit is independently buildable
  and tested. The maintainers will pick at merge time.
- **Do not** add `Co-Authored-By` lines pointing at AI assistants. If you used
  one, the work is yours; sign accordingly.
- **Do not** include hostnames, IPs, or names of internal infrastructure (e.g.
  `eth-archive`, `attestant-N`) in commits, code, or logs.

## Testing Requirements

| Change type | Minimum required |
|---|---|
| Reward math (any file under `pkg/spec/metrics/`) | Unit test that drives `EpochReward` with a known synthetic state and asserts the expected gwei value. **Reward changes will not be merged without a test.** |
| Reorg / lock model (`pkg/analyzer/`) | Unit test reproducing the failure mode if it is a fix. The existing `pkg/analyzer/reorg_deadlock_test.go` is the canonical example. |
| New persisted table | A round-trip integration test under `tests/` that writes a row and reads it back. Update `docs/tables.md`. |
| New CLI flag | The flag must round-trip through `pkg/config` defaults and be listed in `cmd/<command>.go`. Manual run output in the PR is sufficient — no unit test required. |
| Documentation only | None, but spell-check and verify links resolve. |

Run before pushing:

```bash
gofmt -l .                    # must be empty
go vet ./...
go test ./...
```

If your change touches code under `pkg/db`, also run the relevant Python tests
in `tests/` against a local ClickHouse — a passing Go test does not guarantee a
correct migration or query.

## Review Process

- A maintainer (currently @tdahar with @cortze on rewards-related changes) will
  review within a few business days. Please be patient.
- Address review comments by pushing additional commits. Do not force-push during
  review — it makes incremental review impossible. We will squash on merge.
- If review stalls, ping the PR with a short summary of what is blocking and
  tag the maintainer.
- Once approved and CI is green, a maintainer will merge. External contributors
  do not have merge access by design.

## Security Issues

**Do not file security issues publicly.** Email the maintainers directly (see
`AUTHORS.md` for handles; you can reach `@cortze` and `@tdahar` via GitHub
profiles or MigaLabs channels) with a description of the issue, reproduction
steps, and your proposed mitigation. We will acknowledge within five business
days and coordinate disclosure.

If the issue is in a dependency, also report it upstream. If it is in
`go-relay-client` (the submodule), follow that repository's policy.

---

Thank you for contributing. The data this tool produces is used by researchers,
infrastructure operators, and dashboards — careful contributions matter.
