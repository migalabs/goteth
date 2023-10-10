package analyzer

import (
	"fmt"

	"github.com/attestantio/go-eth2-client/spec/bellatrix"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/migalabs/goteth/pkg/spec"
)

func (s *ChainAnalyzer) ProcessBlock(slot phase0.Slot) {
	if !s.metrics.Block {
		return
	}
	routineKey := "slot=" + fmt.Sprintf("%d", slot)
	s.processerBook.Acquire(routineKey)

	block := s.queue.BlockHistory.Wait(slot)
	s.dbClient.Persist(block)

	if s.metrics.Transactions {
		s.processTransactions(block)
	}
	s.processerBook.FreePage(routineKey)
}

func (s ChainAnalyzer) processTransactions(block spec.AgnosticBlock) {

	for idx, tx := range block.ExecutionPayload.Transactions {
		go func(txID int, transaction bellatrix.Transaction) {
			detailedTx, err := s.cli.RequestTransactionDetails(
				transaction,
				block.Slot,
				block.ExecutionPayload.BlockNumber,
				block.ExecutionPayload.Timestamp)
			if err != nil {
				log.Errorf("could not request transaction details in slot %s for transaction %d: %s", block.Slot, txID, err)
			}
			log.Tracef("persisting transaction metrics: slot %d, tx number: %d", block.Slot, txID)
			s.dbClient.Persist(detailedTx)
		}(idx, tx)

	}
}
