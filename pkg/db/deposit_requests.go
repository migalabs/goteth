package db

import (
	"fmt"

	"github.com/ClickHouse/ch-go/proto"
	"github.com/migalabs/goteth/pkg/spec"
)

var (
	depositRequestsTable      = "t_deposit_requests"
	insertDepositRequestQuery = `
	INSERT INTO %s (
		f_slot,
		f_pubkey,
		f_withdrawal_credentials,
		f_amount,
		f_signature,
		f_index
		)
		VALUES`
)

func depositRequestsInput(depositRequests []spec.DepositRequest) proto.Input {
	// one object per column
	var (
		f_slot                   proto.ColUInt64
		f_pubkey                 proto.ColStr
		f_withdrawal_credentials proto.ColStr
		f_amount                 proto.ColUInt64
		f_signature              proto.ColStr
		f_index                  proto.ColUInt64
	)

	for _, depositRequest := range depositRequests {
		f_slot.Append(uint64(depositRequest.Slot))
		f_pubkey.Append(depositRequest.Pubkey.String())
		f_withdrawal_credentials.Append(fmt.Sprintf("0x%x", depositRequest.WithdrawalCredentials))
		f_amount.Append(uint64(depositRequest.Amount))
		f_signature.Append(depositRequest.Signature.String())
		f_index.Append(uint64(depositRequest.Index))
	}

	return proto.Input{
		{Name: "f_slot", Data: f_slot},
		{Name: "f_pubkey", Data: f_pubkey},
		{Name: "f_withdrawal_credentials", Data: f_withdrawal_credentials},
		{Name: "f_amount", Data: f_amount},
		{Name: "f_signature", Data: f_signature},
		{Name: "f_index", Data: f_index},
	}
}

func (p *DBService) PersistDepositRequests(data []spec.DepositRequest) error {
	persistObj := PersistableObject[spec.DepositRequest]{
		input: depositRequestsInput,
		table: depositRequestsTable,
		query: insertDepositRequestQuery,
	}

	for _, item := range data {
		persistObj.Append(item)
	}

	err := p.Persist(persistObj.ExportPersist())
	if err != nil {
		log.Errorf("error persisting depositRequests: %s", err.Error())
	}
	return err
}
