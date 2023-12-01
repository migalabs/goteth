UPDATE t_epoch_metrics_summary
SET f_num_vals = f_num_active;

ALTER TABLE t_epoch_metrics_summary DROP COLUMN f_timestamp;

ALTER TABLE t_epoch_metrics_summary DROP COLUMN f_num_slashed;

ALTER TABLE t_epoch_metrics_summary DROP COLUMN f_num_active;

ALTER TABLE t_epoch_metrics_summary DROP COLUMN f_num_exit;

ALTER TABLE t_epoch_metrics_summary DROP COLUMN f_num_in_activation;

