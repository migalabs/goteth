package state_metrics

import (
	"fmt"

	"github.com/attestantio/go-eth2-client/http"
	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/altair"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/cortze/eth-cl-state-analyzer/pkg/db/model"
	"github.com/cortze/eth-cl-state-analyzer/pkg/state_metrics/fork_state"
	"github.com/cortze/eth-cl-state-analyzer/pkg/utils"
)

type StateMetricsBase struct {
	CurrentState fork_state.ForkStateContentBase
	PrevState    fork_state.ForkStateContentBase
	NextState    fork_state.ForkStateContentBase
}

func (p StateMetricsBase) EpochReward(valIdx phase0.ValidatorIndex) int64 {
	if valIdx < phase0.ValidatorIndex(len(p.CurrentState.Balances)) && valIdx < phase0.ValidatorIndex(len(p.NextState.Balances)) {
		return int64(p.NextState.Balances[valIdx]) - int64(p.CurrentState.Balances[valIdx])
	}

	return 0

}

type StateMetrics interface {
	GetMetricsBase() StateMetricsBase
	GetMaxReward(valIdx phase0.ValidatorIndex) (model.ValidatorRewards, error)
	// keep in mind that att rewards for epoch 10 can be seen at beginning of epoch 12,
	// after state_transition
	// https://notes.ethereum.org/@vbuterin/Sys3GLJbD#Epoch-processing
}

func StateMetricsByForkVersion(nextBstate fork_state.ForkStateContentBase, bstate fork_state.ForkStateContentBase, prevBstate fork_state.ForkStateContentBase, iApi *http.Service) (StateMetrics, error) {
	switch bstate.Version {

	case spec.DataVersionPhase0:
		return NewPhase0Metrics(nextBstate, bstate, prevBstate), nil

	case spec.DataVersionAltair:
		return NewAltairMetrics(nextBstate, bstate, prevBstate), nil

	case spec.DataVersionBellatrix:
		return NewAltairMetrics(nextBstate, bstate, prevBstate), nil // We use Altair as Rewards system is the same

	case spec.DataVersionCapella:
		return NewAltairMetrics(nextBstate, bstate, prevBstate), nil // We use Altair as Rewards system is the same
	default:
		return nil, fmt.Errorf("could not figure out the State Metrics Fork Version: %s", bstate.Version)
	}
}

func (s StateMetricsBase) ExportToEpoch() model.Epoch {
	return model.Epoch{
		Epoch:                 s.CurrentState.Epoch,
		Slot:                  s.CurrentState.Slot,
		NumAttestations:       len(s.NextState.PrevAttestations),
		NumAttValidators:      int(s.NextState.NumAttestingVals),
		NumValidators:         int(s.CurrentState.NumActiveVals),
		TotalBalance:          float32(s.CurrentState.TotalActiveRealBalance) / float32(utils.EFFECTIVE_BALANCE_INCREMENT),
		AttEffectiveBalance:   float32(s.NextState.AttestingBalance[altair.TimelyTargetFlagIndex]) / float32(utils.EFFECTIVE_BALANCE_INCREMENT), // as per BEaconcha.in
		TotalEffectiveBalance: float32(s.CurrentState.TotalActiveBalance) / float32(utils.EFFECTIVE_BALANCE_INCREMENT),
		MissingSource:         int(s.NextState.GetMissingFlagCount(int(altair.TimelySourceFlagIndex))),
		MissingTarget:         int(s.NextState.GetMissingFlagCount(int(altair.TimelyTargetFlagIndex))),
		MissingHead:           int(s.NextState.GetMissingFlagCount(int(altair.TimelyHeadFlagIndex)))}
}
