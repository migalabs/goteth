package analyzer

import (
	"fmt"

	"github.com/attestantio/go-eth2-client/spec/phase0"
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
		s.processTransactions(block)
		s.processBlobSidecars(block, block.ExecutionPayload.AgnosticTransactions)
	}

	s.processSlashings(block)

	s.processerBook.FreePage(routineKey)
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

func (s *ChainAnalyzer) processTransactions(block *spec.AgnosticBlock) {

	txs, err := s.cli.GetBlockTransactions(*block)
	if err != nil {
		log.Errorf("error getting slot %d transactions: %s", block.Slot, err.Error())
	}
	block.ExecutionPayload.AgnosticTransactions = txs

	err = s.dbClient.PersistTransactions(txs)
	if err != nil {
		log.Errorf("error persisting transactions: %s", err.Error())
	}

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

func (s *ChainAnalyzer) processSlashings(block *spec.AgnosticBlock) {

	slashings := make([]spec.AgnosticSlashing, 0)

	for _, proposerSlashing := range block.ProposerSlashings {
		slashings = append(slashings, spec.AgnosticSlashing{
			SlashedValidator: proposerSlashing.SignedHeader1.Message.ProposerIndex,
			SlashedBy:        block.ProposerIndex,
			SlashingReason:   spec.SlashingReasonProposerSlashing,
			Slot:             block.Slot,
			Epoch:            spec.EpochAtSlot(block.Slot),
		})
	}

	for _, attesterSlashing := range block.AttesterSlashings {

		slashedValidatorsIdxs := spec.SlashingIntersection(attesterSlashing.Attestation1.AttestingIndices, attesterSlashing.Attestation2.AttestingIndices)
		for _, idx := range slashedValidatorsIdxs {
			slashings = append(slashings, spec.AgnosticSlashing{
				SlashedValidator: idx,
				SlashedBy:        block.ProposerIndex,
				SlashingReason:   spec.SlashingReasonAttesterSlashing,
				Slot:             block.Slot,
				Epoch:            spec.EpochAtSlot(block.Slot),
			})
		}
	}

	if len(slashings) == 0 {
		return
	}
	err := s.dbClient.PersistSlashings(slashings)
	if err != nil {
		log.Errorf("error persisting slashings: %s", err.Error())
	}

}
