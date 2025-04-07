ALTER TABLE t_epoch_metrics_summary DROP COLUMN f_num_compounding_vals;

ALTER TABLE t_pool_summary DROP COLUMN number_compounding_vals;

ALTER TABLE t_validator_rewards_summary DROP COLUMN f_withdrawal_prefix;

ALTER TABLE t_validator_last_status DROP COLUMN f_withdrawal_prefix;
ALTER TABLE t_validator_last_status DROP COLUMN f_withdrawal_credentials;