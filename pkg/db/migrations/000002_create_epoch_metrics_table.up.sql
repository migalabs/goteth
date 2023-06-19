CREATE TABLE IF NOT EXISTS t_epoch_metrics_summary(
	f_epoch INT,
	f_slot INT,
	f_num_att INT,
	f_num_att_vals INT,
	f_num_vals INT,
	f_total_balance_eth REAL,
	f_att_effective_balance_eth REAL,
	f_total_effective_balance_eth REAL,
	f_missing_source INT, 
	f_missing_target INT,
	f_missing_head INT,
	CONSTRAINT PK_Epoch PRIMARY KEY (f_slot));