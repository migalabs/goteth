package db

import (
	"github.com/ClickHouse/ch-go/proto"
	"github.com/migalabs/goteth/pkg/spec"
)

var (
	withdrawalsTable       = "t_withdrawals"
	insertWithdrawalsQuery = `
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

func withdrawalsInput(withdrawals []spec.Withdrawal) proto.Input {
	// one object per column
	var (
		f_slot    proto.ColUInt64
		f_index   proto.ColUInt64
		f_val_idx proto.ColUInt64
		f_address proto.ColStr
		f_amount  proto.ColUInt64
	)

	for _, withdrawal := range withdrawals {

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

func (p *DBService) PersistWithdrawals(data []spec.Withdrawal) error {
	persistObj := PersistableObject[spec.Withdrawal]{
		input: withdrawalsInput,
		table: withdrawalsTable,
		query: insertWithdrawalsQuery,
	}

	for _, item := range data {
		persistObj.Append(item)
	}

	err := p.Persist(persistObj.ExportPersist())
	if err != nil {
		log.Errorf("error persisting withdrawals: %s", err.Error())
	}
	return err
}
