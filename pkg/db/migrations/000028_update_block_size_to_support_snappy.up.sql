-- rename the previous block size to specify that it's only the payload size
ALTER TABLE t_block_metrics RENAME COLUMN f_size_bytes TO f_payload_size_bytes;
-- add snappy compression fields
ALTER TABLE t_block_metrics ADD COLUMN f_ssz_size_bytes NUMERIC;
ALTER TABLE t_block_metrics ADD COLUMN f_snappy_size_bytes NUMERIC;
ALTER TABLE t_block_metrics ADD COLUMN f_compression_time_ms REAL;
ALTER TABLE t_block_metrics ADD COLUMN f_decompression_time_ms REAL;