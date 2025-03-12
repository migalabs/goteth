ALTER TABLE t_epoch_metrics_summary ADD COLUMN f_consolidation_requests_num UInt64;

ALTER TABLE t_block_metrics ADD COLUMN f_consolidation_requests_num UInt64 AFTER f_deposits;

