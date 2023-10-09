package analyzer

import (
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/migalabs/goteth/pkg/spec"
)

type Queue struct {
	StateHistory    *StatesMap
	BlockHistory    *BlocksMap // Here we will store stateroots from the blocks
	HeadBlock       spec.AgnosticBlock
	LatestFinalized spec.AgnosticBlock
}

func NewQueue() Queue {
	return Queue{
		StateHistory: &StatesMap{
			m:    make(map[phase0.Epoch]spec.AgnosticState),
			subs: make(map[phase0.Epoch][]chan spec.AgnosticState),
		},
		BlockHistory: &BlocksMap{
			m:    make(map[phase0.Slot]spec.AgnosticBlock),
			subs: make(map[phase0.Slot][]chan spec.AgnosticBlock),
		},
	}
}

func (s *Queue) AddNewState(newState spec.AgnosticState) {

	blockList := make([]spec.AgnosticBlock, 0)
	epochStartSlot := phase0.Slot(newState.Epoch * spec.SlotsPerEpoch)
	epochEndSlot := phase0.Slot((newState.Epoch+1)*spec.SlotsPerEpoch - 1)

	for i := epochStartSlot; i <= epochEndSlot; i++ {
		block := s.BlockHistory.Wait(i)

		blockList = append(blockList, block)
	}

	// the 32 blocks were retrieved
	newState.AddBlocks(blockList)

	s.StateHistory.Set(newState.Epoch, newState)
}

func (s *Queue) AddNewBlock(block spec.AgnosticBlock) {

	s.BlockHistory.Set(block.Slot, block)
}

// Advances the finalized checkpoint to the given slot
func (s *Queue) AdvanceFinalized(slot phase0.Slot) {

	for i := s.LatestFinalized.Slot; i < slot; i++ {
		s.BlockHistory.Delete(i)
		s.LatestFinalized = s.BlockHistory.Wait(i + 1)
	}
}
