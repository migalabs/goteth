package metrics

import (
	"fmt"

	"github.com/attestantio/go-eth2-client/http"
	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/altair"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	local_spec "github.com/migalabs/goteth/pkg/spec"
)

type StateMetricsBase struct {
	PrevState    *local_spec.AgnosticState
	CurrentState *local_spec.AgnosticState
	NextState    *local_spec.AgnosticState
}

func (p StateMetricsBase) EpochReward(valIdx phase0.ValidatorIndex) int64 {
	if valIdx < phase0.ValidatorIndex(len(p.CurrentState.Balances)) && valIdx < phase0.ValidatorIndex(len(p.NextState.Balances)) {
		return int64(p.NextState.Balances[valIdx]) - int64(p.CurrentState.Balances[valIdx])
	}

	return 0

}

type StateMetrics interface {
	GetMetricsBase() StateMetricsBase
	GetMaxReward(valIdx phase0.ValidatorIndex) (local_spec.ValidatorRewards, error)
	// keep in mind that att rewards for epoch 10 can be seen at beginning of epoch 12,
	// after state_transition
	// https://notes.ethereum.org/@vbuterin/Sys3GLJbD#Epoch-processing
}

func StateMetricsByForkVersion(
	nextBstate *local_spec.AgnosticState,
	bstate *local_spec.AgnosticState,
	prevBstate *local_spec.AgnosticState,
	iApi *http.Service) (StateMetrics, error) {
	switch nextBstate.Version { // rewards are written at nextState epoch

	case spec.DataVersionPhase0:
		return NewPhase0Metrics(nextBstate, bstate, prevBstate), nil

	case spec.DataVersionAltair:
		return NewAltairMetrics(nextBstate, bstate, prevBstate), nil

	case spec.DataVersionBellatrix:
		return NewAltairMetrics(nextBstate, bstate, prevBstate), nil // We use Altair as Rewards system is the same

	case spec.DataVersionCapella:
		return NewAltairMetrics(nextBstate, bstate, prevBstate), nil // We use Altair as Rewards system is the same

	case spec.DataVersionDeneb:
		return NewAltairMetrics(nextBstate, bstate, prevBstate), nil // We use Altair as Rewards system is the same
	default:
		return nil, fmt.Errorf("could not figure out the State Metrics Fork Version: %s", bstate.Version)
	}
}

func (s StateMetricsBase) ExportToEpoch() local_spec.Epoch {

	return local_spec.Epoch{
		Epoch:                 s.CurrentState.Epoch,
		Slot:                  s.CurrentState.Slot,
		NumAttestations:       len(s.NextState.PrevAttestations),
		NumAttValidators:      int(s.NextState.NumAttestingVals),
		NumValidators:         len(s.CurrentState.Validators),
		TotalBalance:          float32(s.CurrentState.TotalActiveRealBalance) / float32(local_spec.EffectiveBalanceInc),
		AttEffectiveBalance:   float32(s.NextState.AttestingBalance[altair.TimelyTargetFlagIndex]) / float32(local_spec.EffectiveBalanceInc), // as per BEaconcha.in
		TotalEffectiveBalance: float32(s.CurrentState.TotalActiveBalance) / float32(local_spec.EffectiveBalanceInc),
		MissingSource:         int(s.NextState.GetMissingFlagCount(int(altair.TimelySourceFlagIndex))),
		MissingTarget:         int(s.NextState.GetMissingFlagCount(int(altair.TimelyTargetFlagIndex))),
		MissingHead:           int(s.NextState.GetMissingFlagCount(int(altair.TimelyHeadFlagIndex))),
		Timestamp:             int64(s.CurrentState.GenesisTimestamp + uint64(s.CurrentState.Epoch)*local_spec.SlotsPerEpoch*local_spec.SlotSeconds),
		NumSlashed:            int(s.CurrentState.NumSlashedVals),
		NumActive:             int(s.CurrentState.NumActiveVals),
		NumExit:               int(s.CurrentState.NumExitedVals),
		NumInActivation:       int(s.CurrentState.NumQueuedVals),
	}
}
