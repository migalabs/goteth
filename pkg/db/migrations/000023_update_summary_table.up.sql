DROP TABLE t_pool_summary;
CREATE TABLE IF NOT EXISTS t_pool_summary(
		f_pool_name TEXT,
		f_epoch INT,
		aggregated_rewards BIGINT,
		aggregated_max_rewards BIGINT,
		count_sync_committee INT,
		count_missing_source INT,
		count_missing_target INT,
		count_missing_head INT,
		count_expected_attestations INT,
		proposed_blocks_performance INT,
		missed_blocks_performance INT,
		number_active_vals INT,
		CONSTRAINT t_pool_summary_pkey PRIMARY KEY (f_pool_name, f_epoch));