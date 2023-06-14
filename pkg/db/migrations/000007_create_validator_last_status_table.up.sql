CREATE TABLE IF NOT EXISTS t_validator_last_status(
		f_val_idx INT PRIMARY KEY,
		f_epoch INT,
		f_balance_eth REAL,
		f_status SMALLINT);