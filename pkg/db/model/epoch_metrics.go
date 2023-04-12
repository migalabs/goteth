package model

import (
	"github.com/attestantio/go-eth2-client/spec/phase0"
)

var (
	EpochModelOps = map[string]bool{
		INSERT_OP: true,
		DROP_OP:   false,
	}
)

// TODO: review
type Epoch struct {
	Epoch                 phase0.Epoch
	Slot                  phase0.Slot
	NumAttestations       int
	NumAttValidators      int
	NumValidators         int
	TotalBalance          float32
	AttEffectiveBalance   float32
	TotalEffectiveBalance float32
	MissingSource         int
	MissingTarget         int
	MissingHead           int
}

func (f Epoch) InsertOp() bool {
	return true
}

func (f Epoch) DropOp() bool {
	return false
}

// func NewEpochMetrics(input state_metrics.StateMetricsBase) Epoch {

// 	return Epoch{
// 		Epoch:                 input.CurrentState.Epoch,
// 		Slot:                  input.CurrentState.Slot,
// 		NumAttestations:       len(input.NextState.PrevAttestations),
// 		NumAttValidators:      int(input.NextState.NumAttestingVals),
// 		NumValidators:         int(input.CurrentState.NumActiveVals),
// 		TotalBalance:          float32(input.CurrentState.TotalActiveRealBalance) / float32(utils.EFFECTIVE_BALANCE_INCREMENT),
// 		AttEffectiveBalance:   float32(input.NextState.AttestingBalance[altair.TimelyTargetFlagIndex]) / float32(utils.EFFECTIVE_BALANCE_INCREMENT), // as per BEaconcha.in
// 		TotalEffectiveBalance: float32(input.CurrentState.TotalActiveBalance) / float32(utils.EFFECTIVE_BALANCE_INCREMENT),
// 		MissingSource:         int(input.NextState.GetMissingFlagCount(int(altair.TimelySourceFlagIndex))),
// 		MissingTarget:         int(input.NextState.GetMissingFlagCount(int(altair.TimelyTargetFlagIndex))),
// 		MissingHead:           int(input.NextState.GetMissingFlagCount(int(altair.TimelyHeadFlagIndex)))}

// }
