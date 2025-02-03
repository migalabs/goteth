CREATE TABLE t_bls_to_execution_changes(
	f_slot UInt64,
	f_epoch UInt64,
	f_validator_index UInt64,
	f_from_bls_pubkey TEXT,
	f_to_execution_address TEXT
	)
	ENGINE = ReplacingMergeTree()
	ORDER BY (f_slot, f_epoch, f_validator_index);