package analyzer

import (
	"time"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/cortze/eth-cl-state-analyzer/pkg/spec"
	"github.com/sirupsen/logrus"
)

const (
	ValidatorSetSize           = 500000                 // Estimation of current number of validators, used for channel length declaration
	maxWorkers                 = 50                     // maximum number of workers allowed in the tool
	minBlockReqTime            = 100 * time.Millisecond // max 10 queries per second, dont spam beacon node
	minStateReqTime            = 1 * time.Second        // max 1 query per second, dont spam beacon node
	epochsToFinalizedTentative = 3                      // usually, 3 full epochs before the head it is finalized
)

var (
	log = logrus.WithField(
		"module", "analyzer",
	)
)

type SlotRoot struct {
	Slot      phase0.Slot
	Epoch     phase0.Epoch
	StateRoot phase0.Root
}

type StateQueue struct {
	prevState       spec.AgnosticState
	currentState    spec.AgnosticState
	nextState       spec.AgnosticState
	Roots           []SlotRoot
	LatestFinalized SlotRoot
}

func NewStateQueue(finalizedSlot phase0.Slot, finalizedRoot phase0.Root) StateQueue {
	return StateQueue{
		prevState:    spec.AgnosticState{},
		currentState: spec.AgnosticState{},
		nextState:    spec.AgnosticState{},
		Roots:        make([]SlotRoot, 0),
		LatestFinalized: SlotRoot{
			Slot:      finalizedSlot,
			Epoch:     phase0.Epoch(finalizedSlot / spec.SlotsPerEpoch),
			StateRoot: finalizedRoot,
		},
	}
}

func (s *StateQueue) AddNewState(newState spec.AgnosticState) {

	if s.nextState.Epoch != phase0.Epoch(0) && newState.Epoch != s.nextState.Epoch+1 {
		log.Panicf("state at epoch %d is not consecutive to %d...", newState.Epoch, s.nextState.Epoch)
	}

	s.prevState = s.currentState
	s.currentState = s.nextState
	s.nextState = newState

	s.AddRoot(newState.Slot, newState.StateRoot)
}

func (s StateQueue) Complete() bool {
	emptyRoot := phase0.Root{}
	if s.prevState.StateRoot != emptyRoot {
		return true
	}
	return false
}

func (s *StateQueue) AddRoot(iSlot phase0.Slot, iRoot phase0.Root) {
	s.Roots = append(s.Roots, SlotRoot{
		Slot:      iSlot,
		Epoch:     phase0.Epoch(iSlot / spec.SlotsPerEpoch),
		StateRoot: iRoot,
	})
}

func (s *StateQueue) CheckFinalized(iSlot phase0.Slot, iRoot phase0.Root) (phase0.Epoch, bool) {

	if s.LatestFinalized.Epoch == 0 {
		// it has not been configured yet
		s.LatestFinalized = s.Roots[0] // the first position of our history should be the latest finalized
	}

	// SlotRoots are ordered ascending always
	for i, slotRoot := range s.Roots {
		if slotRoot.Slot == iSlot { // found it in our history
			if slotRoot.StateRoot == iRoot { // the root matches, finalized ok
				s.Roots = s.Roots[i+1:] // remove all roots before this one, they are ordered asc

				s.LatestFinalized = slotRoot
				log.Infof("finalized checkpoint at epoch %d successfully verified...", slotRoot.Epoch)
				return slotRoot.Epoch, true
			} else { // we found the slot in the history, but the root does not match
				log.Errorf("the finalized checkpoint was not verfied, probably a reorg happened...")
				log.Errorf("rewinding to epoch %d", s.LatestFinalized.Epoch-2)
				return s.LatestFinalized.Epoch - 2, false // go 2 epochs before the finalized

			}
		}
	}
	// the slot does not exist in our history
	// continue as normal

	return s.LatestFinalized.Epoch, true

}

// Used for the block routine
// Somehow similar to the above
// If we merge both routines then logically we would also join to the above
type SlotRootHistory struct {
	Roots []SlotRoot
}

func NewSlotHistory() SlotRootHistory {
	return SlotRootHistory{
		Roots: make([]SlotRoot, 0),
	}
}

func (s *SlotRootHistory) AddRoot(iSlot phase0.Slot, iRoot phase0.Root) {
	s.Roots = append(s.Roots, SlotRoot{
		Slot:      iSlot,
		Epoch:     phase0.Epoch(iSlot / spec.SlotsPerEpoch),
		StateRoot: iRoot,
	})
}

// Returns whether the finalized root was ok
// If not, it returns the first slot of the history
// That slot would be the first one unverified from our history
func (s *SlotRootHistory) CheckFinalized(iSlot phase0.Slot, iRoot phase0.Root) (bool, phase0.Slot) {
	for i, root := range s.Roots {
		if iSlot == root.Slot { // if slot in in the history
			if root.StateRoot == iRoot { // if root matches
				s.Roots = s.Roots[i+1:] // clean history up to verified root
				log.Infof("checkpoint for slot %d verified...", iSlot)
				return true, iSlot
			}
			// slot in history, but root not matched, clean all the history
			slot := s.Roots[0].Slot
			s.Roots = make([]SlotRoot, 0)
			log.Infof("checkpoint for slot %d was not verified, rewinding...", iSlot)
			return false, slot
		}
	}
	return true, iSlot // slot is not in the history
}
