# Execution Layer Rewards and APR Integration

## Problem Statement

Current dashboards using `goteth` data display only Consensus Layer (CL) rewards and APR, excluding Execution Layer (EL) components (transaction fees and MEV). This leads to:

1. **Incomplete rewards reporting**: EL rewards are not shown alongside CL rewards
2. **Underestimated APR**: APR calculations exclude EL rewards, showing lower returns than actual

EL rewards are indexed by `goteth` in the `t_block_rewards` table but are not integrated into validator rewards summaries or APR calculations used by dashboards.

## Impact

Analysis of recent data (100 epochs, ~11 hours) demonstrates:

**Missing Rewards:**
- CL rewards (displayed): 1,192.92 ETH
- EL rewards (missing): 26.10 ETH (10.86 ETH fees + 15.24 ETH MEV)
- **EL represents 2.19% of total rewards not shown to users**

**Underestimated APR:**
- Current APR (CL only): 0.0121%
- Actual APR (CL + EL): 0.0124%
- EL contribution: +0.0003%
- **APR underestimated by 2.5%** (relative increase)

Both rewards display and APR calculations are affected.

## Root Cause

The `t_validator_rewards_summary` table and associated processing logic (`GetMaxReward` in `pkg/spec/metrics/`) only include CL-related rewards:
- Attestation rewards
- Sync committee rewards
- Base proposer rewards (CL portion)

EL rewards (transaction fees and MEV) are calculated and stored separately in `t_block_rewards` but are not included in:
1. The `ValidatorRewards` struct (`pkg/spec/validator_rewards.go`)
2. The `GetMaxReward` calculation logic
3. Dashboard queries that compute APR

## Data Availability

All necessary data is already indexed:

| Table | Column | Description |
|-------|--------|-------------|
| `t_block_rewards` | `f_reward_fees` | Transaction priority fees (Wei) |
| `t_block_rewards` | `f_bid_commission` | MEV from block builders (Wei) |
| `t_block_metrics` | `f_proposer_index` | Validator who proposed the block |
| `t_block_metrics` | `f_slot`, `f_epoch` | Block timing information |

## Proposed Solutions

### Option A: SQL-Level Integration (Recommended)

Modify dashboard queries to JOIN `t_validator_rewards_summary` with `t_block_rewards` to include EL rewards in APR calculations.

**Advantages:**
- No changes to `goteth` indexer code
- No database schema modifications
- Immediate implementation
- Minimal performance impact with proper indexing

**Implementation:**
SQL queries are provided in `scripts/DASHBOARD_QUERIES_WITH_EL.sql` covering:
1. Network-wide APR by epoch
2. Aggregate network APR
3. Per-validator APR (top performers)
4. Per-pool APR
5. Validation query (before/after comparison)

### Option B: Indexer-Level Integration

Modify `goteth` to include EL rewards in `t_validator_rewards_summary`.

**Advantages:**
- Single source of truth for all rewards
- Simplified dashboard queries
- Consistent data model

**Disadvantages:**
- Requires schema migration
- Backward compatibility considerations
- More complex implementation

## Verification

A Python script is provided in `scripts/verification/` to verify the issue and validate solutions:

```bash
python scripts/verification/verify_apr_calculation.py
```

Expected output demonstrates the ~2% difference between CL-only and CL+EL APR calculations.

## Implementation Notes

### Unit Conversion
- CL rewards (`f_reward`): Already in Gwei
- EL fees (`f_reward_fees`): In Wei, divide by 1e9 for Gwei
- EL MEV (`f_bid_commission`): In Wei, divide by 1e9 for Gwei

### APR Formula
```
APR = (total_rewards / effective_balance) × (365.25 / epochs) × 100
```

Where `total_rewards = cl_rewards + el_fees + el_mev`

### JOIN Strategy
Use `LEFT JOIN` to avoid excluding validators who did not propose blocks:

```sql
FROM t_validator_rewards_summary vr
LEFT JOIN t_block_metrics bm 
    ON bm.f_proposer_index = vr.f_val_idx 
    AND bm.f_epoch = vr.f_epoch
    AND bm.f_proposed = true
LEFT JOIN t_block_rewards br 
    ON br.f_slot = bm.f_slot
```

### Performance Considerations
- Ensure indexes exist on:
  - `t_block_metrics(f_epoch, f_proposer_index, f_proposed)`
  - `t_block_rewards(f_slot)`
- Limit epoch ranges in queries (e.g., last 225 epochs for 30-day dashboards)

## References

- [Ethereum Rewards Documentation](https://ethereum.org/en/developers/docs/consensus-mechanisms/pos/rewards-and-penalties/)
- `goteth` table schema: `docs/tables.md`
- Block rewards implementation: `pkg/analyzer/process_state.go` (lines 141-169)
