package db

import (
	"github.com/ClickHouse/ch-go/proto"
	"github.com/migalabs/goteth/pkg/spec"
)

var (
	eth1DepositsTable       = "t_eth1Deposits"
	insertETH1DepositsQuery = `
		INSERT INTO %s(
			f_block_number,
			f_block_hash,
			f_block_timestamp,
			f_tx_hash,
			f_log_index,
			f_sender,
			f_recipient,
			f_gas_used,
			f_gas_price,
			f_deposit_index,
			f_validator_pubkey,
			f_withdrawal_credentials
			f_signature,
			f_amount)
		VALUES`
)

func eth1DepositsInput(eth1Deposits []spec.ETH1Deposit) proto.Input {
	// one object per column
	var (
		f_block_number           proto.ColUInt64
		f_block_hash             proto.ColStr
		f_block_timestamp        proto.ColUInt64
		f_tx_hash                proto.ColStr
		f_log_index              proto.ColUInt64
		f_sender                 proto.ColStr
		f_recipient              proto.ColStr
		f_gas_used               proto.ColUInt64
		f_gas_price              proto.ColUInt64
		f_deposit_index          proto.ColUInt64
		f_validator_pubkey       proto.ColStr
		f_withdrawal_credentials proto.ColStr
		f_signature              proto.ColStr
		f_amount                 proto.ColUInt64
	)

	for _, eth1Deposit := range eth1Deposits {
		f_block_number.Append(uint64(eth1Deposit.BlockNumber))
		f_block_hash.Append(eth1Deposit.BlockHash)
		f_block_timestamp.Append(uint64(eth1Deposit.BlockTimestamp))
		f_tx_hash.Append(eth1Deposit.TxHash)
		f_log_index.Append(uint64(eth1Deposit.LogIndex))
		f_sender.Append(eth1Deposit.Sender)
		f_recipient.Append(eth1Deposit.Recipient)
		f_gas_used.Append(uint64(eth1Deposit.GasUsed))
		f_gas_price.Append(uint64(eth1Deposit.GasPrice))
		f_deposit_index.Append(uint64(eth1Deposit.DepositIndex))
		f_validator_pubkey.Append(eth1Deposit.ValidatorPubkey)
		f_withdrawal_credentials.Append(eth1Deposit.WithdrawalCredentials)
		f_signature.Append(eth1Deposit.Signature)
		f_amount.Append(uint64(eth1Deposit.Amount))
	}

	return proto.Input{
		{Name: "f_block_number", Data: f_block_number},
		{Name: "f_block_hash", Data: f_block_hash},
		{Name: "f_block_timestamp", Data: f_block_timestamp},
		{Name: "f_tx_hash", Data: f_tx_hash},
		{Name: "f_log_index", Data: f_log_index},
		{Name: "f_sender", Data: f_sender},
		{Name: "f_recipient", Data: f_recipient},
		{Name: "f_gas_used", Data: f_gas_used},
		{Name: "f_gas_price", Data: f_gas_price},
		{Name: "f_deposit_index", Data: f_deposit_index},
		{Name: "f_validator_pubkey", Data: f_validator_pubkey},
		{Name: "f_withdrawal_credentials", Data: f_withdrawal_credentials},
		{Name: "f_signature", Data: f_signature},
		{Name: "f_amount", Data: f_amount},
	}
}

func (p *DBService) PersistETH1Deposits(data []spec.ETH1Deposit) error {
	persistObj := PersistableObject[spec.ETH1Deposit]{
		input: eth1DepositsInput,
		table: eth1DepositsTable,
		query: insertETH1DepositsQuery,
	}

	for _, item := range data {
		persistObj.Append(item)
	}

	err := p.Persist(persistObj.ExportPersist())
	if err != nil {
		log.Errorf("error persisting eth1Deposits: %s", err.Error())
	}
	return err
}
