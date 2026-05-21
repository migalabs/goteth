-- Add f_activation_eligibility_epoch column to t_validator_last_status.
-- Default is FAR_FUTURE_EPOCH (2^64 - 1) so historical rows that pre-date
-- this migration read as "not yet eligible" — which matches the consensus
-- spec semantics for an unset eligibility epoch. goteth rewrites every row
-- on each finalized epoch, so the default is corrected for live validators
-- on the next analyzer pass.
ALTER TABLE t_validator_last_status
ADD COLUMN f_activation_eligibility_epoch UInt64 DEFAULT 18446744073709551615 AFTER f_activation_epoch;
