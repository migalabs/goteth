package clientapi

import (
	"errors"
	"time"

	"github.com/attestantio/go-eth2-client/spec/bellatrix"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/migalabs/goteth/pkg/spec"
	"github.com/migalabs/goteth/pkg/utils"
)

var (
	blobTxType uint8 = 3
)

func (client *APIClient) GetBlockReceipts(block spec.AgnosticBlock) ([]*types.Receipt, error) {
	blockNumber := rpc.BlockNumber(block.ExecutionPayload.BlockNumber)
	receipts, err := client.ELApi.BlockReceipts(client.ctx, rpc.BlockNumberOrHashWithNumber(blockNumber))
	if err != nil {
		return nil, err
	}
	return receipts, nil
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
