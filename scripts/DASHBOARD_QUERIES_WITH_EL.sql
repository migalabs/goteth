-- ============================================
-- QUERIES PARA DASHBOARDS CON EL REWARDS
-- Database: goteth_mainnet
-- ============================================
-- IMPORTANTE: Estas queries reemplazan las queries actuales del dashboard
-- que solo calculan APR con CL rewards. Incluyen EL rewards (fees + MEV)
-- ============================================

-- ===========================================
-- QUERY 1: APR NETWORK-WIDE POR ÉPOCA
-- ===========================================
-- Usar esta query para mostrar el APR de toda la red por época
-- Incluye CL rewards + EL rewards (fees + MEV)

WITH epoch_aggregated AS (
    SELECT 
        vr.f_epoch,
        SUM(vr.f_effective_balance) as total_effective_balance_gwei,
        SUM(vr.f_reward) as total_cl_rewards_gwei,
        COUNT(DISTINCT vr.f_val_idx) as active_validators,
        -- EL Rewards agregados por época
        COALESCE(SUM(br.f_reward_fees / 1e9), 0) as total_el_fees_gwei,
        COALESCE(SUM(br.f_bid_commission / 1e9), 0) as total_el_mev_gwei
    FROM t_validator_rewards_summary vr
    LEFT JOIN t_block_metrics bm 
        ON bm.f_epoch = vr.f_epoch
        AND bm.f_proposed = true
    LEFT JOIN t_block_rewards br 
        ON br.f_slot = bm.f_slot
    WHERE vr.f_epoch >= (SELECT MAX(f_epoch) - 225 FROM t_validator_rewards_summary)
      -- 225 epochs = ~30 días para dashboards en tiempo real
    GROUP BY vr.f_epoch
)
SELECT 
    f_epoch,
    active_validators,
    round(total_effective_balance_gwei / 1e9, 2) as total_staked_eth,
    
    -- Rewards detallados
    round(total_cl_rewards_gwei / 1e9, 4) as total_cl_rewards_eth,
    round(total_el_fees_gwei / 1e9, 4) as total_el_fees_eth,
    round(total_el_mev_gwei / 1e9, 4) as total_el_mev_eth,
    round((total_cl_rewards_gwei + total_el_fees_gwei + total_el_mev_gwei) / 1e9, 4) as total_rewards_with_el_eth,
    
    -- APR SIN EL (INCORRECTO - lo que tienen ahora)
    round(
        (total_cl_rewards_gwei / total_effective_balance_gwei) * 365.25 * 100,
        4
    ) as network_apr_cl_only_percent,
    
    -- APR CON EL (CORRECTO - lo que deberían mostrar)
    round(
        ((total_cl_rewards_gwei + total_el_fees_gwei + total_el_mev_gwei) / total_effective_balance_gwei) * 365.25 * 100,
        4
    ) as network_apr_with_el_percent,
    
    -- Contribución de EL al APR
    round(
        ((total_el_fees_gwei + total_el_mev_gwei) / total_effective_balance_gwei) * 365.25 * 100,
        4
    ) as el_contribution_to_apr_percent
    
FROM epoch_aggregated
ORDER BY f_epoch DESC
LIMIT 30;  -- Últimos 30 epochs


-- ===========================================
-- QUERY 2: APR PROMEDIO RED (ÚLTIMOS 30 DÍAS)
-- ===========================================
-- Query simple para mostrar un único valor de APR de la red

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
    -- APR CORRECTO (con EL rewards)
    round(
        ((total_cl_rewards_gwei + total_el_fees_gwei + total_el_mev_gwei) / total_effective_balance_gwei) 
        * 365.25 * 100,
        2
    ) as network_apr_percent,
    
    -- Desglose
    round((total_cl_rewards_gwei / total_effective_balance_gwei) * 365.25 * 100, 2) as cl_apr_percent,
    round(((total_el_fees_gwei + total_el_mev_gwei) / total_effective_balance_gwei) * 365.25 * 100, 2) as el_apr_percent
    
FROM last_225_epochs;


-- ===========================================
-- QUERY 3: APR POR VALIDATOR (TOP 100)
-- ===========================================
-- Para mostrar APR de validators individuales

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
    HAVING epochs_count > 200  -- Mínimo 200 epochs para tener datos representativos
)
SELECT 
    f_val_idx as validator_index,
    round(avg_effective_balance_gwei, 0) as effective_balance_gwei,
    round(total_cl_reward_gwei / 1e9, 4) as total_cl_rewards_eth,
    round((total_el_fees_gwei + total_el_mev_gwei) / 1e9, 4) as total_el_rewards_eth,
    round((total_cl_reward_gwei + total_el_fees_gwei + total_el_mev_gwei) / 1e9, 4) as total_rewards_eth,
    
    -- APR CON EL (correcto)
    round(
        ((total_cl_reward_gwei + total_el_fees_gwei + total_el_mev_gwei) / avg_effective_balance_gwei) 
        * (365.25 / epochs_count) * 100,
        4
    ) as apr_with_el_percent,
    
    -- Porcentaje de EL sobre total
    round(
        (total_el_fees_gwei + total_el_mev_gwei) / 
        (total_cl_reward_gwei + total_el_fees_gwei + total_el_mev_gwei) * 100,
        2
    ) as el_percentage_of_total
    
FROM validator_epoch_data
ORDER BY total_rewards_eth DESC
LIMIT 100;


-- ===========================================
-- QUERY 4: APR POR POOL (si tienen t_eth2_pubkeys)
-- ===========================================
-- Para mostrar APR por pool de staking

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
    
    -- APR CON EL (correcto)
    round(
        AVG((total_cl_rewards_gwei + total_el_fees_gwei + total_el_mev_gwei) / avg_total_effective_balance_gwei) 
        * 365.25 * 100,
        4
    ) as apr_with_el_percent,
    
    -- Desglose CL vs EL
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
-- QUERY 5: VALIDACIÓN - COMPARAR ANTES VS DESPUÉS
-- ===========================================
-- Query para verificar que el cambio funciona correctamente
-- Muestra la diferencia entre APR actual (sin EL) vs APR corregido (con EL)

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
    'APR ACTUAL (solo CL)' as tipo,
    round((total_cl_gwei / total_balance_gwei) * 365.25 * 100, 4) as apr_percent
FROM network_apr
UNION ALL
SELECT 
    'APR CORREGIDO (CL + EL)' as tipo,
    round(((total_cl_gwei + total_el_gwei) / total_balance_gwei) * 365.25 * 100, 4) as apr_percent
FROM network_apr
UNION ALL
SELECT 
    'DIFERENCIA' as tipo,
    round((total_el_gwei / total_balance_gwei) * 365.25 * 100, 4) as apr_percent
FROM network_apr;


-- ============================================
-- NOTAS DE IMPLEMENTACIÓN
-- ============================================
--
-- 1. CONVERSIÓN DE UNIDADES:
--    - CL rewards (f_reward): Ya están en Gwei
--    - EL fees (f_reward_fees): Están en Wei → dividir entre 1e9 para Gwei
--    - EL MEV (f_bid_commission): Están en Wei → dividir entre 1e9 para Gwei
--
-- 2. FÓRMULA APR:
--    APR = (total_rewards / effective_balance) * (365.25 / epochs) * 100
--
-- 3. PERFORMANCE:
--    - Los JOINs son con LEFT JOIN para no perder validators que no propusieron
--    - Usar índices en f_epoch, f_slot, f_proposer_index si hay problemas de performance
--
-- 4. VALIDACIÓN:
--    - Ejecutar QUERY 5 para verificar que el APR aumenta correctamente (~3-4%)
--    - Si no aumenta, revisar conversión Wei→Gwei y los JOINs
--
-- 5. DASHBOARD:
--    - Query 1: Gráfico de APR por época (línea temporal)
--    - Query 2: Métrica única de APR actual de la red (número grande)
--    - Query 3: Tabla de top validators por APR
--    - Query 4: Tabla de pools por APR
--    - Query 5: Widget de validación mostrando antes/después
--
-- ============================================
