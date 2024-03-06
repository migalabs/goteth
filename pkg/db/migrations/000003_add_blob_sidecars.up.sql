CREATE TABLE IF NOT EXISTS t_blob_sidecars(
	f_arrival_timestamp_ms UInt64,
	f_blob_hash TEXT DEFAULT '',
	f_slot UInt64,
	f_index UInt8,
	f_kzg_commitment TEXT DEFAULT '',
	f_kzg_proof TEXT DEFAULT '')
	ENGINE = ReplacingMergeTree()
	ORDER BY (f_slot, f_index);