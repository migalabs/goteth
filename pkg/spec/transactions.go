package spec

import (
	"encoding/hex"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// A wrapper for blockchain transaction with basic information retrieved from the Ethereum blockchain
type AgnosticTransaction struct {
	TxType      uint8           // type of transaction: LegacyTxType, AccessListTxType, or DynamicFeeTxType
	ChainId     uint8           // a unique identifier for the ethereum network
	Data        string          // the input data of the transaction
	Gas         phase0.Gwei     // the gas limit of the transaction
	GasPrice    phase0.Gwei     // the gas price of the transaction
	GasTipCap   phase0.Gwei     // the tip cap per gas of the transaction
	GasFeeCap   phase0.Gwei     // the fee cap per gas of the transaction
	Value       phase0.Gwei     // the ether amount of the transaction.
	Nonce       uint64          // the sender account nonce of the transaction
	To          *common.Address // transaction recipient's address
	Hash        phase0.Hash32   // the transaction hash
	Size        uint64          // the true encoded storage size of the transaction
	Slot        phase0.Slot     // the slot of the transaction
	BlockNumber uint64          // the number of the block where this transaction was added
	Timestamp   uint64          // timestamp of the block to which this transaction belongs
}

func (txs AgnosticTransaction) Type() ModelType {
	return TransactionsModel
}

// convert transactions from byte sequences to Transaction object
func RequestTransactionDetails(block AgnosticBlock) []*AgnosticTransaction {
	processedTransactions := make([]*AgnosticTransaction, 0)
	for idx := 0; idx < len(block.ExecutionPayload.Transactions); idx++ {
		var parsedTx = &types.Transaction{}
		if err := parsedTx.UnmarshalBinary(block.ExecutionPayload.Transactions[idx]); err != nil {
			return nil
		}

		processedTransactions = append(processedTransactions, &AgnosticTransaction{
			TxType:      parsedTx.Type(),
			ChainId:     uint8(parsedTx.ChainId().Uint64()),
			Data:        hex.EncodeToString(parsedTx.Data()),
			Gas:         phase0.Gwei(parsedTx.Gas()),
			GasPrice:    phase0.Gwei(parsedTx.GasPrice().Uint64()),
			GasTipCap:   phase0.Gwei(parsedTx.GasTipCap().Uint64()),
			GasFeeCap:   phase0.Gwei(parsedTx.GasFeeCap().Uint64()),
			Value:       phase0.Gwei(parsedTx.Value().Uint64()),
			Nonce:       parsedTx.Nonce(),
			To:          parsedTx.To(),
			Hash:        phase0.Hash32(parsedTx.Hash()),
			Size:        parsedTx.Size(),
			Slot:        block.Slot,
			BlockNumber: block.ExecutionPayload.BlockNumber,
			Timestamp:   block.ExecutionPayload.Timestamp,
		})
	}
	return processedTransactions
}
