package clientapi

import (
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/attestantio/go-eth2-client/spec/bellatrix"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/migalabs/goteth/pkg/spec"
	"github.com/migalabs/goteth/pkg/utils"
)

var (
	blockNumTag string = "block="
)

func (a *APIClient) GetBlockTxReceipts(block spec.AgnosticBlock) []spec.AgnosticTransaction {

	blockNum := rpc.BlockNumber(block.ExecutionPayload.BlockNumber)

	if a.ELApi != nil {

		routineKey := fmt.Sprintf("%s%d", blockNumTag, blockNum)
		a.txBook.Acquire(routineKey)
		defer a.txBook.FreePage(routineKey)

		blockNumber := rpc.BlockNumberOrHash{BlockNumber: &blockNum}
		err := errors.New("first attempt")

		var receipts []*types.Receipt

		attempts := 0
		for err != nil && attempts < maxRetries {

			start := time.Now()
			receipts, err = a.ELApi.BlockReceipts(a.ctx, blockNumber)
			elapsedTime := time.Since(start)
			log.Infof("block receipts on slot %d downloaded in %f seconds", block.Slot, elapsedTime.Seconds())

			if err != nil {
				ticker := time.NewTicker(utils.RoutineFlushTimeout)
				log.Warnf("retrying block receipts request: %d", blockNum)
				<-ticker.C
			}
			attempts += 1

		}

		if err != nil {
			log.Fatalf("unable to retrieve transaction receipts for block %d: %s", blockNum, err.Error())
		}

		return a.GetAgnosticTransactions(
			block,
			receipts)

	}
	return []spec.AgnosticTransaction{}
}

func (a *APIClient) GetAgnosticTransactions(
	block spec.AgnosticBlock,
	receipts []*types.Receipt) []spec.AgnosticTransaction {

	var blockTxList []bellatrix.Transaction
	blockTxList = block.ExecutionPayload.Transactions

	agnosticTxs := make([]spec.AgnosticTransaction, 0)

	for _, item := range receipts {
		receiptTxHash := item.TxHash.String()

		var parsedTx = &types.Transaction{}

		for idx, item := range blockTxList { // find the transaction in the block
			var tmpParsedTx = &types.Transaction{}
			tmpParsedTx, _ = parseTxHash(item)
			blockTxHash := tmpParsedTx.Hash().String()

			if receiptTxHash == blockTxHash {
				parsedTx = tmpParsedTx
				blockTxList = remove(blockTxList, idx)
				break
			}
		}

		if parsedTx == nil { // we did not find the transaction
			log.Fatalf("we did not find the transactions for the given receipt")
		}

		from, err := types.Sender(types.LatestSignerForChainID(parsedTx.ChainId()), parsedTx)
		if err != nil {
			log.Warnf("unable to retrieve sender address from transaction %s: %s", item.TxHash.String(), err)
		}

		tmpAgnosticTx := spec.AgnosticTransaction{
			TxType:          item.Type,
			ChainId:         uint8(parsedTx.ChainId().Uint64()),
			Data:            hex.EncodeToString(parsedTx.Data()),
			Gas:             phase0.Gwei(item.GasUsed),
			GasPrice:        phase0.Gwei(item.EffectiveGasPrice.Int64()),
			GasTipCap:       phase0.Gwei(parsedTx.GasTipCap().Uint64()),
			GasFeeCap:       phase0.Gwei(parsedTx.GasFeeCap().Uint64()),
			Value:           phase0.Gwei(parsedTx.Value().Uint64()),
			Nonce:           parsedTx.Nonce(),
			To:              parsedTx.To(),
			From:            from,
			Hash:            phase0.Hash32(parsedTx.Hash()),
			Size:            parsedTx.Size(),
			Slot:            block.Slot,
			BlockNumber:     uint64(block.ExecutionPayload.BlockNumber),
			Timestamp:       block.ExecutionPayload.Timestamp,
			ContractAddress: item.ContractAddress,
		}
		agnosticTxs = append(agnosticTxs, tmpAgnosticTx)
	}

	return agnosticTxs
}
