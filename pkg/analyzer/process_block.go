package analyzer

import (
	"fmt"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/migalabs/goteth/pkg/db"
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

	var blocks db.InsertBlocks
	var err error

	block := s.downloadCache.BlockHistory.Wait(SlotTo[uint64](slot))
	blocks.Append(*block)

	err = s.dbClient.Persist(blocks)
	if err != nil {
		log.Fatalf("error persisting blocks: %s", err.Error())
	}

	var withdrawals db.InsertWithdrawals
	for _, item := range block.ExecutionPayload.Withdrawals {
		withdrawals.Append(spec.Withdrawal{
			Slot:           block.Slot,
			Index:          item.Index,
			ValidatorIndex: item.ValidatorIndex,
			Address:        item.Address,
			Amount:         item.Amount,
		})
	}

	err = s.dbClient.Persist(withdrawals)
	if err != nil {
		log.Errorf("error persisting withdrawals: %s", err.Error())
	}

	if s.metrics.Transactions {
		s.processTransactions(block)
	}
	s.processerBook.FreePage(routineKey)
}

func (s *ChainAnalyzer) processTransactions(block *spec.AgnosticBlock) {

	var err error
	var transactions db.InsertTransactions
	for idx, tx := range block.ExecutionPayload.Transactions {
		detailedTx, err := s.cli.RequestTransactionDetails(
			tx,
			block.Slot,
			block.ExecutionPayload.BlockNumber,
			block.ExecutionPayload.Timestamp)
		if err != nil {
			log.Errorf("could not request transaction details in slot %d for transaction %d: %s", block.Slot, idx, err)
		}

		transactions.Append(*detailedTx)
	}
	log.Tracef("persisting transaction metrics: slot %d", block.Slot)
	err = s.dbClient.Persist(transactions)
	if err != nil {
		log.Fatalf("error persisting transactions: %s", err.Error())
	}
}
