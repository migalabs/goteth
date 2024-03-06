package spec

import (
	"math/big"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethereum/go-ethereum/common"
)

// A wrapper for blockchain transaction with basic information retrieved from the Ethereum blockchain
type AgnosticTransaction struct {
	TxType          uint8           // type of transaction: LegacyTxType, AccessListTxType, or DynamicFeeTxType
	ChainId         uint8           // a unique identifier for the ethereum network
	Data            string          // the input data of the transaction
	Gas             phase0.Gwei     // the gas limit of the transaction
	GasPrice        phase0.Gwei     // the gas price of the transaction
	GasTipCap       phase0.Gwei     // the tip cap per gas of the transaction
	GasFeeCap       phase0.Gwei     // the fee cap per gas of the transaction
	Value           phase0.Gwei     // the ether amount of the transaction.
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
	BlobGasUsed   uint64
	BlobGasPrice  *big.Int
	BlobGasLimit  uint64
	BlobGasFeeCap *big.Int
}

func (txs AgnosticTransaction) Type() ModelType {
	return TransactionsModel
}
