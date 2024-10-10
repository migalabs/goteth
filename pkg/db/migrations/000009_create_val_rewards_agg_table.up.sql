CREATE TABLE t_validator_rewards_aggregation(
	f_val_idx UInt64,
	f_start_epoch UInt64,
	f_end_epoch UInt64,
	f_reward Int64,
	f_max_reward UInt64,
	f_max_att_reward UInt64,
	f_max_sync_reward UInt64,
	f_base_reward UInt64,
	f_in_sync_committee_count UInt16,
	f_missing_source_count UInt16,
	f_missing_target_count UInt16,
	f_missing_head_count UInt16,
	f_block_api_reward UInt64,
	f_block_experimental_reward UInt64,
	f_inclusion_delay_sum UInt32
	)
	ENGINE = ReplacingMergeTree()
	ORDER BY (f_start_epoch, f_val_idx);