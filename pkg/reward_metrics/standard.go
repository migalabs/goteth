package reward_metrics

import (
	"fmt"

	"github.com/attestantio/go-eth2-client/http"
	"github.com/attestantio/go-eth2-client/spec"
	"github.com/cortze/eth2-state-analyzer/pkg/reward_metrics/fork_state"
)

type StateMetricsBase struct {
	CurrentState fork_state.ForkStateContentBase
	PrevState    fork_state.ForkStateContentBase
	NextState    fork_state.ForkStateContentBase
}

func (p StateMetricsBase) EpochReward(valIdx uint64) int64 {
	if valIdx < uint64(len(p.CurrentState.Balances)) && valIdx < uint64(len(p.NextState.Balances)) {
		return int64(p.NextState.Balances[valIdx]) - int64(p.CurrentState.Balances[valIdx])
	}

	return 0

}

func (p StateMetricsBase) GetAttSlot(valIdx uint64) uint64 {

	return p.PrevState.EpochStructs.ValidatorAttSlot[valIdx]
}

func (p StateMetricsBase) GetAttInclusionSlot(valIdx uint64) int64 {

	for i, item := range p.CurrentState.ValAttestationInclusion[valIdx].AttestedSlot {
		// we are looking for a vote to the previous epoch
		if item >= p.PrevState.Slot+1-fork_state.SLOTS_PER_EPOCH &&
			item <= p.PrevState.Slot {
			return int64(p.CurrentState.ValAttestationInclusion[valIdx].InclusionSlot[i])
		}
	}
	return int64(-1)
}

type StateMetrics interface {
	GetMetricsBase() StateMetricsBase
	GetMaxReward(valIdx uint64) (ValidatorSepRewards, error)
}

func StateMetricsByForkVersion(nextBstate fork_state.ForkStateContentBase, bstate fork_state.ForkStateContentBase, prevBstate fork_state.ForkStateContentBase, iApi *http.Service) (StateMetrics, error) {
	switch bstate.Version {

	case spec.DataVersionPhase0:
		return NewPhase0Metrics(nextBstate, bstate, prevBstate), nil

	case spec.DataVersionAltair:
		return NewAltairMetrics(nextBstate, bstate, prevBstate), nil

	case spec.DataVersionBellatrix:
		return NewAltairMetrics(nextBstate, bstate, prevBstate), nil // We use Altair as Rewards system is the same
	default:
		return nil, fmt.Errorf("could not figure out the Beacon State Fork Version: %s", bstate.Version)
	}
}

type ValidatorSepRewards struct {
	Attestation     uint64
	InclusionDelay  uint64
	FlagIndex       uint64
	SyncCommittee   uint64
	MaxReward       uint64
	BaseReward      uint64
	InSyncCommittee bool
	ProposerSlot    int64
}
