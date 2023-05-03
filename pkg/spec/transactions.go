package spec

import (
	"encoding/hex"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// A wrapper for blockchain transaction with basic information retrieved from the Ethereum blockchain
type AgnosticTransaction struct {
	TxType      uint8
	ChainId     uint64
	Data        string
	Gas         uint64
	GasPrice    uint64
	GasTipCap   uint64
	GasFeeCap   uint64
	Value       uint64
	Nonce       uint64
	To          *common.Address
	Hash        common.Hash
	Size        uint64
	Slot        phase0.Slot
	BlockNumber uint64
	Timestamp   uint64
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
			ChainId:     parsedTx.ChainId().Uint64(),
			Data:        hex.EncodeToString(parsedTx.Data()),
			Gas:         parsedTx.Gas(),
			GasPrice:    parsedTx.GasPrice().Uint64(),
			GasTipCap:   parsedTx.GasTipCap().Uint64(),
			GasFeeCap:   parsedTx.GasFeeCap().Uint64(),
			Value:       parsedTx.Value().Uint64(),
			Nonce:       parsedTx.Nonce(),
			To:          parsedTx.To(),
			Hash:        parsedTx.Hash(),
			Size:        parsedTx.Size(),
			Slot:        block.Slot,
			BlockNumber: block.ExecutionPayload.BlockNumber,
			Timestamp:   block.ExecutionPayload.Timestamp,
		})
	}
	return processedTransactions
}
