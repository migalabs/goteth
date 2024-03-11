CREATE TABLE IF NOT EXISTS t_blob_sidecars(
	f_blob_hash TEXT DEFAULT '',
	f_tx_hash TEXT DEFAULT '',
	f_slot UInt64,
	f_index UInt8,
	f_kzg_commitment TEXT DEFAULT '',
	f_kzg_proof TEXT DEFAULT '')
	ENGINE = ReplacingMergeTree()
	ORDER BY (f_slot, f_index);

CREATE TABLE IF NOT EXISTS t_blob_sidecars_events(
	f_arrival_timestamp_ms UInt64,
	f_blob_hash TEXT DEFAULT '')
	ENGINE = ReplacingMergeTree()
	ORDER BY (f_blob_hash);

ALTER TABLE t_transactions ADD COLUMN f_blob_gas_used UInt64;
ALTER TABLE t_transactions ADD COLUMN f_blob_gas_price UInt64;
ALTER TABLE t_transactions ADD COLUMN f_blob_gas_limit UInt64;
ALTER TABLE t_transactions ADD COLUMN f_blob_gas_fee_cap UInt64;