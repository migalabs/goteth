
DROP TABLE IF EXISTS t_block_rewards

ALTER TABLE t_validator_rewards_summary ADD COLUMN f_block_experimental_reward UInt64;
ALTER TABLE t_validator_rewards_summary ADD COLUMN f_block_api_reward UInt64;
