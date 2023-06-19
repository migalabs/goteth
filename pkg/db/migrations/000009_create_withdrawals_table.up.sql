CREATE TABLE IF NOT EXISTS t_withdrawals(
		f_slot INT,
		f_index INT,
		f_val_idx INT,
		f_address TEXT,
		f_amount BIGINT,
		CONSTRAINT PK_Withdrawal PRIMARY KEY (f_slot, f_index));