ALTER TABLE t_validator_last_status
ADD COLUMN f_slashed bool,
ADD COLUMN f_activation_epoch numeric(21, 0),
ADD COLUMN f_withdrawal_epoch numeric(21, 0),
ADD COLUMN f_exit_epoch numeric(21, 0),
ADD COLUMN f_public_key text;