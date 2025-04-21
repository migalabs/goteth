package db

import (
	"github.com/ClickHouse/ch-go/proto"
	"github.com/migalabs/goteth/pkg/spec"
)

var (
	withdrawalRequestsTable      = "t_withdrawal_requests"
	insertWithdrawalRequestQuery = `
	INSERT INTO %s (
		f_slot,
		f_index,
		f_source_address,
		f_validator_pubkey,
		f_amount,
		f_result
		)
		VALUES`
)

func withdrawalRequestsInput(withdrawalRequests []spec.WithdrawalRequest) proto.Input {
	// one object per column
	var (
		f_slot             proto.ColUInt64
		f_index            proto.ColUInt64
		f_source_address   proto.ColStr
		f_validator_pubkey proto.ColStr
		f_amount           proto.ColUInt64
		f_result           proto.ColUInt8
	)

	for _, withdrawalRequest := range withdrawalRequests {

		f_slot.Append(uint64(withdrawalRequest.Slot))
		f_index.Append(withdrawalRequest.Index)
		f_source_address.Append(withdrawalRequest.SourceAddress.String())
		f_validator_pubkey.Append(withdrawalRequest.ValidatorPubkey.String())
		f_amount.Append(uint64(withdrawalRequest.Amount))
		f_result.Append(uint8(withdrawalRequest.Result))
	}

	return proto.Input{
		{Name: "f_slot", Data: f_slot},
		{Name: "f_index", Data: f_index},
		{Name: "f_source_address", Data: f_source_address},
		{Name: "f_validator_pubkey", Data: f_validator_pubkey},
		{Name: "f_amount", Data: f_amount},
		{Name: "f_result", Data: f_result},
	}
}

func (p *DBService) PersistWithdrawalRequests(data []spec.WithdrawalRequest) error {
	persistObj := PersistableObject[spec.WithdrawalRequest]{
		input: withdrawalRequestsInput,
		table: withdrawalRequestsTable,
		query: insertWithdrawalRequestQuery,
	}

	for _, item := range data {
		persistObj.Append(item)
	}

	err := p.Persist(persistObj.ExportPersist())
	if err != nil {
		log.Errorf("error persisting withdrawalRequests: %s", err.Error())
	}
	return err
}
