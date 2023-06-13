package metrics

import (
	"fmt"

	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/altair"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	local_spec "github.com/cortze/eth-cl-state-analyzer/pkg/spec"
)

type StateMetricsBase struct {
	FirstState  local_spec.AgnosticState // from where attestations are measured
	SecondState local_spec.AgnosticState // not used for now
	ThirdState  local_spec.AgnosticState // at which metrics are written
	FourthState local_spec.AgnosticState // to measure prev attestations in thirdstate
}

func (p StateMetricsBase) EpochReward(valIdx phase0.ValidatorIndex) int64 {
	// If the validator exists in the second state
	// and the validator exists in the third state
	// Just t avoid any panic when getting the values from the slices
	if valIdx < phase0.ValidatorIndex(len(p.SecondState.Balances)) && valIdx < phase0.ValidatorIndex(len(p.ThirdState.Balances)) {
		return int64(p.ThirdState.Balances[valIdx]) - int64(p.SecondState.Balances[valIdx])
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
	first local_spec.AgnosticState,
	second local_spec.AgnosticState,
	third local_spec.AgnosticState,
	fourth local_spec.AgnosticState) (StateMetrics, error) {
	switch third.Version {

	case spec.DataVersionPhase0:
		return NewPhase0Metrics(first, second, third, fourth), nil

	case spec.DataVersionAltair:
		return NewAltairMetrics(first, second, third, fourth), nil

	case spec.DataVersionBellatrix:
		return NewAltairMetrics(first, second, third, fourth), nil // We use Altair as Rewards system is the same

	case spec.DataVersionCapella:
		return NewAltairMetrics(first, second, third, fourth), nil // We use Altair as Rewards system is the same
	default:
		return nil, fmt.Errorf("could not figure out the State Metrics Fork Version: %s", third.Version)
	}
}

func (s StateMetricsBase) ExportToEpoch() local_spec.Epoch {
	return local_spec.Epoch{
		Epoch:                 s.ThirdState.Epoch,
		Slot:                  s.ThirdState.Slot,
		NumAttestations:       len(s.FourthState.PrevAttestations),
		NumAttValidators:      int(s.FourthState.NumAttestingVals),
		NumValidators:         int(s.ThirdState.NumActiveVals),
		TotalBalance:          float32(s.ThirdState.TotalActiveRealBalance) / float32(local_spec.EffectiveBalanceInc),
		AttEffectiveBalance:   float32(s.FourthState.AttestingBalance[altair.TimelyTargetFlagIndex]) / float32(local_spec.EffectiveBalanceInc), // as per BEaconcha.in
		TotalEffectiveBalance: float32(s.ThirdState.TotalActiveBalance) / float32(local_spec.EffectiveBalanceInc),
		MissingSource:         int(s.FourthState.GetMissingFlagCount(int(altair.TimelySourceFlagIndex))),
		MissingTarget:         int(s.FourthState.GetMissingFlagCount(int(altair.TimelyTargetFlagIndex))),
		MissingHead:           int(s.FourthState.GetMissingFlagCount(int(altair.TimelyHeadFlagIndex)))}
}
