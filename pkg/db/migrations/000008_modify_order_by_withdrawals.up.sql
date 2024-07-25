-- Create new table with desired order by
CREATE TABLE new_t_withdrawals(
	f_slot UInt64,
	f_index UInt64,
	f_val_idx UInt64,
	f_address TEXT,
	f_amount UInt64)
	ENGINE = ReplacingMergeTree()
	ORDER BY (f_index);

-- Populate new table
INSERT INTO new_t_withdrawals
SELECT * FROM t_withdrawals;

-- Rename tables
RENAME TABLE t_withdrawals TO t_withdrawals_old;
RENAME TABLE new_t_withdrawals TO t_withdrawals;

-- Drop temporary table
DROP TABLE t_withdrawals_old;