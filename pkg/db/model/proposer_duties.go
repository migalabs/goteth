package model

import "github.com/attestantio/go-eth2-client/spec/phase0"

type ProposerDuty struct {
	ValIdx       phase0.ValidatorIndex
	ProposerSlot phase0.Slot
	Proposed     bool
}

func (f ProposerDuty) InsertOp() bool {
	return true
}

func (f ProposerDuty) DropOp() bool {
	return false
}
