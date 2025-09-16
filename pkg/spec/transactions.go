package spec

import (
	"encoding/hex"

	"github.com/attestantio/go-eth2-client/spec/bellatrix"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

var (
	blobTxType uint8 = 3
)

// A wrapper for blockchain transaction with basic information retrieved from the Ethereum blockchain
type AgnosticTransaction struct {
	TxIdx           uint64          // transaction index in the block
	TxType          uint8           // type of transaction: LegacyTxType, AccessListTxType, or DynamicFeeTxType
	ChainId         uint8           // a unique identifier for the ethereum network
	Data            string          // the input data of the transaction
	Gas             uint64          // the gas limit of the transaction
	GasPrice        uint64          // the gas price of the transaction
	GasTipCap       uint64          // the tip cap per gas of the transaction
	GasFeeCap       uint64          // the fee cap per gas of the transaction
	Value           uint64          // the ether amount of the transaction.
	Nonce           uint64          // the sender account nonce of the transaction
	To              *common.Address // transaction recipient's address
	From            common.Address  // transaction sender's address
	Hash            phase0.Hash32   // the transaction hash
	Size            uint64          // the true encoded storage size of the transaction
	Slot            phase0.Slot     // the slot of the transaction
	BlockNumber     uint64          // the number of the block where this transaction was added
	Timestamp       uint64          // timestamp of the block to which this transaction belongs
	ContractAddress common.Address  // address of the smart contract associated with this transaction

	// Blobs
	BlobHashes    []common.Hash
	BlobGasUsed   uint64 // amount of gas used
	BlobGasPrice  uint64 // price per unit of gas used => Wei
	BlobGasLimit  uint64 // maximum gas allowed
	BlobGasFeeCap uint64

	// Receipt
	Receipt *types.Receipt
}

func (txs AgnosticTransaction) Type() ModelType {
	return TransactionsModel
}

func ParseTransactionsFromBlock(block AgnosticBlock, receipts []*types.Receipt) ([]AgnosticTransaction, error) {
	agnosticTxs := make([]AgnosticTransaction, 0)
	txs := make([]bellatrix.Transaction, len(block.ExecutionPayload.Transactions))
	copy(txs, block.ExecutionPayload.Transactions)

	// match receipts and transactions

	for txIdx, tx := range txs {
		var parsedTx = &types.Transaction{}
		if err := parsedTx.UnmarshalBinary(tx); err != nil {
			return nil, err
		}
		for _, receipt := range receipts {
			if receipt.TxHash.String() == parsedTx.Hash().String() {
				// we found a match
				agnosticTx, err := ParseTransactionFromReceipt(
					parsedTx,
					receipt,
					block.Slot,
					block.ExecutionPayload.BlockNumber,
					block.ExecutionPayload.Timestamp,
					uint64(txIdx))
				if err != nil {
					return nil, err
				}
				agnosticTxs = append(agnosticTxs, agnosticTx)
				break
			}
		}
	}
	return agnosticTxs, nil
}

func ParseTransactionFromReceipt(
	parsedTx *types.Transaction,
	receipt *types.Receipt,
	slot phase0.Slot,
	blockNumber uint64,
	timestamp uint64,
	txIdx uint64,
) (AgnosticTransaction, error) {

	from, err := types.Sender(types.LatestSignerForChainID(parsedTx.ChainId()), parsedTx)
	if err != nil {
		log.Warnf("unable to retrieve sender address from transaction: %s", err)
		return AgnosticTransaction{}, err
	}

	gasUsed := parsedTx.Gas()
	gasPrice := parsedTx.GasPrice().Uint64()
	contractAddress := common.Address{}
	blobGasUsed := uint64(0)
	blobGasPrice := uint64(0)
	blobGasLimit := uint64(0)
	blobGasFeeCap := uint64(0)

	if receipt != nil {
		gasUsed = receipt.GasUsed
		gasPrice = receipt.EffectiveGasPrice.Uint64()
		contractAddress = receipt.ContractAddress
	}

	if parsedTx.Type() == blobTxType {
		blobGasUsed = receipt.BlobGasUsed
		blobGasPrice = receipt.BlobGasPrice.Uint64()
		blobGasLimit = parsedTx.BlobGas()
		blobGasFeeCap = parsedTx.BlobGasFeeCap().Uint64()
	}

	return AgnosticTransaction{
		TxIdx:           txIdx,
		TxType:          parsedTx.Type(),
		ChainId:         uint8(parsedTx.ChainId().Uint64()),
		Data:            hex.EncodeToString(parsedTx.Data()),
		Gas:             gasUsed,
		GasPrice:        gasPrice,
		GasTipCap:       parsedTx.GasTipCap().Uint64(),
		GasFeeCap:       parsedTx.GasFeeCap().Uint64(),
		Value:           parsedTx.Value().Uint64(),
		Nonce:           parsedTx.Nonce(),
		To:              parsedTx.To(),
		From:            from,
		Hash:            phase0.Hash32(parsedTx.Hash()),
		Size:            parsedTx.Size(),
		Slot:            slot,
		BlockNumber:     blockNumber,
		Timestamp:       timestamp,
		ContractAddress: contractAddress,
		BlobGasUsed:     blobGasUsed,
		BlobGasPrice:    blobGasPrice,
		BlobGasLimit:    blobGasLimit,
		BlobGasFeeCap:   blobGasFeeCap,
		BlobHashes:      parsedTx.BlobHashes(),
		Receipt:         receipt,
	}, nil

}
