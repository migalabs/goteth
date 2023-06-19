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
		f_status)
	VALUES ($1, $2, $3, $4)
	ON CONFLICT ON CONSTRAINT t_validator_last_status_pkey
		DO 
			UPDATE SET 
				f_epoch = excluded.f_epoch, 
				f_balance_eth = excluded.f_balance_eth,
				f_status = excluded.f_status;
	`
)

func insertValidatorLastStatus(inputValidator spec.ValidatorLastStatus) (string, []interface{}) {
	resultArgs := make([]interface{}, 0)
	resultArgs = append(resultArgs, inputValidator.ValIdx)
	resultArgs = append(resultArgs, inputValidator.Epoch)
	resultArgs = append(resultArgs, inputValidator.BalanceToEth())
	resultArgs = append(resultArgs, inputValidator.CurrentStatus)

	return UpsertValidatorLastStatus, resultArgs
}

func ValidatorLastStatusOperation(inputValidator spec.ValidatorLastStatus) (string, []interface{}) {

	q, args := insertValidatorLastStatus(inputValidator)
	return q, args

}
