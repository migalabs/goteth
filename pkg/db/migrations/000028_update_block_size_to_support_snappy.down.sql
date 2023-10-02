-- rollback rename of the payload size
ALTER TABLE t_block_metrics RENAME COLUMN f_payload_size_bytes TO f_size_bytes;
-- remove the extra snappy columns
ALTER TABLE t_block_metrics DROP COLUMN IF EXISTS f_ssz_size_bytes;
ALTER TABLE t_block_metrics DROP COLUMN IF EXISTS f_snappy_size_bytes;
ALTER TABLE t_block_metrics DROP COLUMN IF EXISTS f_compression_time_ms;
ALTER TABLE t_block_metrics DROP COLUMN IF EXISTS f_decompression_time_ms;