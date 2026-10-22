package analyzer

import (
	"fmt"

	eth2_client_spec "github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/migalabs/goteth/pkg/spec"
)

var (
	slotProcesserTag = "slot="
)

func (s *ChainAnalyzer) ProcessBlock(slot phase0.Slot) {
	if !s.metrics.Block {
		return
	}
	routineKey := fmt.Sprintf("%s%d", slotProcesserTag, slot)
	s.processerBook.Acquire(routineKey) // register a new slot to process, good for monitoring

	block, err := s.downloadCache.BlockHistory.Wait(s.ctx, SlotTo[uint64](slot))
	if err != nil {
		s.processerBook.FreePage(routineKey)
		log.Errorf("context cancelled waiting for block at slot %d: %s", slot, err)
		return
	}
	err = s.dbClient.PersistBlocks([]spec.AgnosticBlock{*block})
	if err != nil {
		log.Errorf("error persisting blocks: %s", err.Error())
	}

	s.processWithdrawals(block)

	s.ProcessETH1Data(block)

	s.processBLSToExecutionChanges(block)
	s.processDeposits(block)
	s.processerBook.FreePage(routineKey)
}

func (s *ChainAnalyzer) ProcessETH1Data(block *spec.AgnosticBlock) {
	if s.metrics.Transactions {
		receipts, err := s.cli.GetBlockReceipts(*block)
		if err != nil {
			log.Errorf("error getting slot %d receipts: %s", block.Slot, err.Error())
			return
		}

		err = s.processTransactions(block, receipts)
		if err != nil {
			log.Errorf("error processing transactions: %s", err.Error())
			return
		}

		// process eth1 deposits depends on processTransactions storing the receipts on the Agnostic transactions
		err = s.processETH1Deposits(block)
		if err != nil {
			log.Errorf("error processing eth1 deposits: %s", err.Error())
			return
		}
	}

	if block.HardForkVersion >= eth2_client_spec.DataVersionDeneb && s.metrics.BlobSidecars {
		s.processBlobSidecars(block)
	}
}

func (s *ChainAnalyzer) processETH1Deposits(block *spec.AgnosticBlock) error {
	var deposits []spec.ETH1Deposit
	for _, tx := range block.ExecutionPayload.AgnosticTransactions {
		for _, logEntry := range tx.Receipt.Logs {
			if logEntry.Address == s.beaconContractAddress { //&& logEntry.Topics[0] == common.HexToHash(spec.DepositEventTopic) {
				if len(logEntry.Topics) == 0 {
					log.Warnf("deposit log entry has no topics")
					continue
				}
				if logEntry.Topics[0].String() != spec.DepositEventTopic { // Added because on sepolia deposits have more than one log
					continue
				}
				if len(logEntry.Data) < spec.DepositEventDataLength {
					log.Warnf("deposit log entry has the incorrect length")
					continue
				}
				deposit := spec.ParseETH1DepositFromLog(logEntry, &tx)
				deposits = append(deposits, deposit)
			}
		}
	}

	if len(deposits) == 0 {
		return nil
	}
	err := s.dbClient.PersistETH1Deposits(deposits)
	if err != nil {
		log.Errorf("error persisting eth1 deposits: %s", err.Error())
	}
	return err
}

// Process consensus layer deposits
func (s *ChainAnalyzer) processDeposits(block *spec.AgnosticBlock) {
	if len(block.Deposits) == 0 {
		return
	}
	var deposits []spec.Deposit
	for i, item := range block.Deposits {
		deposits = append(deposits, spec.Deposit{
			Slot:                  block.Slot,
			EpochProcessed:        spec.EpochAtSlot(block.Slot),
			PublicKey:             item.Data.PublicKey,
			WithdrawalCredentials: item.Data.WithdrawalCredentials,
			Amount:                item.Data.Amount,
			Signature:             item.Data.Signature,
			Index:                 uint8(i),
		})
	}

	err := s.dbClient.PersistDeposits(deposits)
	if err != nil {
		log.Errorf("error persisting deposits: %s", err.Error())
	}

}

func (s *ChainAnalyzer) processBLSToExecutionChanges(block *spec.AgnosticBlock) {
	if len(block.BLSToExecutionChanges) == 0 {
		return
	}
	var blsToExecutionChanges []spec.BLSToExecutionChange
	for _, item := range block.BLSToExecutionChanges {
		blsToExecutionChanges = append(blsToExecutionChanges, spec.BLSToExecutionChange{
			Slot:               block.Slot,
			Epoch:              spec.EpochAtSlot(block.Slot),
			ValidatorIndex:     item.Message.ValidatorIndex,
			FromBLSPublicKey:   item.Message.FromBLSPubkey,
			ToExecutionAddress: item.Message.ToExecutionAddress,
		})
	}

	err := s.dbClient.PersistBLSToExecutionChanges(blsToExecutionChanges)
	if err != nil {
		log.Errorf("error persisting bls to execution changes: %s", err.Error())
	}
}

func (s *ChainAnalyzer) processWithdrawals(block *spec.AgnosticBlock) {
	var withdrawals []spec.Withdrawal
	for _, item := range block.ExecutionPayload.Withdrawals {
		withdrawals = append(withdrawals, spec.Withdrawal{
			Slot:           block.Slot,
			Index:          item.Index,
			ValidatorIndex: item.ValidatorIndex,
			Address:        item.Address,
			Amount:         item.Amount,
		})
	}

	err := s.dbClient.PersistWithdrawals(withdrawals)
	if err != nil {
		log.Errorf("error persisting withdrawals: %s", err.Error())
	}

}

func (s *ChainAnalyzer) processTransactions(block *spec.AgnosticBlock, receipts []*types.Receipt) error {

	txs, err := spec.ParseTransactionsFromBlock(*block, receipts)
	if err != nil {
		log.Errorf("error getting slot %d transactions: %s", block.Slot, err.Error())
		return err
	}
	block.ExecutionPayload.AgnosticTransactions = txs
	if len(txs) == 0 {
		return nil
	}
	err = s.dbClient.PersistTransactions(txs)
	if err != nil {
		log.Errorf("error persisting transactions: %s", err.Error())
	}
	return err
}

// recoverBlockReceipts retries fetching EL receipts for blocks where
// ProcessETH1Data failed to populate AgnosticTransactions.
// Called from processBlockRewards as a second chance before fee calculation.
// See https://github.com/migalabs/goteth/issues/251
func (s *ChainAnalyzer) recoverBlockReceipts(block *spec.AgnosticBlock) {
	if len(block.ExecutionPayload.AgnosticTransactions) > 0 {
		return // already populated
	}
	if len(block.ExecutionPayload.Transactions) == 0 {
		return // no transactions in block
	}
	if !s.metrics.Transactions {
		return // transactions metric not enabled
	}

	log.Warnf("slot %d: retrying receipt fetch for block reward calculation", block.Slot)
	receipts, err := s.cli.GetBlockReceipts(*block)
	if err != nil {
		log.Errorf("slot %d: receipt recovery failed: %s", block.Slot, err)
		return
	}
	txs, err := spec.ParseTransactionsFromBlock(*block, receipts)
	if err != nil {
		log.Errorf("slot %d: transaction parse failed during recovery: %s", block.Slot, err)
		return
	}
	block.ExecutionPayload.AgnosticTransactions = txs
	log.Infof("slot %d: recovered %d transaction receipts for fee calculation", block.Slot, len(txs))
}

func (s *ChainAnalyzer) processBlobSidecars(block *spec.AgnosticBlock) {
	var blobs []*spec.AgnosticBlobSidecar
	var err error

	if block.HardForkVersion >= eth2_client_spec.DataVersionFulu {
		blobs, err = s.cli.RequestFuluBlobs(block.Slot)
	} else {
		blobs, err = s.cli.RequestBlobSidecars(block.Slot)
	}

	if err != nil {
		log.Errorf("could not download blobs for slot %d: %s", block.Slot, err)
	}
	if len(blobs) > 0 {
		// The block's own transaction list, not AgnosticTransactions: the
		// latter is filtered by which EL receipts arrived, and a missing
		// receipt would cost this slot every one of its blob attributions.
		if err := spec.AssignTxHashes(blobs, block.ExecutionPayload.Transactions); err != nil {
			log.Errorf("slot %d: could not attribute blob sidecars to transactions: %s", block.Slot, err)
		}
		s.dbClient.PersistBlobSidecars(blobs)
	}
}
