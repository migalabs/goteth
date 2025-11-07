# GotEth Developer Onboarding Guide

This document explains how the repository is structured, how the code executes end-to-end, and what to watch out for when extending the system. It is meant for engineers who already feel comfortable with Go and Ethereum’s consensus/execution APIs.

---

## 1. Mission & Architecture Snapshot

- **Goal**: index validator-, epoch-, and block-level insights from Ethereum’s consensus and execution layers into ClickHouse so other products (Pandametrics, Ethseer, etc.) can query them.
- **Core runtime**: a CLI binary (`goteth`) exposing the `blocks` pipeline (main indexer) and `val-window` maintenance routine.
- **High-level flow**:
  1. Parse CLI flags/env vars into `pkg/config`.
  2. `pkg/analyzer.ChainAnalyzer` orchestrates downloads, processing, and persistence.
  3. `pkg/clientapi` talks to beacon/execution nodes; `pkg/relay` enriches blocks with MEV builder data; `pkg/events` streams head/finality events.
  4. `pkg/spec` converts fork-specific data into “agnostic” structs used everywhere else.
  5. `pkg/db` batches/persists records into ClickHouse and emits Prometheus metrics.

Keep [docs/tables.md](./tables.md) handy for the ClickHouse schema the code is filling.

---

## 2. Runtime Entry Points

### CLI commands (`cmd/*.go`)

| Command | Code | Purpose |
| --- | --- | --- |
| `goteth blocks` | `cmd/blocks_cmd.go` | Full indexer; orchestrates block/state download and metrics persistence. |
| `goteth val-window` | `cmd/validator_window_cmd.go` | Prunes old validator reward rows using finalized checkpoints. |

Both commands accept flags via urfave/cli; every flag has an `ANALYZER_*` env alias (see `pkg/config/defaults.go` for defaults). `main.go` wires commands into the CLI app and configures logrus output.

### Configuration layer (`pkg/config`)

- `AnalyzerConfig` and `ValidatorWindowConfig` hold runtime knobs across endpoints, slot ranges, metrics toggles, worker counts, etc.
- `Apply(*cli.Context)` overrides defaults with CLI/env values, so new flags only require a default + setter here.
- Reuse these structs whenever you add features that need to be passed across packages; the analyzer receives a fully populated config and never reads env vars directly.

---

## 3. Core Analyzer Pipeline (`pkg/analyzer`)

`ChainAnalyzer` (see `pkg/analyzer/chain_analyzer.go`) is the heart of GotEth. It wires together every dependency during `NewChainAnalyzer` and then runs one of two modes:

1. **Historical/backfill**: enumerate slots between `initSlot` and `finalSlot` and download/process sequentially.
2. **Finalized/head**: backfill from the latest database slot until the head, then subscribe to head/finality events and keep up with the chain.

### Initialization

During construction the analyzer:

1. Sets up a cancellable context, Prometheus exporter, and `utils.RoutineBook` instances to monitor goroutines.
2. Creates a ClickHouse client (`pkg/db.New`) and connects before doing any work.
3. Instantiates the HTTP API client (`pkg/clientapi.NewAPIClient`) with optional execution endpoint, reward API toggles, and Prometheus hooks.
4. Derives the beacon deposit contract address either from `mainnet/holesky/...` shortcuts or a raw hex address.
5. Requests genesis time to pick the right relay list (`pkg/relay.InitRelaysMonitorer`) and seeds the DB with genesis metadata.
6. Parses the enabled metric set (`db.NewMetrics`) so downstream steps can short-circuit work when the caller is only interested in a subset (e.g., skip transactions).

### Download & Processing Loops

- `runHistorical` and `runHead` generate download tasks per slot and push them onto `downloadTaskChan`.
- `runDownloadBlocks` fans out work:
  - `DownloadBlockControlled` waits for states two epochs behind to exist before downloading the next block, avoiding race conditions between state- and block-derived metrics.
  - `ProcessBlock` persists the block, withdrawals, BLS changes, deposits, transactions, receipts, and blob sidecars (Deneb+) via various `db.Persist*` helpers.
  - When the slot completes an epoch, it triggers `DownloadState` and `ProcessStateTransitionMetrics` to operate on the triplet (prev/current/next states). That step calculates epoch summaries, proposer duties, validator rewards, slashings, deposits/withdrawals requests (Electra), pool summaries, etc.
- `ChainCache` holds `AgnosticBlock` and `AgnosticState` objects keyed by slot/epoch; `Wait` blocks until data is available, acting as a synchronization primitive between downloaders and processors.
- `validatorsRewardsAggregations` buffers per-validator totals when `--rewards-aggregation-epochs > 1`, flushing to ClickHouse once the configured window completes.

### Event-driven behavior

- `pkg/events` subscribes to head, finalized checkpoint, reorg, and blob sidecar streams and feeds them back into the analyzer:
  - `Head` events insert into ClickHouse (`db.PersistHeadEvents`) and unlock new download slots.
  - `FinalizedCheckpoint` events advance the cleanup window (`AdvanceFinalized`) so cached blocks/states do not grow without bounds.
  - `Reorg` events persist diagnostics and trigger a reprocessing path.
  - `BlobSidecar` events mirror availability data into ClickHouse for monitoring.

### Relay + MEV data (`pkg/relay`)

- `RelaysMonitor` instantiates HTTPS clients for every known relay on the current network (based on genesis timestamp).
- For each epoch, `processBlockRewards` resolves inline consensus rewards, transaction fees, and delivered MEV bids (`GetDeliveredBidsPerSlotRange`) to flush a `db.BlockReward` row per slot.

### Prometheus instrumentation

- Analyzer, DB service, API client, and `RoutineBook`s register metrics modules inside `pkg/metrics`. Every module defines `IndvMetrics` with an init/update hook and the server exposes them via `/metrics` on the configured port.

---

## 4. Supporting Services & Packages

| Package | Responsibility | Notes |
| --- | --- | --- |
| `pkg/clientapi` | Thin wrapper around beacon (`go-eth2-client/http`) and execution (`go-ethereum/ethclient`) RPC clients. Manages retry logic, rate-limiting (via `RoutineBook`), blob sidecars, state roots, rewards API, receipts, etc. |
| `pkg/spec` | Converts fork-specific structs into “agnostic” `AgnosticBlock`, `AgnosticState`, `AgnosticTransaction`, etc., so upstream code does not branch on hard forks. Contains reward math, validator status helpers, and Electra-specific trackers. |
| `pkg/spec/metrics` | Houses `StateMetrics` implementations per fork (`state_phase0.go`, `state_altair.go`, `state_deneb.go`, `state_electra.go`) that compute attestation/validator rewards, epoch KPIs, and derived values. |
| `pkg/db` | ClickHouse integration. `service.go` manages low-level (`ch-go`) and high-level (`clickhouse-go/v2`) connections, batching via `PersistableObject`s, and Prometheus monitors. Each `*.go` file (e.g., `block_metrics.go`, `validator_rewards.go`) describes inserts/deletes for a table. Migrations live under `pkg/db/migrations`. |
| `pkg/utils` | Shared helpers: logging defaults, byte/SSZ compression (`snappy.go`), `RoutineBook` for concurrency backpressure, validator index parsing, and various time helpers. |
| `pkg/validator_window` | Listens to finalized checkpoints and removes `t_validator_rewards_summary` rows older than `--num-epochs`, keeping disk usage in check. |
| `pkg/metrics` | Minimal Prometheus module builder used by analyzer, DB, and API clients for introspection. |
| `go-relay-client` | Git submodule fork pinned via `replace` in `go.mod`; provides MEV relay APIs (`DeliveredBulkBidTrace`). |

---

## 5. Data Model & Persistence

- **Tables**: Documented in `docs/tables.md` with column descriptions and engines. Keep this file updated when schema changes ship.
- **Migrations**: Stored under `pkg/db/migrations`. Use the scripts in `db_migration/` with `golang-migrate` when you need to force versions or apply manual fixes (see README’s “Database migrations” section).
- **Batching**: Each `db.Persist…` method fills a `PersistableObject` and flushes via the low-level client. If you add a new persistence path, follow the existing pattern so Prometheus metrics stay accurate.
- **Deletions**: Use `db.DeletableObject` via the high-level client for maintenance tasks (validator window, pruning orphaned rows, etc.).

---

## 6. Local Development Workflow

1. **Toolchain**: Go 1.23+ (toolchain pinned to 1.24) and ClickHouse. Execution node access is optional unless you persist transactions or blob sidecars.
2. **Build**: `make build` produces `./build/goteth`; `make install` drops it into your `$GOBIN`.
3. **Config**: Copy `.env` from Docker or set env vars manually; flags have env aliases (e.g., `ANALYZER_BN_ENDPOINT`).
4. **Run**:
   ```bash
   ./build/goteth blocks --bn-endpoint http://localhost:5052 --db-url clickhouse://...
   ./build/goteth val-window --num-epochs 256
   ```
5. **Testing**:
   - Go unit tests: `go test ./...` (e.g., `pkg/analyzer/metrics_test.go` validates reward math).
   - ClickHouse integration tests (Python): located in `tests/*.py`, relying on `tests/requirements.txt`.
6. **Containers**: `docker-compose.yml` spins up GotEth alongside ClickHouse using the `.env` file; useful for smoke tests.

---

## 7. Extending GotEth

### Adding a new metric/table

1. Define the ClickHouse schema + migration under `pkg/db/migrations`.
2. Extend `docs/tables.md` so ops teams know what changed.
3. Create an insert helper in `pkg/db/<table>.go` and register Prometheus monitoring if needed.
4. Populate the data inside `pkg/analyzer` (block or state pipeline) using the `pkg/spec` helpers.
5. Update `pkg/db/metrics.go` if the metric needs a CLI toggle (e.g., `blob_sidecars`).

### Consuming additional beacon/relay data

- Consider adding a typed method to `pkg/clientapi` for the beacon API you want and expose it via the analyzer. Reuse `utils.RoutineBook` to avoid overwhelming remote nodes.
- For MEV-related data, add a relay URI to `pkg/relay/constants.go` or enrich `RelayBidsPerSlot`.

### Introducing CLI flags

- Add the flag in `cmd/*.go`, update defaults/env var name in `pkg/config/defaults.go`, and thread the value through the config struct.

### Handling new forks

- Implement fork-specific state/block conversions in `pkg/spec` (`New<Fork>Block`, `New<Fork>State`).
- Provide a matching `StateMetrics` implementation in `pkg/spec/metrics/state_<fork>.go`.
- Adjust reward math and Electra/Fulu data structures as needed so `processEpochValRewards` keeps working.

---

## 8. Directory Reference Map

| Path | What to look for |
| --- | --- |
| `cmd/` | CLI wiring and flag definitions. |
| `pkg/analyzer/` | Pipeline orchestration, download loops, processing steps, reorg handling, Prometheus integration. |
| `pkg/clientapi/` | Beacon/execution RPC wrappers, retry logic, blob downloads, state roots, reward API calls. |
| `pkg/db/` | ClickHouse service, batching, table-specific persist/delete logic, migrations. |
| `pkg/spec/` | Fork-agnostic representations and reward math. |
| `pkg/events/` | SSE subscription glue code for head/finality/reorg/blob events. |
| `pkg/relay/` | MEV relay monitor and bid aggregation. |
| `pkg/utils/` | Logging, compression, concurrency helpers. |
| `pkg/validator_window/` | Pruning workflow for validator reward tables. |
| `tests/` | Python-based ClickHouse smoke tests. |

---

## 9. Operational Guardrails

- **Backpressure**: `utils.RoutineBook` limits concurrent downloads/processes. Respect it when adding goroutines to avoid starving the node or DB.
- **Cleanup windows**: `ChainCache.CleanUpTo` and `AdvanceFinalized` prevent unbounded memory growth; ensure long-running loops keep updating `s.stop` and checking caches.
- **Retries**: All beacon/execution calls honor `--max-request-retries`; avoid unconditional infinite retries.
- **DB lifecycle**: Always call `DBService.Finish()` (already done in `ChainAnalyzer.Run()` and `ValidatorWindowRunner.EndProcesses()`) so bulk-insert buffers flush cleanly on shutdown.

Following these conventions will help you extend GotEth without breaking the ingestion pipeline or the downstream dashboards that depend on it.

