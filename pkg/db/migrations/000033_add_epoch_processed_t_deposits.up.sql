-- Create new table with f_epoch_processed column
CREATE TABLE new_t_deposits(
	f_slot UInt64,
	f_epoch_processed UInt64,
	f_public_key TEXT,
	f_withdrawal_credentials TEXT,
	f_amount UInt64,
	f_signature TEXT,
	f_index UInt8
	)
	ENGINE = ReplacingMergeTree()
ORDER BY (f_slot, f_epoch_processed, f_index);

-- Migrate data from old t_deposits to new_t_deposits
INSERT INTO new_t_deposits
SELECT
	f_slot,
	CAST(f_slot / 32 AS UInt64) AS f_epoch_processed,
	f_public_key,
	f_withdrawal_credentials,
	f_amount,
	f_signature,
	f_index
FROM t_deposits;

-- Rename tables
RENAME TABLE t_deposits TO t_deposits_old;
RENAME TABLE new_t_deposits TO t_deposits;

-- Drop temporary table
DROP TABLE t_deposits_old;