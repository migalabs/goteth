-- ============================================
-- DASHBOARD QUERIES WITH EL REWARDS
-- Database: goteth_mainnet
-- ============================================
-- IMPORTANT: These queries replace current dashboard queries
-- that only calculate APR with CL rewards. They include EL rewards (fees + MEV)
-- ============================================

-- ===========================================
-- QUERY 1: NETWORK-WIDE APR BY EPOCH
-- ===========================================
-- Use this query to display network-wide APR by epoch
-- Includes CL rewards + EL rewards (fees + MEV)

WITH epoch_aggregated AS (
    SELECT 
        vr.f_epoch,
        SUM(vr.f_effective_balance) as total_effective_balance_gwei,
        SUM(vr.f_reward) as total_cl_rewards_gwei,
        COUNT(DISTINCT vr.f_val_idx) as active_validators,
        -- EL Rewards aggregated by epoch
        COALESCE(SUM(br.f_reward_fees / 1e9), 0) as total_el_fees_gwei,
        COALESCE(SUM(br.f_bid_commission / 1e9), 0) as total_el_mev_gwei
    FROM t_validator_rewards_summary vr
    LEFT JOIN t_block_metrics bm 
        ON bm.f_epoch = vr.f_epoch
        AND bm.f_proposed = true
    LEFT JOIN t_block_rewards br 
        ON br.f_slot = bm.f_slot
    WHERE vr.f_epoch >= (SELECT MAX(f_epoch) - 225 FROM t_validator_rewards_summary)
      -- 225 epochs = ~30 days for real-time dashboards
    GROUP BY vr.f_epoch
)
SELECT 
    f_epoch,
    active_validators,
    round(total_effective_balance_gwei / 1e9, 2) as total_staked_eth,
    
    -- Detailed rewards
    round(total_cl_rewards_gwei / 1e9, 4) as total_cl_rewards_eth,
    round(total_el_fees_gwei / 1e9, 4) as total_el_fees_eth,
    round(total_el_mev_gwei / 1e9, 4) as total_el_mev_eth,
    round((total_cl_rewards_gwei + total_el_fees_gwei + total_el_mev_gwei) / 1e9, 4) as total_rewards_with_el_eth,
    
    -- APR WITHOUT EL (INCORRECT - current state)
    round(
        (total_cl_rewards_gwei / total_effective_balance_gwei) * 365.25 * 100,
        4
    ) as network_apr_cl_only_percent,
    
    -- APR WITH EL (CORRECT - target state)
    round(
        ((total_cl_rewards_gwei + total_el_fees_gwei + total_el_mev_gwei) / total_effective_balance_gwei) * 365.25 * 100,
        4
    ) as network_apr_with_el_percent,
    
    -- EL contribution to APR
    round(
        ((total_el_fees_gwei + total_el_mev_gwei) / total_effective_balance_gwei) * 365.25 * 100,
        4
    ) as el_contribution_to_apr_percent
    
FROM epoch_aggregated
ORDER BY f_epoch DESC
LIMIT 30;  -- Last 30 epochs


-- ===========================================
-- QUERY 2: AVERAGE NETWORK APR (LAST 30 DAYS)
-- ===========================================
-- Simple query to display a single network APR value

WITH last_225_epochs AS (
    SELECT 
        SUM(vr.f_effective_balance) as total_effective_balance_gwei,
        SUM(vr.f_reward) as total_cl_rewards_gwei,
        COALESCE(SUM(br.f_reward_fees / 1e9), 0) as total_el_fees_gwei,
        COALESCE(SUM(br.f_bid_commission / 1e9), 0) as total_el_mev_gwei
    FROM t_validator_rewards_summary vr
    LEFT JOIN t_block_metrics bm 
        ON bm.f_epoch = vr.f_epoch AND bm.f_proposed = true
    LEFT JOIN t_block_rewards br 
        ON br.f_slot = bm.f_slot
    WHERE vr.f_epoch >= (SELECT MAX(f_epoch) - 225 FROM t_validator_rewards_summary)
)
SELECT 
    -- CORRECT APR (with EL rewards)
    round(
        ((total_cl_rewards_gwei + total_el_fees_gwei + total_el_mev_gwei) / total_effective_balance_gwei) 
        * 365.25 * 100,
        2
    ) as network_apr_percent,
    
    -- Breakdown
    round((total_cl_rewards_gwei / total_effective_balance_gwei) * 365.25 * 100, 2) as cl_apr_percent,
    round(((total_el_fees_gwei + total_el_mev_gwei) / total_effective_balance_gwei) * 365.25 * 100, 2) as el_apr_percent
    
FROM last_225_epochs;


-- ===========================================
-- QUERY 3: APR PER VALIDATOR (TOP 100)
-- ===========================================
-- To display individual validator APR

WITH validator_epoch_data AS (
    SELECT 
        vr.f_val_idx,
        COUNT(DISTINCT vr.f_epoch) as epochs_count,
        AVG(vr.f_effective_balance) as avg_effective_balance_gwei,
        SUM(vr.f_reward) as total_cl_reward_gwei,
        COALESCE(SUM(br.f_reward_fees / 1e9), 0) as total_el_fees_gwei,
        COALESCE(SUM(br.f_bid_commission / 1e9), 0) as total_el_mev_gwei
    FROM t_validator_rewards_summary vr
    LEFT JOIN t_block_metrics bm 
        ON bm.f_proposer_index = vr.f_val_idx 
        AND bm.f_epoch = vr.f_epoch
        AND bm.f_proposed = true
    LEFT JOIN t_block_rewards br 
        ON br.f_slot = bm.f_slot
    WHERE vr.f_epoch >= (SELECT MAX(f_epoch) - 225 FROM t_validator_rewards_summary)
    GROUP BY vr.f_val_idx
    HAVING epochs_count > 200  -- Minimum 200 epochs for representative data
)
SELECT 
    f_val_idx as validator_index,
    round(avg_effective_balance_gwei, 0) as effective_balance_gwei,
    round(total_cl_reward_gwei / 1e9, 4) as total_cl_rewards_eth,
    round((total_el_fees_gwei + total_el_mev_gwei) / 1e9, 4) as total_el_rewards_eth,
    round((total_cl_reward_gwei + total_el_fees_gwei + total_el_mev_gwei) / 1e9, 4) as total_rewards_eth,
    
    -- APR WITH EL (correct)
    round(
        ((total_cl_reward_gwei + total_el_fees_gwei + total_el_mev_gwei) / avg_effective_balance_gwei) 
        * (365.25 / epochs_count) * 100,
        4
    ) as apr_with_el_percent,
    
    -- EL percentage of total
    round(
        (total_el_fees_gwei + total_el_mev_gwei) / 
        (total_cl_reward_gwei + total_el_fees_gwei + total_el_mev_gwei) * 100,
        2
    ) as el_percentage_of_total
    
FROM validator_epoch_data
ORDER BY total_rewards_eth DESC
LIMIT 100;


-- ===========================================
-- QUERY 4: APR PER POOL (if t_eth2_pubkeys exists)
-- ===========================================
-- To display APR per staking pool

WITH pool_epoch_data AS (
    SELECT 
        pk.f_pool_name,
        COUNT(DISTINCT vr.f_epoch) as epochs_count,
        AVG(SUM(vr.f_effective_balance)) as avg_total_effective_balance_gwei,
        SUM(vr.f_reward) as total_cl_rewards_gwei,
        COALESCE(SUM(br.f_reward_fees / 1e9), 0) as total_el_fees_gwei,
        COALESCE(SUM(br.f_bid_commission / 1e9), 0) as total_el_mev_gwei
    FROM t_validator_rewards_summary vr
    INNER JOIN t_eth2_pubkeys pk 
        ON pk.f_val_idx = vr.f_val_idx
    LEFT JOIN t_block_metrics bm 
        ON bm.f_proposer_index = vr.f_val_idx 
        AND bm.f_epoch = vr.f_epoch
        AND bm.f_proposed = true
    LEFT JOIN t_block_rewards br 
        ON br.f_slot = bm.f_slot
    WHERE vr.f_epoch >= (SELECT MAX(f_epoch) - 225 FROM t_validator_rewards_summary)
      AND pk.f_pool_name != ''
    GROUP BY pk.f_pool_name, vr.f_epoch
)
SELECT 
    f_pool_name as pool_name,
    round(avg_total_effective_balance_gwei / 1e9, 2) as avg_staked_eth,
    
    -- APR WITH EL (correct)
    round(
        AVG((total_cl_rewards_gwei + total_el_fees_gwei + total_el_mev_gwei) / avg_total_effective_balance_gwei) 
        * 365.25 * 100,
        4
    ) as apr_with_el_percent,
    
    -- Breakdown CL vs EL
    round(AVG(total_cl_rewards_gwei / avg_total_effective_balance_gwei) * 365.25 * 100, 4) as cl_apr_percent,
    round(AVG((total_el_fees_gwei + total_el_mev_gwei) / avg_total_effective_balance_gwei) * 365.25 * 100, 4) as el_apr_percent,
    
    -- Total rewards
    round(SUM(total_cl_rewards_gwei + total_el_fees_gwei + total_el_mev_gwei) / 1e9, 2) as total_rewards_eth
    
FROM pool_epoch_data
GROUP BY f_pool_name
HAVING avg_staked_eth > 32  -- Al menos 1 validator
ORDER BY apr_with_el_percent DESC
LIMIT 50;


-- ===========================================
-- QUERY 5: VALIDATION - COMPARE BEFORE VS AFTER
-- ===========================================
-- Query to verify that the change works correctly
-- Shows the difference between current APR (without EL) vs corrected APR (with EL)

WITH network_apr AS (
    SELECT 
        SUM(vr.f_effective_balance) as total_balance_gwei,
        SUM(vr.f_reward) as total_cl_gwei,
        COALESCE(SUM(br.f_reward_fees / 1e9), 0) + COALESCE(SUM(br.f_bid_commission / 1e9), 0) as total_el_gwei
    FROM t_validator_rewards_summary vr
    LEFT JOIN t_block_metrics bm ON bm.f_epoch = vr.f_epoch AND bm.f_proposed = true
    LEFT JOIN t_block_rewards br ON br.f_slot = bm.f_slot
    WHERE vr.f_epoch >= (SELECT MAX(f_epoch) - 225 FROM t_validator_rewards_summary)
)
SELECT 
    'CURRENT APR (CL only)' as type,
    round((total_cl_gwei / total_balance_gwei) * 365.25 * 100, 4) as apr_percent
FROM network_apr
UNION ALL
SELECT 
    'CORRECTED APR (CL + EL)' as type,
    round(((total_cl_gwei + total_el_gwei) / total_balance_gwei) * 365.25 * 100, 4) as apr_percent
FROM network_apr
UNION ALL
SELECT 
    'DIFFERENCE' as type,
    round((total_el_gwei / total_balance_gwei) * 365.25 * 100, 4) as apr_percent
FROM network_apr;


-- ============================================
-- IMPLEMENTATION NOTES
-- ============================================
--
-- 1. UNIT CONVERSION:
--    - CL rewards (f_reward): Already in Gwei
--    - EL fees (f_reward_fees): In Wei, divide by 1e9 for Gwei
--    - EL MEV (f_bid_commission): In Wei, divide by 1e9 for Gwei
--
-- 2. APR FORMULA:
--    APR = (total_rewards / effective_balance) * (365.25 / epochs) * 100
--
-- 3. PERFORMANCE:
--    - JOINs use LEFT JOIN to avoid losing non-proposing validators
--    - Use indexes on f_epoch, f_slot, f_proposer_index if performance issues occur
--
-- 4. VALIDATION:
--    - Run QUERY 5 to verify APR increases correctly (~3-4%)
--    - If no increase, check Wei to Gwei conversion and JOINs
--
-- 5. DASHBOARD:
--    - Query 1: APR by epoch chart (timeline)
--    - Query 2: Single network APR metric (large number)
--    - Query 3: Top validators by APR table
--    - Query 4: Pools by APR table
--    - Query 5: Validation widget showing before/after
--
-- ============================================
