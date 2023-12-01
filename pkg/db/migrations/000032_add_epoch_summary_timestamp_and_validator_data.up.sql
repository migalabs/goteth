ALTER TABLE t_epoch_metrics_summary ADD COLUMN f_timestamp BIGINT;

UPDATE t_epoch_metrics_summary
SET f_timestamp = (SELECT f_genesis_time FROM t_genesis LIMIT 1) + (f_epoch * 32 * 12);

ALTER TABLE t_epoch_metrics_summary
ADD COLUMN f_num_slashed INTEGER DEFAULT 0;

ALTER TABLE t_epoch_metrics_summary
ADD COLUMN f_num_active INTEGER DEFAULT 0;

ALTER TABLE t_epoch_metrics_summary
ADD COLUMN f_num_exit INTEGER DEFAULT 0;

ALTER TABLE t_epoch_metrics_summary
ADD COLUMN f_num_in_activation INTEGER DEFAULT 0;

UPDATE t_epoch_metrics_summary
SET f_num_active = f_num_vals,
    f_num_vals = 0;
