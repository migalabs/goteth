CREATE TABLE IF NOT EXISTS t_pool_summary(
		f_pool_name TEXT,
		f_epoch INT,
		f_reward INT,
		f_max_reward INT,
		f_max_att_reward INT,
		f_max_sync_reward INT,
		f_base_reward INT,
		f_sum_missing_source INT,
		f_sum_missing_target INT, 
		f_sum_missing_head INT,
		f_num_active_vals INT,
		f_sync_vals INT,
		CONSTRAINT PK_EpochPool PRIMARY KEY (f_pool_name,f_epoch));