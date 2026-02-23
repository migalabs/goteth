package analyzer

import (
	"context"
	"fmt"
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

func (s *ChainCache) AddNewState(ctx context.Context, newState *spec.AgnosticState) error {

	if newState == nil {
		return fmt.Errorf("cannot add nil state to cache")
	}

	blockList := make([]*spec.AgnosticBlock, 0)
	epochStartSlot := phase0.Slot(newState.Epoch * spec.SlotsPerEpoch)
	epochEndSlot := phase0.Slot((newState.Epoch+1)*spec.SlotsPerEpoch - 1)

	for i := epochStartSlot; i <= epochEndSlot; i++ {
		block, err := s.BlockHistory.Wait(ctx, SlotTo[uint64](i))
		if err != nil {
			return fmt.Errorf("waiting for block at slot %d: %w", i, err)
		}

		blockList = append(blockList, block)
	}

	// the 32 blocks were retrieved
	newState.AddBlocks(blockList)

	s.StateHistory.Set(EpochTo[uint64](newState.Epoch), newState)
	log.Debugf("state at slot %d successfully added to the queue", newState.Slot)
	return nil
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

func (s *ChainCache) GetHeadBlock() spec.AgnosticBlock {
	s.Lock()
	defer s.Unlock()
	return *s.HeadBlock
}

// RefreshStateBlocks re-reads blocks from BlockHistory and updates the
// state's Blocks array. This is needed after reorgs where block objects in
// BlockHistory have been replaced but the state still holds pointers to the
// old (pre-reorg) blocks. Without this, metrics that depend on block data
// (e.g. isFlagPossible for head reward) would use stale Proposed flags.
func (s *ChainCache) RefreshStateBlocks(ctx context.Context, epoch uint64) error {
	state, err := s.StateHistory.Wait(ctx, epoch)
	if err != nil {
		return fmt.Errorf("waiting for state at epoch %d: %w", epoch, err)
	}

	blockList := make([]*spec.AgnosticBlock, 0)
	epochStartSlot := phase0.Slot(epoch * spec.SlotsPerEpoch)
	epochEndSlot := phase0.Slot((epoch+1)*spec.SlotsPerEpoch - 1)

	for i := epochStartSlot; i <= epochEndSlot; i++ {
		block, err := s.BlockHistory.Wait(ctx, SlotTo[uint64](i))
		if err != nil {
			return fmt.Errorf("waiting for block at slot %d: %w", i, err)
		}
		blockList = append(blockList, block)
	}

	state.RefreshBlocks(blockList)
	return nil
}

func (s *ChainCache) CleanUpTo(maxSlot phase0.Slot) {

	stateKeys := s.StateHistory.GetKeyList()

	// Delete from History

	for _, epoch := range stateKeys {
		if (epoch * spec.SlotsPerEpoch) >= uint64(maxSlot) {
			continue // only process epochs that are before the maxSlot
		}

		s.StateHistory.Delete(epoch)
		// loop over slots in the epoch
		for slot := (epoch * spec.SlotsPerEpoch); slot < ((epoch + 1) * spec.SlotsPerEpoch); slot++ {
			s.BlockHistory.Delete(slot)
		}
	}

}
