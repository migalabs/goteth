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

	block := s.downloadCache.BlockHistory.Wait(SlotTo[uint64](slot))
	err := s.dbClient.PersistBlocks([]spec.AgnosticBlock{*block})
	if err != nil {
		log.Errorf("error persisting blocks: %s", err.Error())
	}

	s.processWithdrawals(block)

	if s.metrics.Transactions {
		s.ProcessETH1Data(block)
	}
	s.processBLSToExecutionChanges(block)
	s.processDeposits(block)
	s.processerBook.FreePage(routineKey)
}

func (s *ChainAnalyzer) ProcessETH1Data(block *spec.AgnosticBlock) {
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
	if block.HardForkVersion >= eth2_client_spec.DataVersionDeneb {
		s.processBlobSidecars(block, block.ExecutionPayload.AgnosticTransactions)
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

func (s *ChainAnalyzer) processBlobSidecars(block *spec.AgnosticBlock, txs []spec.AgnosticTransaction) {
	blobs, err := s.cli.RequestBlobSidecars(block.Slot)
	if err != nil {
		log.Fatalf("could not download blob sidecars for slot %d: %s", block.Slot, err)
	}
	if len(blobs) > 0 {
		for _, blob := range blobs {
			blob.GetTxHash(txs)
		}
		s.dbClient.PersistBlobSidecars(blobs)
	}
}
