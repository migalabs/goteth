-- Remove f_effective_balance column from t_validator_last_status
ALTER TABLE t_validator_last_status
DROP COLUMN f_effective_balance;

-- Remove f_effective_balance column from t_validator_rewards_summary
ALTER TABLE t_validator_rewards_summary
DROP COLUMN f_effective_balance;

-- Remove aggregated_effective_balance column from t_pool_summary
ALTER TABLE t_pool_summary
DROP COLUMN aggregated_effective_balance;

