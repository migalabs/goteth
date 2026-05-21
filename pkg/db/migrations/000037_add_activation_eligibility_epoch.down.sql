-- Remove f_activation_eligibility_epoch column from t_validator_last_status.
ALTER TABLE t_validator_last_status
DROP COLUMN f_activation_eligibility_epoch;
