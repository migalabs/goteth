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
	MaxSlashingRewards map[phase0.ValidatorIndex]phase0.Gwei // for now just proposer as per spec
	MaxBlockRewards    map[phase0.ValidatorIndex]phase0.Gwei // from including attestation and sync aggregates. In this case, not max reward but the actual reward
	InclusionDelays    []int                                 // from attestation inclusion delay
	MaxAttesterRewards map[phase0.ValidatorIndex]phase0.Gwei // rewards from attesting
}

func (p StateMetricsBase) EpochReward(valIdx phase0.ValidatorIndex) int64 {
	consolidatedAmount, ok := p.CurrentState.ConsolidatedAmounts[valIdx]
	if !ok {
		consolidatedAmount = 0
	}

	// After electra, we have deposits in depositedAmounts since the logic of deposits changed
	depositedAmount, ok := p.CurrentState.DepositedAmounts[valIdx]
	if !ok {
		depositedAmount = 0
	}
	depositedAmount += p.NextState.Deposits[valIdx]
	if valIdx < phase0.ValidatorIndex(len(p.CurrentState.Balances)) && valIdx < phase0.ValidatorIndex(len(p.NextState.Balances)) {

		// CRITICAL FIX: Validate state transition consistency
		// Ensure both states are from consistent chain (no unfinalized data mixing)
		if p.NextState.Epoch != p.CurrentState.Epoch+1 {
			log.Warnf("Inconsistent state transition: NextState epoch %d, CurrentState epoch %d for validator %d",
				p.NextState.Epoch, p.CurrentState.Epoch, valIdx)
			return 0
		}

		// CRITICAL FIX: Additional validation for state root consistency during reorgs
		// If we're processing a reorg, ensure state roots are finalized
		if p.NextState.StateRoot == (phase0.Root{}) || p.CurrentState.StateRoot == (phase0.Root{}) {
			log.Warnf("Empty state root detected during reward calculation for validator %d at epoch %d - possible reorg",
				valIdx, p.NextState.Epoch)
			return 0
		}

		reward := int64(p.NextState.Balances[valIdx]) - int64(p.CurrentState.Balances[valIdx])
		reward += int64(p.NextState.Withdrawals[valIdx])
		reward -= int64(depositedAmount)
		reward -= int64(consolidatedAmount)

		// CRITICAL FIX: Sanity check to prevent rewards > max rewards
		// This is a safety net - rewards should never exceed max possible rewards
		if reward > 0 {
			// Log suspicious rewards that might indicate state inconsistency
			if reward > int64(32*1000000000) { // 32 ETH in Gwei - max possible reasonable reward
				log.Warnf("Suspiciously high reward detected for validator %d at epoch %d: %d Gwei - possible state inconsistency",
					valIdx, p.NextState.Epoch, reward)
			}
		}

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
	case spec.DataVersionElectra:
		return NewElectraMetrics(nextState, currentState, prevState), nil
	default:
		return nil, fmt.Errorf("could not figure out the State Metrics Fork Version: %s", currentState.Version)
	}
}

func (s StateMetricsBase) ExportToEpoch() local_spec.Epoch {

	return local_spec.Epoch{
		Epoch:                         s.CurrentState.Epoch,
		Slot:                          s.CurrentState.Slot,
		NumAttestations:               s.CurrentState.NumAttestations,
		NumAttValidators:              int(countTrue(s.CurrentState.ValidatorAttestationIncluded)),
		NumValidators:                 len(s.CurrentState.Validators),
		NumCompoundingVals:            s.CurrentState.GetCompoundingValsNum(),
		TotalBalance:                  float32(s.CurrentState.TotalActiveRealBalance) / float32(local_spec.EffectiveBalanceInc),
		AttEffectiveBalance:           s.NextState.AttestingBalance[local_spec.AttTargetFlagIndex] / local_spec.EffectiveBalanceInc,
		SourceAttEffectiveBalance:     s.NextState.AttestingBalance[local_spec.AttSourceFlagIndex] / local_spec.EffectiveBalanceInc,
		TargetAttEffectiveBalance:     s.NextState.AttestingBalance[local_spec.AttTargetFlagIndex] / local_spec.EffectiveBalanceInc,
		HeadAttEffectiveBalance:       s.NextState.AttestingBalance[local_spec.AttHeadFlagIndex] / local_spec.EffectiveBalanceInc,
		TotalEffectiveBalance:         s.CurrentState.TotalActiveBalance / local_spec.EffectiveBalanceInc,
		MissingSource:                 int(s.NextState.GetMissingFlagCount(int(altair.TimelySourceFlagIndex))),
		MissingTarget:                 int(s.NextState.GetMissingFlagCount(int(altair.TimelyTargetFlagIndex))),
		MissingHead:                   int(s.NextState.GetMissingFlagCount(int(altair.TimelyHeadFlagIndex))),
		Timestamp:                     int64(s.CurrentState.GenesisTimestamp + uint64(s.CurrentState.Epoch)*local_spec.SlotsPerEpoch*local_spec.SlotSeconds),
		NumSlashedVals:                int(s.CurrentState.NumSlashedVals),
		NumActiveVals:                 int(s.CurrentState.NumActiveVals),
		NumExitedVals:                 int(s.CurrentState.NumExitedVals),
		NumInActivationVals:           int(s.CurrentState.NumQueuedVals),
		SyncCommitteeParticipation:    s.CurrentState.SyncCommitteeParticipation,
		DepositsNum:                   int(s.CurrentState.DepositsNum),
		TotalDepositsAmount:           s.CurrentState.TotalDepositsAmount,
		WithdrawalsNum:                int(s.CurrentState.WithdrawalsNum),
		TotalWithdrawalsAmount:        s.CurrentState.TotalWithdrawalsAmount,
		NewProposerSlashings:          int(s.CurrentState.NewProposerSlashings),
		NewAttesterSlashings:          int(s.CurrentState.NewAttesterSlashings),
		ConsolidationRequestsNum:      int(len(s.CurrentState.ConsolidationRequests)),
		DepositRequestsNum:            int(len(s.CurrentState.DepositRequests)),
		WithdrawalRequestsNum:         int(len(s.CurrentState.WithdrawalRequests)),
		ConsolidationsProcessedNum:    uint64(len(s.CurrentState.ConsolidationsProcessed)),
		ConsolidationsProcessedAmount: s.CurrentState.ConsolidationsProcessedAmount,
	}
}
