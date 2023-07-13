package db

import (
	"github.com/cortze/eth-cl-state-analyzer/pkg/spec"
)

// Postgres intregration variables
var (
	UpsertValidatorLastStatus = `
	INSERT INTO t_validator_last_status (	
		f_val_idx, 
		f_epoch, 
		f_balance_eth, 
		f_status,
		f_slashed,
		f_activation_epoch,
		f_withdrawal_epoch,
		f_exit_epoch,
		f_public_key)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	ON CONFLICT ON CONSTRAINT t_validator_last_status_pkey
		DO 
			UPDATE SET 
				f_epoch = excluded.f_epoch, 
				f_balance_eth = excluded.f_balance_eth,
				f_status = excluded.f_status,
				f_slashed = excluded.f_slashed,
				f_activation_epoch = excluded.f_activation_epoch,
				f_withdrawal_epoch = excluded.f_withdrawal_epoch,
				f_exit_epoch = excluded.f_exit_epoch,
				f_public_key = excluded.f_public_key;
	`
)

func insertValidatorLastStatus(inputValidator spec.ValidatorLastStatus) (string, []interface{}) {
	resultArgs := make([]interface{}, 0)
	resultArgs = append(resultArgs, inputValidator.ValIdx)
	resultArgs = append(resultArgs, inputValidator.Epoch)
	resultArgs = append(resultArgs, inputValidator.BalanceToEth())
	resultArgs = append(resultArgs, inputValidator.CurrentStatus)
	resultArgs = append(resultArgs, inputValidator.Slashed)
	resultArgs = append(resultArgs, inputValidator.ActivationEpoch)
	resultArgs = append(resultArgs, inputValidator.WithdrawalEpoch)
	resultArgs = append(resultArgs, inputValidator.ExitEpoch)
	resultArgs = append(resultArgs, inputValidator.PublicKey.String())

	return UpsertValidatorLastStatus, resultArgs
}

func ValidatorLastStatusOperation(inputValidator spec.ValidatorLastStatus) (string, []interface{}) {

	q, args := insertValidatorLastStatus(inputValidator)
	return q, args

}
