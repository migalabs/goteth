package db

import (
	"fmt"

	"github.com/ClickHouse/ch-go/proto"
	"github.com/migalabs/goteth/pkg/spec"
)

var (
	depositsTable      = "t_deposits"
	insertDepositQuery = `
	INSERT INTO %s (
		f_slot,
		f_public_key,
		f_withdrawal_credentials,
		f_amount,
		f_signature,
		f_index
		)
		VALUES`
)

func depositsInput(depositss []spec.Deposit) proto.Input {
	// one object per column
	var (
		f_slot                   proto.ColUInt64
		f_public_key             proto.ColStr
		f_withdrawal_credentials proto.ColStr
		f_amount                 proto.ColUInt64
		f_signature              proto.ColStr
		f_index                  proto.ColUInt8
	)

	for _, deposit := range depositss {

		f_slot.Append(uint64(deposit.Slot))
		f_public_key.Append(deposit.PublicKey.String())
		f_withdrawal_credentials.Append(fmt.Sprintf("%#x", deposit.WithdrawalCredentials))
		f_amount.Append(uint64(deposit.Amount))
		f_signature.Append(deposit.Signature.String())
		f_index.Append(uint8(deposit.Index))
	}

	return proto.Input{
		{Name: "f_slot", Data: f_slot},
		{Name: "f_public_key", Data: f_public_key},
		{Name: "f_withdrawal_credentials", Data: f_withdrawal_credentials},
		{Name: "f_amount", Data: f_amount},
		{Name: "f_signature", Data: f_signature},
		{Name: "f_index", Data: f_index},
	}
}

func (p *DBService) PersistDeposits(data []spec.Deposit) error {
	persistObj := PersistableObject[spec.Deposit]{
		input: depositsInput,
		table: depositsTable,
		query: insertDepositQuery,
	}

	for _, item := range data {
		persistObj.Append(item)
	}

	err := p.Persist(persistObj.ExportPersist())
	if err != nil {
		log.Errorf("error persisting deposits: %s", err.Error())
	}
	return err
}
