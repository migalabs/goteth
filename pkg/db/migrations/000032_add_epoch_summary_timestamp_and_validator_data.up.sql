ALTER TABLE t_epoch_metrics_summary ADD COLUMN f_timestamp BIGINT;

CREATE OR REPLACE FUNCTION calculate_default_timestamp()
  RETURNS TRIGGER AS
$$
BEGIN
  NEW.f_timestamp := (SELECT f_genesis_time FROM t_genesis LIMIT 1) + (NEW.f_epoch * 32 * 12);
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;



CREATE TRIGGER set_default_timestamp
  BEFORE INSERT ON t_epoch_metrics_summary
  FOR EACH ROW
  EXECUTE FUNCTION calculate_default_timestamp();

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