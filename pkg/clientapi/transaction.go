package clientapi

import (
	"encoding/hex"
	"errors"
	"time"

	"github.com/attestantio/go-eth2-client/spec/bellatrix"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/migalabs/goteth/pkg/spec"
	"github.com/migalabs/goteth/pkg/utils"
)

var (
	blobTxType uint8 = 3
)

func (client *APIClient) ParseSingleTx(
	parsedTx types.Transaction,
	receipt *types.Receipt,
	slot phase0.Slot,
	blockNumber uint64,
	timestamp uint64) (spec.AgnosticTransaction, error) {

	from, err := types.Sender(types.LatestSignerForChainID(parsedTx.ChainId()), &parsedTx)
	if err != nil {
		log.Warnf("unable to retrieve sender address from transaction: %s", err)
		return spec.AgnosticTransaction{}, err
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

	return spec.AgnosticTransaction{
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
		Slot:            slot,
		BlockNumber:     blockNumber,
		Timestamp:       timestamp,
		ContractAddress: contractAddress,
		BlobGasUsed:     blobGasUsed,
		BlobGasPrice:    blobGasPrice,
		BlobGasLimit:    blobGasLimit,
		BlobGasFeeCap:   blobGasFeeCap,
		BlobHashes:      parsedTx.BlobHashes(),
	}, nil

}

func (client *APIClient) GetBlockTransactions(block spec.AgnosticBlock) ([]spec.AgnosticTransaction, error) {
	agnosticTxs := make([]spec.AgnosticTransaction, 0)
	blockNumber := rpc.BlockNumber(block.ExecutionPayload.BlockNumber)
	receipts, err := client.ELApi.BlockReceipts(client.ctx, rpc.BlockNumberOrHashWithNumber(blockNumber))
	if err != nil {
		return nil, err
	}

	txs := make([]bellatrix.Transaction, len(block.ExecutionPayload.Transactions))
	copy(txs, block.ExecutionPayload.Transactions)

	// match receipts and transactions

	for _, tx := range txs {
		var parsedTx = &types.Transaction{}
		if err := parsedTx.UnmarshalBinary(tx); err != nil {
			return nil, err
		}
		for _, receipt := range receipts {
			if receipt.TxHash.String() == parsedTx.Hash().String() {
				// we found a match
				agnosticTx, err := client.ParseSingleTx(
					*parsedTx,
					receipt,
					block.Slot,
					block.ExecutionPayload.BlockNumber,
					block.ExecutionPayload.Timestamp)
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

// convert transactions from byte sequences to Transaction object
func (s *APIClient) RequestTransactionDetails(iTx bellatrix.Transaction,
	iSlot phase0.Slot,
	iBlockNumber uint64,
	iTimestamp uint64) (*spec.AgnosticTransaction, error) {

	var parsedTx = &types.Transaction{}
	if err := parsedTx.UnmarshalBinary(iTx); err != nil {
		log.Warnf("unable to unmarshal transaction: %s", err)
		return nil, err
	}

	var receipt *types.Receipt
	err := errors.New("first attempt")

	if s.ELApi == nil {
		log.Warn("EL endpoint not provided. The gas price read from the CL may not be the effective gas price.")
		attempts := 0
		for err != nil && attempts < maxRetries {
			receipt, err = s.ELApi.TransactionReceipt(s.ctx, parsedTx.Hash())

			if err != nil {
				ticker := time.NewTicker(utils.RoutineFlushTimeout)
				log.Warnf("retrying transaction request: %s", parsedTx.Hash().String())
				<-ticker.C
			}
			attempts += 1

		}
		if err != nil {
			log.Errorf("could not retrieve the receipt for tx %s: %s", parsedTx.Hash().String(), err)
		}
	}

	agnosticTx, err := s.ParseSingleTx(
		*parsedTx,
		receipt,
		iSlot,
		iBlockNumber,
		iTimestamp)

	if err != nil {
		return nil, err
	}
	return &agnosticTx, nil

}
