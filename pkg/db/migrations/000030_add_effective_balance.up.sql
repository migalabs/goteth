-- Add f_effective_balance column to t_validator_last_status
ALTER TABLE t_validator_last_status
ADD COLUMN f_effective_balance UInt64 DEFAULT 0 AFTER f_balance_eth;

-- Add f_effective_balance column to t_validator_rewards_summary
ALTER TABLE t_validator_rewards_summary
ADD COLUMN f_effective_balance UInt64 DEFAULT 0 AFTER f_balance_eth;

-- Add aggregated_effective_balance column to t_pool_summary
ALTER TABLE t_pool_summary
ADD COLUMN aggregated_effective_balance UInt64 DEFAULT 0 AFTER aggregated_max_rewards;

