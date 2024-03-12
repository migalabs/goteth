DROP TABLE IF EXISTS t_blob_sidecars;
DROP TABLE IF EXISTS t_blob_sidecars_events;

ALTER TABLE t_transactions DROP COLUMN f_blob_gas_used;
ALTER TABLE t_transactions DROP COLUMN f_blob_gas_price;
ALTER TABLE t_transactions DROP COLUMN f_blob_gas_limit;
ALTER TABLE t_transactions DROP COLUMN f_blob_gas_fee_cap;