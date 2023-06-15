package spec

import (
	"encoding/hex"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/cortze/eth-cl-state-analyzer/pkg/clientapi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
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
}

func (txs AgnosticTransaction) Type() ModelType {
	return TransactionsModel
}

// convert transactions from byte sequences to Transaction object
func RequestTransactionDetails(block AgnosticBlock, client *clientapi.APIClient) ([]*AgnosticTransaction, error) {
	processedTransactions := make([]*AgnosticTransaction, 0)
	if client.ELApi == nil {
		log.Warn("EL endpoint not provided. The gas price read from the CL may not be the effective gas price.")
	}
	for idx := 0; idx < len(block.ExecutionPayload.Transactions); idx++ {
		var parsedTx = &types.Transaction{}
		if err := parsedTx.UnmarshalBinary(block.ExecutionPayload.Transactions[idx]); err != nil {
			log.Warnf("unable to unmarshal transaction: %s", err)
			return processedTransactions, err
		}
		from, err := types.Sender(types.LatestSignerForChainID(parsedTx.ChainId()), parsedTx)
		if err != nil {
			log.Warnf("unable to retrieve sender address from transaction: %s", err)
			return processedTransactions, err
		}

		gasUsed := parsedTx.Gas()
		gasPrice := parsedTx.GasPrice().Uint64()
		contractAddress := *&common.Address{}

		if client.ELApi != nil {
			receipt, err := clientapi.GetReceipt(parsedTx.Hash(), client)

			if err != nil {
				log.Warnf("unable to retrieve transaction receipt for hash %x: %s", parsedTx.Hash(), err.Error())
			} else {
				gasUsed = receipt.GasUsed
				gasPrice = receipt.EffectiveGasPrice.Uint64()
				contractAddress = receipt.ContractAddress
			}

		}

		processedTransaction := &AgnosticTransaction{
			TxType:          parsedTx.Type(),
			ChainId:         uint8(parsedTx.ChainId().Uint64()),
			Data:            hex.EncodeToString(parsedTx.Data()),
			Gas:             phase0.Gwei(gasUsed),
			GasPrice:        phase0.Gwei(gasPrice),
			GasTipCap:       phase0.Gwei(parsedTx.GasTipCap().Uint64()),
			GasFeeCap:       phase0.Gwei(parsedTx.GasFeeCap().Uint64()),
			Value:           phase0.Gwei(parsedTx.Value().Uint64()),
			Nonce:           parsedTx.Nonce(),
			To:              parsedTx.To(),
			From:            from,
			Hash:            phase0.Hash32(parsedTx.Hash()),
			Size:            parsedTx.Size(),
			Slot:            block.Slot,
			BlockNumber:     block.ExecutionPayload.BlockNumber,
			Timestamp:       block.ExecutionPayload.Timestamp,
			ContractAddress: contractAddress,
		}

		processedTransactions = append(processedTransactions, processedTransaction)

	}
	return processedTransactions, nil
}
