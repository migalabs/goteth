package model

// Postgres intregration variables
var (
	CreateLastValidatorStatusTable = `
	CREATE TABLE IF NOT EXISTS t_validator_last_status(
		f_val_idx INT PRIMARY KEY,
		f_epoch INT,
		f_balance_eth REAL,
		f_status SMALLINT);`

	UpsertValidatorLastStatus = `
	INSERT INTO t_validator_last_status (	
		f_val_idx, 
		f_epoch, 
		f_balance_eth, 
		f_status)
	VALUES ($1, $2, $3, $4)
	ON CONFLICT ON CONSTRAINT t_validator_last_status_pkey
		DO 
			UPDATE SET 
				f_epoch = excluded.f_epoch, 
				f_balance_eth = excluded.f_balance_eth,
				f_status = excluded.f_status;
	`
)
