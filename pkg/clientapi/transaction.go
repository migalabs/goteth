package clientapi

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/attestantio/go-eth2-client/spec/bellatrix"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/migalabs/goteth/pkg/spec"
	"github.com/migalabs/goteth/pkg/utils"
)

func (client *APIClient) GetBlockReceipts(block spec.AgnosticBlock) ([]*types.Receipt, error) {
	if client.ELApi == nil {
		return nil, errors.New("execution endpoint not configured")
	}

	maxAttempts := client.maxRetries
	if maxAttempts <= 0 {
		maxAttempts = 1
	}

	blockHash := common.BytesToHash(block.ExecutionPayload.BlockHash[:])
	blockNumber := rpc.BlockNumber(block.ExecutionPayload.BlockNumber)
	routineKey := fmt.Sprintf("receipts=%s:%d", blockHash.Hex(), block.ExecutionPayload.BlockNumber)
	client.txBook.Acquire(routineKey)
	defer client.txBook.FreePage(routineKey)
	var (
		receipts []*types.Receipt
		err      error
	)

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		receipts, err = client.requestBlockReceipts(blockHash, blockNumber)
		if err == nil {
			return receipts, nil
		}

		if attempt < maxAttempts {
			waitTime := utils.RoutineFlushTimeout * time.Duration(attempt)
			if waitTime <= 0 {
				waitTime = utils.RoutineFlushTimeout
			}
			log.Warnf("retrying block receipts request: block=%d attempt=%d err=%s", block.ExecutionPayload.BlockNumber, attempt, err)
			select {
			case <-time.After(waitTime):
			case <-client.ctx.Done():
				reason := classifyReceiptError(err)
				client.recordReceiptFailure(reason, attempt)
				return nil, fmt.Errorf("context cancelled while waiting for block %d receipts: %w", block.ExecutionPayload.BlockNumber, client.ctx.Err())
			}
		}
	}

	reason := classifyReceiptError(err)
	client.recordReceiptFailure(reason, maxAttempts)
	return nil, fmt.Errorf("unable to retrieve block %d receipts after %d attempts: %w", block.ExecutionPayload.BlockNumber, maxAttempts, err)
}

// convert transactions from byte sequences to Transaction object
func (s *APIClient) GetTransactionReceipt(iTx bellatrix.Transaction,
	iSlot phase0.Slot,
	iBlockNumber uint64,
	iTimestamp uint64) (*types.Receipt, error) {

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
		for err != nil && attempts < s.maxRetries {
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

	return receipt, nil

}

func (client *APIClient) recordReceiptFailure(reason string, attempts int) {
	if client == nil || client.receiptMetrics == nil {
		return
	}
	client.receiptMetrics.recordFailure(reason, attempts)
}

func classifyReceiptError(err error) string {
	if err == nil {
		return "unknown"
	}

	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "canonical"):
		return "not_canonical"
	case strings.Contains(msg, "not found"):
		return "not_found"
	case strings.Contains(msg, "context deadline exceeded"):
		return "deadline_exceeded"
	case strings.Contains(msg, "context canceled"), strings.Contains(msg, "context cancelled"):
		return "context_cancelled"
	default:
		return "other"
	}
}

func (client *APIClient) requestBlockReceipts(blockHash common.Hash, blockNumber rpc.BlockNumber) ([]*types.Receipt, error) {
	var selector rpc.BlockNumberOrHash
	if blockHash != (common.Hash{}) {
		selector = rpc.BlockNumberOrHashWithHash(blockHash, false)
	} else {
		selector = rpc.BlockNumberOrHashWithNumber(blockNumber)
	}

	receipts, err := client.ELApi.BlockReceipts(client.ctx, selector)
	if err != nil && blockHash != (common.Hash{}) {
		log.Debugf("receipt request by hash failed (%s), retrying via block number %d", blockHash.Hex(), blockNumber)
		return client.ELApi.BlockReceipts(client.ctx, rpc.BlockNumberOrHashWithNumber(blockNumber))
	}
	return receipts, err
}
