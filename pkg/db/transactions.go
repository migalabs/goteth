package db

import (
	"fmt"

	"github.com/ClickHouse/ch-go/proto"
	"github.com/migalabs/goteth/pkg/spec"
)

var (
	transactionsTable       = "t_transactions"
	insertTransactionsQuery = `
		INSERT INTO %s(
			f_tx_idx,
			f_tx_type, 
			f_chain_id, 
			f_data, f_gas, 
			f_gas_price, 
			f_gas_tip_cap, 
			f_gas_fee_cap, 
			f_value, 
			f_nonce, 
			f_to, 
			f_hash,
			f_size, 
			f_slot, 
			f_el_block_number, 
			f_timestamp, 
			f_from, 
			f_contract_address,
			f_blob_gas_used,
			f_blob_gas_price,
			f_blob_gas_limit,
			f_blob_gas_fee_cap)
		VALUES`

	deleteTransactionsQuery = `
		DELETE FROM %s
		WHERE f_slot = $1;
`

	selectTransactionGapsRangeQuery = `
		WITH tx_counts AS (
			SELECT
				f_slot,
				count() AS tx_count
			FROM %[1]s
			WHERE f_slot BETWEEN %[2]d AND %[3]d
			GROUP BY f_slot
		)
		SELECT
			bm.f_slot AS f_slot,
			bm.f_el_transactions AS f_el_transactions,
			COALESCE(tx_counts.tx_count, 0) AS tx_count
		FROM %[4]s AS bm
		LEFT JOIN tx_counts ON bm.f_slot = tx_counts.f_slot
		WHERE bm.f_slot BETWEEN %[2]d AND %[3]d
			AND bm.f_el_transactions != COALESCE(tx_counts.tx_count, 0)
		ORDER BY bm.f_slot`
)

func transactionsInput(transactions []spec.AgnosticTransaction) proto.Input {
	// one object per column
	var (
		f_tx_idx           proto.ColUInt64
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
		f_blob_gas_used    proto.ColUInt64
		f_blob_gas_price   proto.ColUInt64
		f_blob_gas_limit   proto.ColUInt64
		f_blob_gas_fee_cap proto.ColUInt64
	)

	for _, transaction := range transactions {
		f_tx_idx.Append(transaction.TxIdx)
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

		f_blob_gas_used.Append(transaction.BlobGasUsed)
		f_blob_gas_price.Append(transaction.BlobGasPrice)
		f_blob_gas_limit.Append(transaction.BlobGasLimit)
		f_blob_gas_fee_cap.Append(transaction.BlobGasFeeCap)
	}

	return proto.Input{
		{Name: "f_tx_idx", Data: f_tx_idx},
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
		{Name: "f_blob_gas_used", Data: f_blob_gas_used},
		{Name: "f_blob_gas_price", Data: f_blob_gas_price},
		{Name: "f_blob_gas_limit", Data: f_blob_gas_limit},
		{Name: "f_blob_gas_fee_cap", Data: f_blob_gas_fee_cap},
	}
}

func (p *DBService) PersistTransactions(data []spec.AgnosticTransaction) error {
	persistObj := PersistableObject[spec.AgnosticTransaction]{
		input: transactionsInput,
		table: transactionsTable,
		query: insertTransactionsQuery,
	}

	for _, item := range data {
		persistObj.Append(item)
	}

	err := p.Persist(persistObj.ExportPersist())
	if err != nil {
		log.Errorf("error persisting transactions: %s", err.Error())
	}
	return err
}

type TransactionGap struct {
	Slot     uint64
	Expected uint64
	Actual   uint64
}

func (p *DBService) RetrieveTransactionGapsRange(startSlot, endSlot uint64) ([]TransactionGap, error) {
	if endSlot < startSlot {
		return []TransactionGap{}, nil
	}
	query := fmt.Sprintf(selectTransactionGapsRangeQuery, transactionsTable, startSlot, endSlot, blocksTable)
	var dest []struct {
		F_slot            uint64 `ch:"f_slot"`
		F_el_transactions uint64 `ch:"f_el_transactions"`
		Tx_count          uint64 `ch:"tx_count"`
	}

	err := p.highSelect(query, &dest)
	if err != nil {
		return nil, err
	}

	gaps := make([]TransactionGap, len(dest))
	for i, row := range dest {
		gaps[i] = TransactionGap{
			Slot:     row.F_slot,
			Expected: row.F_el_transactions,
			Actual:   row.Tx_count,
		}
	}
	return gaps, nil
}
