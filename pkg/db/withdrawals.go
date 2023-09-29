package db

import (
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/migalabs/goteth/pkg/spec"
)

// Postgres intregration variables
var (
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

	DropWithdrawalsQuery = `
		DELETE FROM t_withdrawals
		WHERE f_slot >= $1;
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

type WithdrawalDropType phase0.Slot

func (s WithdrawalDropType) Type() spec.ModelType {
	return spec.WithdrawalDropModel
}

func DropWitdrawals(slot WithdrawalDropType) (string, []interface{}) {
	resultArgs := make([]interface{}, 0)
	resultArgs = append(resultArgs, slot)
	return DropWithdrawalsQuery, resultArgs
}
