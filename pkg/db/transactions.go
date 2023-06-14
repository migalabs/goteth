package db

import (
	"github.com/cortze/eth-cl-state-analyzer/pkg/spec"
)

var (
	UpsertTransaction = `
INSERT INTO t_transactions(
	f_tx_type, f_chain_id, f_data, f_gas, f_gas_price, f_gas_tip_cap, f_gas_fee_cap, f_value, f_nonce, f_to, f_hash,
                           f_size, f_slot, f_el_block_number, f_timestamp, f_from)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
ON CONFLICT DO NOTHING;`
)

/**
 * Extract parameters required to create transaction and return query with args
 */
func insertTransaction(transaction *spec.AgnosticTransaction) (string, []interface{}) {
	resultArgs := make([]interface{}, 0)

	resultArgs = append(resultArgs, transaction.Type())
	resultArgs = append(resultArgs, transaction.ChainId)
	resultArgs = append(resultArgs, transaction.Data)
	resultArgs = append(resultArgs, transaction.Gas)
	resultArgs = append(resultArgs, transaction.GasPrice)
	resultArgs = append(resultArgs, transaction.GasTipCap)
	resultArgs = append(resultArgs, transaction.GasFeeCap)
	resultArgs = append(resultArgs, transaction.Value)
	resultArgs = append(resultArgs, transaction.Nonce)
	if transaction.To != nil { // some transactions appear to have nil to field
		resultArgs = append(resultArgs, transaction.To.String())
	} else {
		resultArgs = append(resultArgs, "")
	}
	resultArgs = append(resultArgs, transaction.Hash.String())
	resultArgs = append(resultArgs, transaction.Size)
	resultArgs = append(resultArgs, transaction.Slot)
	resultArgs = append(resultArgs, transaction.BlockNumber)
	resultArgs = append(resultArgs, transaction.Timestamp)
	resultArgs = append(resultArgs, transaction.From.String())
	return UpsertTransaction, resultArgs
}

/**
 * Handle block db operation by forming the insertion query from transaction info
 */
func TransactionOperation(transaction *spec.AgnosticTransaction) (string, []interface{}) {
	q, args := insertTransaction(transaction)

	return q, args
}
