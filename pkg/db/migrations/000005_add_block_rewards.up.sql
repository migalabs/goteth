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
