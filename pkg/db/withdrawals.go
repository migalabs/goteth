package db

import (
	"fmt"

	"github.com/ClickHouse/ch-go/proto"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/migalabs/goteth/pkg/spec"
)

// Postgres intregration variables
var (
	withdrawalsTable  = "t_withdrawals"
	insertWithdrawals = `
	INSERT INTO %s (
		f_slot,
		f_index, 
		f_val_idx,
		f_address,
		f_amount)
		VALUES`

	deleteWithdrawalsQuery = `
		DELETE FROM %s
		WHERE f_slot = $1;`
)

type InsertWithdrawals struct {
	withdrawals []spec.Withdrawal
}

func (d InsertWithdrawals) Table() string {
	return withdrawalsTable
}

func (d *InsertWithdrawals) Append(newWithdrawal spec.Withdrawal) {
	d.withdrawals = append(d.withdrawals, newWithdrawal)
}

func (d InsertWithdrawals) Columns() int {
	return len(d.Input().Columns())
}

func (d InsertWithdrawals) Rows() int {
	return len(d.withdrawals)
}

func (d InsertWithdrawals) Query() string {
	return fmt.Sprintf(insertWithdrawals, withdrawalsTable)
}
func (d InsertWithdrawals) Input() proto.Input {
	// one object per column
	var (
		f_slot    proto.ColUInt64
		f_index   proto.ColUInt64
		f_val_idx proto.ColUInt64
		f_address proto.ColStr
		f_amount  proto.ColUInt64
	)

	for _, withdrawal := range d.withdrawals {

		f_slot.Append(uint64(withdrawal.Slot))
		f_index.Append(uint64(withdrawal.Index))
		f_val_idx.Append(uint64(withdrawal.ValidatorIndex))
		f_address.Append(withdrawal.Address.String())
		f_amount.Append(uint64(withdrawal.Amount))
	}

	return proto.Input{

		{Name: "f_slot", Data: f_slot},
		{Name: "f_index", Data: f_index},
		{Name: "f_val_idx", Data: f_val_idx},
		{Name: "f_address", Data: f_address},
		{Name: "f_amount", Data: f_amount},
	}
}

type DeleteWithdrawals struct {
	slot phase0.Slot
}

func (d DeleteWithdrawals) Query() string {
	return fmt.Sprintf(deleteWithdrawalsQuery, withdrawalsTable)
}

func (d DeleteWithdrawals) Table() string {
	return withdrawalsTable
}

func (d DeleteWithdrawals) Args() []any {
	return []any{d.slot}
}
