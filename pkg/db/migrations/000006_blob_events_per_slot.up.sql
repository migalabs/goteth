CREATE TABLE IF NOT EXISTS t_blob_sidecars_events_copy(
	f_arrival_timestamp_ms UInt64,
	f_blob_hash TEXT DEFAULT '',
	f_slot UInt64)
	ENGINE = ReplacingMergeTree()
	ORDER BY (f_arrival_timestamp_ms, f_blob_hash, f_slot);

INSERT INTO t_blob_sidecars_events_copy(f_arrival_timestamp_ms, f_blob_hash)
	SELECT *
	FROM t_blob_sidecars_events;

DROP TABLE t_blob_sidecars_events;
RENAME TABLE t_blob_sidecars_events_copy TO t_blob_sidecars_events;