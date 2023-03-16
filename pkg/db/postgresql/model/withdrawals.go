package model

// Postgres intregration variables
var (
	CreateWithdrawalsTable = `
	CREATE TABLE IF NOT EXISTS t_withdrawals(
		f_slot INT,
		f_index INT,
		f_val_idx INT,
		f_address TEXT,
		f_amount BIGINT,
		CONSTRAINT PK_Withdrawal PRIMARY KEY (f_slot, f_index));`

	UpsertWithdrawal = `
	INSERT INTO t_withdrawals (
		f_slot,
		f_index, 
		f_val_idx,
		f_address,
		f_amount)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT ON CONSTRAINT PK_Withdrawal
		DO
		UPDATE SET 
			f_val_idx = excluded.f_val_idx,
			f_address = excluded.f_address,
			f_amount = excluded.f_amount;
	`
)
