#!/usr/bin/env python3
"""
Verification script to demonstrate APR calculation discrepancy when excluding EL rewards.

This script connects to a Clickhouse database containing goteth data and compares:
- APR calculated with CL rewards only (current implementation)
- APR calculated with CL + EL rewards (correct implementation)

Usage:
    python verify_apr_calculation.py

Requirements:
    - clickhouse-connect
    - Active SSH tunnel to Clickhouse server (if remote)
    - Database credentials configured in this script

Configuration:
    Update the connection parameters below with your environment settings.
"""

import clickhouse_connect
import sys

# Database connection configuration
# Update these values to match your environment
DB_CONFIG = {
    'host': 'localhost',
    'port': 8123,
    'username': 'your_username',
    'password': 'your_password',
    'database': 'goteth_mainnet'
}

# Analysis parameters
EPOCHS_TO_ANALYZE = 100  # Approximately 11 hours of data

def main():
    print("="*80)
    print("APR CALCULATION VERIFICATION - CL vs CL+EL")
    print("="*80)
    
    try:
        client = clickhouse_connect.get_client(**DB_CONFIG)
    except Exception as e:
        print(f"Error connecting to database: {e}")
        print("Ensure SSH tunnel is active and credentials are correct.")
        sys.exit(1)

    query = f"""
    WITH epoch_range AS (
        SELECT 
            MAX(f_epoch) - {EPOCHS_TO_ANALYZE} as start_epoch,
            MAX(f_epoch) - 1 as end_epoch
        FROM t_validator_rewards_summary
    ),
    cl_rewards_total AS (
        SELECT 
            SUM(f_effective_balance) as total_balance_gwei,
            SUM(f_reward) as total_cl_rewards_gwei,
            COUNT(*) as total_validator_epochs
        FROM t_validator_rewards_summary vr
        CROSS JOIN epoch_range er
        WHERE vr.f_epoch >= er.start_epoch 
          AND vr.f_epoch <= er.end_epoch
    ),
    el_rewards_total AS (
        SELECT 
            SUM(br.f_reward_fees / 1e9) as total_el_fees_gwei,
            SUM(br.f_bid_commission / 1e9) as total_el_mev_gwei,
            COUNT(DISTINCT bm.f_slot) as blocks_proposed
        FROM t_block_rewards br
        INNER JOIN t_block_metrics bm ON bm.f_slot = br.f_slot
        CROSS JOIN epoch_range er
        WHERE bm.f_epoch >= er.start_epoch 
          AND bm.f_epoch <= er.end_epoch
          AND bm.f_proposed = true
    )
    SELECT 
        round(cl.total_balance_gwei / 1e9, 2) as total_staked_eth,
        round(cl.total_cl_rewards_gwei / 1e9, 2) as total_cl_rewards_eth,
        cl.total_validator_epochs,
        round(el.total_el_fees_gwei / 1e9, 2) as total_el_fees_eth,
        round(el.total_el_mev_gwei / 1e9, 2) as total_el_mev_eth,
        round((el.total_el_fees_gwei + el.total_el_mev_gwei) / 1e9, 2) as total_el_eth,
        el.blocks_proposed,
        round((cl.total_cl_rewards_gwei / cl.total_balance_gwei) * 365.25 * 100, 4) as apr_cl_only_percent,
        round(((cl.total_cl_rewards_gwei + el.total_el_fees_gwei + el.total_el_mev_gwei) / cl.total_balance_gwei) * 365.25 * 100, 4) as apr_with_el_percent,
        round(((el.total_el_fees_gwei + el.total_el_mev_gwei) / cl.total_balance_gwei) * 365.25 * 100, 4) as apr_el_contribution_percent,
        round((el.total_el_fees_gwei + el.total_el_mev_gwei) / cl.total_cl_rewards_gwei * 100, 2) as el_percent_of_cl
    FROM cl_rewards_total cl
    CROSS JOIN el_rewards_total el
    """
    
    print(f"\nAnalyzing last {EPOCHS_TO_ANALYZE} epochs (approximately {EPOCHS_TO_ANALYZE * 6.4 / 60:.1f} hours)")
    print("Executing query...")
    
    try:
        result = client.query(query)
    except Exception as e:
        print(f"Query execution failed: {e}")
        sys.exit(1)
    
    print("-"*80)
    
    if not result.result_rows:
        print("No results returned. Verify data exists in the specified epoch range.")
        sys.exit(1)
    
    row = result.result_rows[0]
    (staked_eth, cl_eth, val_epochs, fees_eth, mev_eth, total_el_eth, blocks,
     apr_cl, apr_with_el, apr_el, el_pct) = row
    
    print("\nRESULTS:")
    print("-"*80)
    print(f"Total staked (average):      {staked_eth:>15,.2f} ETH")
    print(f"Validator-epochs processed:  {val_epochs:>15,}")
    print(f"Blocks proposed:             {blocks:>15,}")
    print()
    print("REWARDS BREAKDOWN:")
    print(f"  CL rewards (all):          {cl_eth:>15,.2f} ETH  (100.00%)")
    print(f"  EL fees (proposers):       {fees_eth:>15,.2f} ETH")
    print(f"  EL MEV (proposers):        {mev_eth:>15,.2f} ETH")
    print(f"  EL TOTAL:                  {total_el_eth:>15,.2f} ETH  ({el_pct:>6.2f}%)")
    print(f"  {'─'*60}")
    print(f"  TOTAL (CL + EL):           {cl_eth + total_el_eth:>15,.2f} ETH")
    print()
    print("APR COMPARISON:")
    print(f"  Current APR (CL only):     {apr_cl:>15.4f}%")
    print(f"  Correct APR (CL + EL):     {apr_with_el:>15.4f}%")
    print(f"  EL contribution to APR:    +{apr_el:>14.4f}%")
    print()
    
    if el_pct > 1.0:
        increase_pct = (apr_el / apr_cl * 100)
        print("CONCLUSION:")
        print(f"  Issue confirmed: EL rewards represent {el_pct:.2f}% of CL rewards")
        print(f"  Current APR underestimated by {increase_pct:.1f}%")
        print(f"  Dashboards should display {apr_with_el:.4f}% instead of {apr_cl:.4f}%")
    else:
        print(f"WARNING: EL rewards percentage ({el_pct:.2f}%) is unexpectedly low.")
        print("Consider increasing EPOCHS_TO_ANALYZE or verify data integrity.")
    
    print("\n" + "="*80)
    print("See scripts/DASHBOARD_QUERIES_WITH_EL.sql for implementation queries.")
    print("See docs/EL_REWARDS_AND_APR.md for detailed documentation.")
    print("="*80)

if __name__ == "__main__":
    main()
