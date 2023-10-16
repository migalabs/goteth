package analyzer

import (
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/migalabs/goteth/pkg/spec"
)

type Queue struct {
	StateHistory    *AgnosticMap[spec.AgnosticState]
	BlockHistory    *AgnosticMap[spec.AgnosticBlock] // Here we will store stateroots from the blocks
	HeadBlock       spec.AgnosticBlock
	LatestFinalized spec.AgnosticBlock
}

func NewQueue() Queue {
	return Queue{
		StateHistory: NewAgnosticMap[spec.AgnosticState](),
		BlockHistory: NewAgnosticMap[spec.AgnosticBlock](),
	}
}

func (s *Queue) AddNewState(newState spec.AgnosticState) {

	blockList := make([]spec.AgnosticBlock, 0)
	epochStartSlot := phase0.Slot(newState.Epoch * spec.SlotsPerEpoch)
	epochEndSlot := phase0.Slot((newState.Epoch+1)*spec.SlotsPerEpoch - 1)

	for i := epochStartSlot; i <= epochEndSlot; i++ {
		block := s.BlockHistory.Wait(SlotTo[uint64](i))

		blockList = append(blockList, block)
	}

	// the 32 blocks were retrieved
	newState.AddBlocks(blockList)

	s.StateHistory.Set(EpochTo[uint64](newState.Epoch), newState)
}

func (s *Queue) AddNewBlock(block spec.AgnosticBlock) {

	s.BlockHistory.Set(SlotTo[uint64](block.Slot), block)
}

// // Advances the finalized checkpoint to the given slot
// func (s *Queue) AdvanceFinalized(slot phase0.Slot) {

// 	for i := s.LatestFinalized.Slot; i < slot; i++ {
// 		s.BlockHistory.Delete(i)
// 		s.LatestFinalized = s.BlockHistory.Wait(i + 1)
// 	}
// }
