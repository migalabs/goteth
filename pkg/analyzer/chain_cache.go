package analyzer

import (
	"sync"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/migalabs/goteth/pkg/spec"
)

type ChainCache struct {
	StateHistory *AgnosticMap[spec.AgnosticState]
	BlockHistory *AgnosticMap[spec.AgnosticBlock] // Here we will store stateroots from the blocks

	sync.Mutex
	HeadBlock       *spec.AgnosticBlock
	LatestFinalized *spec.AgnosticBlock
}

func NewQueue() ChainCache {
	return ChainCache{
		StateHistory: NewAgnosticMap[spec.AgnosticState](),
		BlockHistory: NewAgnosticMap[spec.AgnosticBlock](),
	}
}

func (s *ChainCache) AddNewState(newState *spec.AgnosticState) {

	blockList := make([]spec.AgnosticBlock, 0)
	epochStartSlot := phase0.Slot(newState.Epoch * spec.SlotsPerEpoch)
	epochEndSlot := phase0.Slot((newState.Epoch+1)*spec.SlotsPerEpoch - 1)

	for i := epochStartSlot; i <= epochEndSlot; i++ {
		block := s.BlockHistory.Wait(SlotTo[uint64](i))

		blockList = append(blockList, *block)
	}

	// the 32 blocks were retrieved
	newState.AddBlocks(blockList)

	s.StateHistory.Set(EpochTo[uint64](newState.Epoch), newState)
	log.Debugf("state at slot %d successfully added to the queue", newState.Slot)
}

func (s *ChainCache) AddNewBlock(block *spec.AgnosticBlock) {

	keys := s.BlockHistory.GetKeyList()

	s.BlockHistory.Set(SlotTo[uint64](block.Slot), block)
	log.Tracef("block at slot %d successfully added to the queue", block.Slot)

	for _, key := range keys {
		if key >= uint64(block.Slot) { // if there is any key greater than the current evaluated block
			return // no more tasks
		}
	}

	// if we are here, the new block is greater than the rest
	s.Lock()
	s.HeadBlock = block
	s.Unlock()

}

func (s *ChainCache) CleanUpTo(maxSlot phase0.Slot) {

	stateKeys := s.StateHistory.GetKeyList()

	// Delete from History

	for _, epoch := range stateKeys {
		if epoch >= uint64(maxSlot) {
			continue // only process epochs that are before the finalized
		}

		s.StateHistory.Delete(epoch)
		// loop over slots in the epoch
		for slot := (epoch * spec.SlotsPerEpoch); slot < ((epoch + 1) * spec.SlotsPerEpoch); slot++ {
			s.BlockHistory.Delete(slot)
		}
	}

}
