package db

import (
	"fmt"

	"github.com/ClickHouse/ch-go/proto"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/migalabs/goteth/pkg/spec"
)

var (
	transactionsTable       = "t_transactions"
	insertTransactionsQuery = `
		INSERT INTO %s(
			f_tx_type, f_chain_id, f_data, f_gas, f_gas_price, f_gas_tip_cap, f_gas_fee_cap, f_value, f_nonce, f_to, f_hash,
								f_size, f_slot, f_el_block_number, f_timestamp, f_from, f_contract_address)
		VALUES`

	deleteTransactionsQuery = `
		DELETE FROM t_transactions
		WHERE f_slot = $1;
`
)

type InsertTransactions struct {
	transactions []spec.AgnosticTransaction
}

func (d InsertTransactions) Table() string {
	return transactionsTable
}

func (d *InsertTransactions) Append(newtransaction spec.AgnosticTransaction) {
	d.transactions = append(d.transactions, newtransaction)
}

func (d InsertTransactions) Columns() int {
	return len(d.Input().Columns())
}

func (d InsertTransactions) Rows() int {
	return len(d.transactions)
}

func (d InsertTransactions) Query() string {
	return fmt.Sprintf(insertTransactionsQuery, transactionsTable)
}
func (d InsertTransactions) Input() proto.Input {
	// one object per column
	var (
		f_tx_type          proto.ColUInt64
		f_chain_id         proto.ColUInt64
		f_data             proto.ColStr
		f_gas              proto.ColUInt64
		f_gas_price        proto.ColUInt64
		f_gas_tip_cap      proto.ColUInt64
		f_gas_fee_cap      proto.ColUInt64
		f_value            proto.ColFloat32
		f_nonce            proto.ColUInt64
		f_to               proto.ColStr
		f_hash             proto.ColStr
		f_size             proto.ColUInt64
		f_slot             proto.ColUInt64
		f_el_block_number  proto.ColUInt64
		f_timestamp        proto.ColUInt64
		f_from             proto.ColStr
		f_contract_address proto.ColStr
	)

	for _, transaction := range d.transactions {

		f_tx_type.Append(uint64(transaction.TxType))
		f_chain_id.Append(uint64(transaction.ChainId))
		f_data.Append(transaction.Data)
		f_gas.Append(uint64(transaction.Gas))
		f_gas_price.Append(uint64(transaction.GasPrice))
		f_gas_tip_cap.Append(uint64(transaction.GasTipCap))
		f_gas_fee_cap.Append(uint64(transaction.GasFeeCap))
		f_value.Append(float32(transaction.Value))
		f_nonce.Append(transaction.Nonce)
		// to sometimes is empty or nil
		tx := ""
		if transaction.To != nil {
			tx = transaction.To.String()
		}
		f_to.Append(tx)
		f_hash.Append(transaction.Hash.String())
		f_size.Append(transaction.Size)
		f_slot.Append(uint64(transaction.Slot))
		f_el_block_number.Append(transaction.BlockNumber)
		f_timestamp.Append(uint64(transaction.Timestamp))
		f_from.Append(transaction.From.String())
		f_contract_address.Append(transaction.ContractAddress.String())
	}

	return proto.Input{

		{Name: "f_tx_type", Data: f_tx_type},
		{Name: "f_chain_id", Data: f_chain_id},
		{Name: "f_data", Data: f_data},
		{Name: "f_gas", Data: f_gas},
		{Name: "f_gas_price", Data: f_gas_price},
		{Name: "f_gas_tip_cap", Data: f_gas_tip_cap},
		{Name: "f_gas_fee_cap", Data: f_gas_fee_cap},
		{Name: "f_value", Data: f_value},
		{Name: "f_nonce", Data: f_nonce},
		{Name: "f_to", Data: f_to},
		{Name: "f_hash", Data: f_hash},
		{Name: "f_size", Data: f_size},
		{Name: "f_slot", Data: f_slot},
		{Name: "f_el_block_number", Data: f_el_block_number},
		{Name: "f_timestamp", Data: f_timestamp},
		{Name: "f_from", Data: f_from},
		{Name: "f_contract_address", Data: f_contract_address},
	}
}

type DeleteTransactions struct {
	slot phase0.Slot
}

func (d DeleteTransactions) Query() string {
	return fmt.Sprintf(deleteTransactionsQuery, transactionsTable)
}

func (d DeleteTransactions) Table() string {
	return transactionsTable
}

func (d DeleteTransactions) Args() []any {
	return []any{d.slot}
}
