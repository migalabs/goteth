-- PeerDAS (Fulu) replaced blob-level p2p propagation with column-level propagation:
-- a node no longer receives whole blobs, it receives DataColumnSidecars. The
-- beacon API's `blob_sidecar` event stopped firing at the Fulu fork, so
-- t_blob_sidecars_events silently stopped filling (last row 2025-12-03) while
-- every other blob table kept working. This table is its replacement: one row per
-- column arrival, which is the finest-grained propagation signal available now
-- (and strictly richer than the old one: it exposes the reconstruction curve of
-- each blob, not just a single arrival timestamp).
--
-- Note: blob *metadata* still comes from the block itself (t_blob_sidecars) and is
-- unaffected. Only the p2p arrival timing moved here.
CREATE TABLE IF NOT EXISTS t_data_column_sidecars_events (
	f_arrival_timestamp_ms UInt64 CODEC(DoubleDelta, ZSTD(3)),
	f_slot                 UInt64 CODEC(DoubleDelta, ZSTD(3)),
	f_block_root           String CODEC(ZSTD(3)),
	f_column_index         UInt16 CODEC(T64, ZSTD(3)),
	f_kzg_commitments      Array(String) CODEC(ZSTD(3)),
	f_num_commitments      UInt16 CODEC(T64, ZSTD(3))
	)
	ENGINE = ReplacingMergeTree()
	PARTITION BY intDiv(f_slot, 100000)
	ORDER BY (f_slot, f_column_index, f_block_root);

-- The blob_sidecar event no longer exists post-Fulu, so this table can never
-- receive another row. Its historical rows (pre-2025-12-03) are preserved by
-- renaming rather than dropping.
--
-- Deploy note: any bindeth serving this database must already include bindeth
-- PR #352 (which re-points the blob queries to t_data_column_sidecars_events),
-- otherwise its blob endpoints break on the renamed table.
RENAME TABLE t_blob_sidecars_events TO t_blob_sidecars_events_pre_fulu;
