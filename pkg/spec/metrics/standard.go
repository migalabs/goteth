package metrics

import (
	"fmt"

	"github.com/attestantio/go-eth2-client/http"
	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/altair"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	local_spec "github.com/migalabs/goteth/pkg/spec"
	"github.com/sirupsen/logrus"
)

var (
	log = logrus.WithField(
		"module", "spec_metrics",
	)
)

type StateMetricsBase struct {
	PrevState    *local_spec.AgnosticState
	CurrentState *local_spec.AgnosticState
	NextState    *local_spec.AgnosticState
	// these are the max rewards calculated by our tool
	MaxSlashingRewards      map[phase0.ValidatorIndex]phase0.Gwei // for now just proposer as per spec
	MaxBlockRewards         map[phase0.ValidatorIndex]phase0.Gwei // from including attestation and sync aggregates. In this case, not max reward but the actual reward
	InclusionDelays         map[phase0.ValidatorIndex]int         // from attestation inclusion delay
	MaxAttesterRewards      map[phase0.ValidatorIndex]phase0.Gwei // rewards from attesting
	CurrentNumAttestingVals []bool                                // array that marks whether each validator has attested or not
}

func (p StateMetricsBase) EpochReward(valIdx phase0.ValidatorIndex) int64 {
	if valIdx < phase0.ValidatorIndex(len(p.CurrentState.Balances)) && valIdx < phase0.ValidatorIndex(len(p.NextState.Balances)) {
		reward := int64(p.NextState.Balances[valIdx]) - int64(p.CurrentState.Balances[valIdx])
		reward += int64(p.NextState.Withdrawals[valIdx])
		reward -= int64(p.NextState.Deposits[valIdx])
		return reward
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
	nextState *local_spec.AgnosticState,
	currentState *local_spec.AgnosticState,
	prevState *local_spec.AgnosticState,
	iApi *http.Service) (StateMetrics, error) {
	switch nextState.Version { // rewards are written at nextState epoch

	case spec.DataVersionPhase0:
		return NewPhase0Metrics(nextState, currentState, prevState), nil

	case spec.DataVersionAltair:
		return NewAltairMetrics(nextState, currentState, prevState), nil

	case spec.DataVersionBellatrix:
		return NewAltairMetrics(nextState, currentState, prevState), nil // We use Altair as Rewards system is the same

	case spec.DataVersionCapella:
		return NewAltairMetrics(nextState, currentState, prevState), nil // We use Altair as Rewards system is the same

	case spec.DataVersionDeneb:
		return NewDenebMetrics(nextState, currentState, prevState), nil
	default:
		return nil, fmt.Errorf("could not figure out the State Metrics Fork Version: %s", currentState.Version)
	}
}

func (s StateMetricsBase) ExportToEpoch() local_spec.Epoch {

	return local_spec.Epoch{
		Epoch:                 s.CurrentState.Epoch,
		Slot:                  s.CurrentState.Slot,
		NumAttestations:       len(s.NextState.PrevAttestations),
		NumAttValidators:      int(countTrue(s.CurrentNumAttestingVals)),
		NumValidators:         len(s.CurrentState.Validators),
		TotalBalance:          float32(s.CurrentState.TotalActiveRealBalance) / float32(local_spec.EffectiveBalanceInc),
		AttEffectiveBalance:   float32(s.NextState.AttestingBalance[altair.TimelyTargetFlagIndex]) / float32(local_spec.EffectiveBalanceInc), // as per BEaconcha.in
		TotalEffectiveBalance: float32(s.CurrentState.TotalActiveBalance) / float32(local_spec.EffectiveBalanceInc),
		MissingSource:         int(s.NextState.GetMissingFlagCount(int(altair.TimelySourceFlagIndex))),
		MissingTarget:         int(s.NextState.GetMissingFlagCount(int(altair.TimelyTargetFlagIndex))),
		MissingHead:           int(s.NextState.GetMissingFlagCount(int(altair.TimelyHeadFlagIndex))),
		Timestamp:             int64(s.CurrentState.GenesisTimestamp + uint64(s.CurrentState.Epoch)*local_spec.SlotsPerEpoch*local_spec.SlotSeconds),
		NumSlashedVals:        int(s.CurrentState.NumSlashedVals),
		NumActiveVals:         int(s.CurrentState.NumActiveVals),
		NumExitedVals:         int(s.CurrentState.NumExitedVals),
		NumInActivationVals:   int(s.CurrentState.NumQueuedVals),
	}
}
