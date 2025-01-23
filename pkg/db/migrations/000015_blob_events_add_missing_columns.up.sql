ALTER TABLE t_blob_sidecars_events ADD COLUMN f_block_root TEXT;
ALTER TABLE t_blob_sidecars_events ADD COLUMN f_index UInt8;
ALTER TABLE t_blob_sidecars_events ADD COLUMN f_kzg_commitment TEXT;