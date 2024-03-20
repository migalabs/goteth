CREATE TABLE IF NOT EXISTS t_block_rewards(
	f_slot UInt64,
	f_reward_fees UInt64,
	f_burnt_fees UInt64,
	f_cl_manual_reward UInt64,
	f_cl_api_reward UInt64,
	f_relays Array(TEXT),
	f_builder_pubkey Array(TEXT),
	f_bid_commission UInt64)
	ENGINE = ReplacingMergeTree()
	ORDER BY (f_slot);

INSERT INTO t_block_rewards(f_slot, f_cl_manual_reward) 
	SELECT f_slot, f_block_experimental_reward
	FROM (
		SELECT f_epoch, f_val_idx, f_block_experimental_reward 
		FROM t_validator_rewards_summary 
		WHERE f_block_experimental_reward > 0
	) as experimental_block_reward
	inner join t_block_metrics
	on experimental_block_reward.f_epoch = intDiv(t_block_metrics.f_slot, 32) AND
		 experimental_block_reward.f_val_idx = t_block_metrics.f_proposer_index;

INSERT INTO t_block_rewards(f_slot, f_cl_api_reward) 
	SELECT f_slot, f_block_api_reward
	FROM (
		SELECT f_epoch, f_val_idx, f_block_api_reward 
		FROM t_validator_rewards_summary 
		WHERE f_block_api_reward > 0
	) as api_block_reward
	inner join t_block_metrics
	on api_block_reward.f_epoch = intDiv(t_block_metrics.f_slot, 32) AND
		 api_block_reward.f_val_idx = t_block_metrics.f_proposer_index;

ALTER TABLE t_validator_rewards_summary DROP COLUMN f_block_experimental_reward;
ALTER TABLE t_validator_rewards_summary DROP COLUMN f_block_api_reward;
