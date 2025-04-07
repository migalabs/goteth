ALTER TABLE t_epoch_metrics_summary ADD COLUMN f_num_compounding_vals UInt64 AFTER f_num_vals;

ALTER TABLE t_pool_summary ADD COLUMN number_compounding_vals UInt64 AFTER number_active_vals;

ALTER TABLE t_validator_rewards_summary ADD COLUMN f_withdrawal_prefix UInt8 AFTER f_balance_eth;

ALTER TABLE t_validator_last_status ADD COLUMN f_withdrawal_prefix UInt8;
ALTER TABLE t_validator_last_status ADD COLUMN f_withdrawal_credentials TEXT;