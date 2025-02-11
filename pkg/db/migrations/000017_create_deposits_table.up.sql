CREATE TABLE t_deposits(
	f_slot UInt64,
	f_public_key TEXT,
	f_withdrawal_credentials TEXT,
	f_amount UInt64,
	f_signature TEXT,
	f_index UInt8,
	)
	ENGINE = ReplacingMergeTree()
	ORDER BY (f_slot, f_index);