package db

import (
	"github.com/cortze/eth-cl-state-analyzer/pkg/spec"
	"github.com/pkg/errors"
)

// Postgres intregration variables
var (
	CreateWithdrawalsTable = `
	CREATE TABLE IF NOT EXISTS t_withdrawals(
		f_slot INT,
		f_index INT,
		f_val_idx INT,
		f_address TEXT,
		f_amount BIGINT,
		CONSTRAINT PK_Withdrawal PRIMARY KEY (f_slot, f_index));`

	UpsertWithdrawal = `
	INSERT INTO t_withdrawals (
		f_slot,
		f_index, 
		f_val_idx,
		f_address,
		f_amount)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT ON CONSTRAINT PK_Withdrawal
		DO
		UPDATE SET 
			f_val_idx = excluded.f_val_idx,
			f_address = excluded.f_address,
			f_amount = excluded.f_amount;
	`
)

func insertWithdrawal(inputWithdrawal spec.Withdrawal) (string, []interface{}) {
	resultArgs := make([]interface{}, 0)
	resultArgs = append(resultArgs, inputWithdrawal.Slot)
	resultArgs = append(resultArgs, inputWithdrawal.Index)
	resultArgs = append(resultArgs, inputWithdrawal.ValidatorIndex)
	resultArgs = append(resultArgs, inputWithdrawal.Address)
	resultArgs = append(resultArgs, inputWithdrawal.Amount)

	return UpsertWithdrawal, resultArgs
}

func WithdrawalOperation(inputWithdrawal spec.Withdrawal) (string, []interface{}) {

	q, args := insertWithdrawal(inputWithdrawal)
	return q, args
}

func (p *PostgresDBService) createWithdrawalsTable() error {
	// create the tables
	_, err := p.psqlPool.Exec(p.ctx, CreateWithdrawalsTable)
	if err != nil {
		return errors.Wrap(err, "error creating withdrawals table")
	}
	return nil
}
