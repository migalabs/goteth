ALTER TABLE t_validator_last_status
DROP COLUMN IF EXISTS f_slashed,
DROP COLUMN IF EXISTS f_activation_epoch,
DROP COLUMN IF EXISTS f_withdrawal_epoch,
DROP COLUMN IF EXISTS f_exit_epoch,
DROP COLUMN IF EXISTS f_public_key;