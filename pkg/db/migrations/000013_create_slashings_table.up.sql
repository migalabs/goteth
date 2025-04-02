CREATE TABLE t_slashings(
	f_slashed_validator_index UInt64,
	f_slashed_by_validator_index UInt64,
	f_slashing_reason TEXT,
	f_slot UInt64,
	f_epoch UInt64
	)
	ENGINE = ReplacingMergeTree()
	ORDER BY (f_slot, f_slashed_validator_index);