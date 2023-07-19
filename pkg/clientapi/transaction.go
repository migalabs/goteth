package clientapi

import (
	"encoding/hex"

	"github.com/attestantio/go-eth2-client/spec/bellatrix"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/cortze/eth-cl-state-analyzer/pkg/spec"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// GetReceipt retrieves receipt for the given transaction hash
func (client APIClient) GetReceipt(txHash common.Hash) (*types.Receipt, error) {
	receipt, err := client.ELApi.TransactionReceipt(client.ctx, txHash)
	return receipt, err
}

// convert transactions from byte sequences to Transaction object
func (s APIClient) RequestTransactionDetails(iTx bellatrix.Transaction,
	iSlot phase0.Slot,
	iBlockNumber uint64,
	iTimestamp uint64) (*spec.AgnosticTransaction, error) {
	// starTime := time.Now()
	if s.ELApi == nil {
		log.Warn("EL endpoint not provided. The gas price read from the CL may not be the effective gas price.")
	}
	var parsedTx = &types.Transaction{}
	if err := parsedTx.UnmarshalBinary(iTx); err != nil {
		log.Warnf("unable to unmarshal transaction: %s", err)
		return &spec.AgnosticTransaction{}, err
	}
	from, err := types.Sender(types.LatestSignerForChainID(parsedTx.ChainId()), parsedTx)
	if err != nil {
		log.Warnf("unable to retrieve sender address from transaction: %s", err)
		return &spec.AgnosticTransaction{}, err
	}

	gasUsed := parsedTx.Gas()
	gasPrice := parsedTx.GasPrice().Uint64()
	contractAddress := *&common.Address{}

	if s.ELApi != nil {
		receipt, err := s.GetReceipt(parsedTx.Hash())

		if err != nil {
			log.Warnf("unable to retrieve transaction receipt for hash %x: %s", parsedTx.Hash(), err.Error())
		} else {
			gasUsed = receipt.GasUsed
			gasPrice = receipt.EffectiveGasPrice.Uint64()
			contractAddress = receipt.ContractAddress
		}

	}

	//log.Infof("transaction details took %f seconds", time.Since(starTime).Seconds())

	return &spec.AgnosticTransaction{
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
		Slot:            iSlot,
		BlockNumber:     iBlockNumber,
		Timestamp:       iTimestamp,
		ContractAddress: contractAddress,
	}, nil
}
