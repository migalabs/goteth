package postgresql

import (
	"fmt"

	"github.com/cortze/eth-cl-state-analyzer/pkg/db/model"
	"github.com/pkg/errors"
)

// Postgres intregration variables
var (
	CreateLastValidatorStatusTable = `
	CREATE TABLE IF NOT EXISTS t_validator_last_status(
		f_val_idx INT PRIMARY KEY,
		f_epoch INT,
		f_balance_eth REAL,
		f_status SMALLINT);`

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

func insertValidatorLastStatus(inputValidator model.ValidatorLastStatus) (string, []interface{}) {
	resultArgs := make([]interface{}, 0)
	resultArgs = append(resultArgs, inputValidator.ValIdx)
	resultArgs = append(resultArgs, inputValidator.Epoch)
	resultArgs = append(resultArgs, inputValidator.BalanceToEth())
	resultArgs = append(resultArgs, inputValidator.CurrentStatus)

	return UpsertValidatorLastStatus, resultArgs
}

func ValidatorLastStatusOperation(inputValidator model.ValidatorLastStatus, op string) (string, []interface{}, error) {

	if op == model.INSERT_OP {
		q, args := insertValidatorLastStatus(inputValidator)
		return q, args, nil
	}

	return "", nil, fmt.Errorf("validator last status operation not permitted: %s", op)
}

func (p *PostgresDBService) createLastStatusTable() error {
	// create the tables
	_, err := p.psqlPool.Exec(p.ctx, CreateLastValidatorStatusTable)
	if err != nil {
		return errors.Wrap(err, "error creating validator last status table")
	}
	return nil
}
