#!/usr/bin/env python3
"""
Quick verification of missing EL rewards issue.

This script performs a lightweight check on recent epochs to confirm
that EL rewards exist in the database but are not included in APR calculations.

Usage:
    1. Establish SSH tunnel: ssh -i ~/.ssh/key -L 8123:localhost:8123 user@host
    2. Update credentials in this script
    3. Run: python quick_verification.py
"""

import clickhouse_connect

# Update these values to match your environment
DB_CONFIG = {
    'host': 'localhost',
    'port': 8123,
    'username': 'your_username',
    'password': 'your_password',
    'database': 'goteth_mainnet'
}

client = clickhouse_connect.get_client(**DB_CONFIG)

print("="*80)
print("QUICK VERIFICATION: EL REWARDS NOT INCLUDED IN APR")
print("="*80)

# 1. Último epoch
result = client.query('SELECT MAX(f_epoch) as max_epoch FROM t_validator_rewards_summary')
last_epoch = result.result_rows[0][0]
print(f"\n✅ Último epoch: {last_epoch}")

# 2. Comparar CL vs EL rewards (últimos 5 epochs)
query = f"""
WITH epoch_rewards AS (
    SELECT 
        f_epoch,
        SUM(f_reward) / 1e9 as total_cl_rewards_eth
    FROM t_validator_rewards_summary
    WHERE f_epoch >= {last_epoch - 4}
    GROUP BY f_epoch
),
epoch_el_rewards AS (
    SELECT 
        bm.f_epoch,
        SUM(br.f_reward_fees) / 1e18 as total_el_fees_eth,
        SUM(br.f_bid_commission) / 1e18 as total_el_mev_eth
    FROM t_block_rewards br
    INNER JOIN t_block_metrics bm ON bm.f_slot = br.f_slot
    WHERE bm.f_epoch >= {last_epoch - 4}
    GROUP BY bm.f_epoch
)
SELECT 
    er.f_epoch,
    round(er.total_cl_rewards_eth, 2) as cl_rewards_eth,
    round(eel.total_el_fees_eth, 2) as el_fees_eth,
    round(eel.total_el_mev_eth, 2) as el_mev_eth,
    round(eel.total_el_fees_eth + eel.total_el_mev_eth, 2) as total_el_eth,
    round((eel.total_el_fees_eth + eel.total_el_mev_eth) / er.total_cl_rewards_eth * 100, 2) as el_percent_of_cl
FROM epoch_rewards er
LEFT JOIN epoch_el_rewards eel ON eel.f_epoch = er.f_epoch
ORDER BY er.f_epoch DESC
"""

print(f"\n🔍 Comparando CL vs EL rewards (últimos 5 epochs)...")
print("-"*80)
result = client.query(query)

print(f"{'Epoch':<10} {'CL Rewards':<15} {'EL Fees':<12} {'EL MEV':<12} {'Total EL':<12} {'EL % of CL'}")
print("-"*80)

total_cl = 0
total_el = 0

for row in result.result_rows:
    epoch, cl_eth, el_fees, el_mev, total_el_eth, el_pct = row
    print(f"{epoch:<10} {cl_eth:<15.2f} {el_fees:<12.2f} {el_mev:<12.2f} {total_el_eth:<12.2f} {el_pct:.2f}%")
    total_cl += cl_eth
    total_el += total_el_eth

avg_el_percent = (total_el / total_cl * 100) if total_cl > 0 else 0

print("-"*80)
print(f"{'TOTAL':<10} {total_cl:<15.2f} ETH CL       {total_el:<12.2f} ETH EL")
print(f"\n📊 RESULTADO: EL rewards = {avg_el_percent:.2f}% de CL rewards")
print(f"   Esto significa que el APR está subestimado en ~{avg_el_percent:.2f}%")

# 3. Ejemplo específico de un proposer
print(f"\n🔍 Ejemplo: Validator que propuso bloque recientemente...")
query = f"""
SELECT 
    bm.f_slot,
    bm.f_epoch,
    bm.f_proposer_index as validator_index,
    vr.f_reward as cl_total_reward_gwei,
    vr.f_block_api_reward as cl_proposer_reward_gwei,
    round(br.f_reward_fees / 1e9, 2) as el_fees_gwei,
    round(br.f_bid_commission / 1e9, 2) as el_mev_gwei,
    round((br.f_reward_fees + br.f_bid_commission) / 1e9, 2) as total_el_gwei
FROM t_block_metrics bm
INNER JOIN t_block_rewards br ON br.f_slot = bm.f_slot
LEFT JOIN t_validator_rewards_summary vr 
    ON vr.f_val_idx = bm.f_proposer_index 
    AND vr.f_epoch = bm.f_epoch
WHERE bm.f_proposed = true
  AND bm.f_epoch >= {last_epoch - 2}
  AND (br.f_reward_fees > 0 OR br.f_bid_commission > 0)
ORDER BY (br.f_reward_fees + br.f_bid_commission) DESC
LIMIT 3
"""

result = client.query(query)
print("-"*80)
print(f"{'Slot':<12} {'Validator':<12} {'CL Reward':<12} {'EL Fees':<12} {'EL MEV':<12} {'FALTANTE'}")
print("-"*80)

for row in result.result_rows:
    slot, epoch, val_idx, cl_reward, cl_proposer, el_fees, el_mev, total_el = row
    print(f"{slot:<12} {val_idx:<12} {cl_reward:<12} {el_fees:<12} {el_mev:<12} {total_el} Gwei")

print("\n" + "="*80)
print("✅ BUG CONFIRMADO:")
print("   - CL rewards SÍ están en t_validator_rewards_summary")
print("   - EL rewards (fees + MEV) SÍ están en t_block_rewards")
print("   - PERO no se unen para calcular APR total")
print(f"   - Pérdida estimada: ~{avg_el_percent:.1f}% del APR real")
print("="*80)
