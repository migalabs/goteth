UPDATE t_epoch_metrics_summary
SET f_num_vals = f_num_active_vals;

ALTER TABLE t_epoch_metrics_summary DROP COLUMN f_timestamp_vals;

ALTER TABLE t_epoch_metrics_summary DROP COLUMN f_num_slashed_vals;

ALTER TABLE t_epoch_metrics_summary DROP COLUMN f_num_active_vals;

ALTER TABLE t_epoch_metrics_summary DROP COLUMN f_num_exited_vals;

ALTER TABLE t_epoch_metrics_summary DROP COLUMN f_num_in_activation_vals;

