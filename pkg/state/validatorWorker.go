package state

import (
	"sync"
	"time"

	"github.com/attestantio/go-eth2-client/spec/altair"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/cortze/eth-cl-state-analyzer/pkg/db/postgresql"
	"github.com/cortze/eth-cl-state-analyzer/pkg/db/postgresql/model"
	"github.com/cortze/eth-cl-state-analyzer/pkg/state_metrics"
	"github.com/cortze/eth-cl-state-analyzer/pkg/state_metrics/fork_state"
	"github.com/jackc/pgx/v4"
	"github.com/sirupsen/logrus"
)

func (s *StateAnalyzer) runWorker(wlog *logrus.Entry, wgWorkers *sync.WaitGroup, processFinishedFlag *bool) {
	defer wgWorkers.Done()
	batch := pgx.Batch{}
	// keep iterating until the channel is closed due to finishing
loop:
	for {

		if *processFinishedFlag && len(s.ValTaskChan) == 0 {
			wlog.Warn("the task channel has been closed, finishing worker routine")
			if batch.Len() > 0 {

				wlog.Debugf("Sending last validator batch to be stored...")
				s.dbClient.WriteChan <- batch
				batch = pgx.Batch{}

			}
			break loop
		}

		select {
		case valTask, ok := <-s.ValTaskChan:
			// check if the channel has been closed
			if !ok {
				wlog.Warn("the task channel has been closed, finishing worker routine")
				return
			}

			stateMetrics := valTask.StateMetricsObj
			wlog.Debugf("task received for val %d - %d in slot %d", valTask.ValIdxs[0], valTask.ValIdxs[len(valTask.ValIdxs)-1], stateMetrics.GetMetricsBase().CurrentState.Slot)
			// Proccess State
			snapshot := time.Now()

			// batch metrics
			summaryMet := NewSummaryMetrics()

			// process each validator
			for _, valIdx := range valTask.ValIdxs {

				if valIdx >= uint64(len(stateMetrics.GetMetricsBase().NextState.Validators)) {
					continue // validator is not in the chain yet
				}
				// get max reward at given epoch using the formulas
				maxRewards, err := stateMetrics.GetMaxReward(valIdx)

				if err != nil {
					log.Errorf("Error obtaining max reward: ", err.Error())
					continue
				}

				// calculate the current balance of validator
				balance := stateMetrics.GetMetricsBase().NextState.Balances[valIdx]

				// keep in mind that att rewards for epoch 10 can be seen at beginning of epoch 12,
				// after state_transition
				// https://notes.ethereum.org/@vbuterin/Sys3GLJbD#Epoch-processing

				// CurrentState Flags measure previous epoch flags
				flags := stateMetrics.GetMetricsBase().CurrentState.MissingFlags(valIdx)

				// create a model to be inserted into the db in the next epoch
				validatorDBRow := model.NewValidatorRewards(
					valIdx,
					stateMetrics.GetMetricsBase().NextState.Slot,
					stateMetrics.GetMetricsBase().NextState.Epoch,
					balance,
					stateMetrics.GetMetricsBase().EpochReward(valIdx), // reward is written after state transition
					maxRewards.MaxReward,
					maxRewards.Attestation,
					maxRewards.SyncCommittee,
					stateMetrics.GetMetricsBase().GetAttSlot(valIdx),
					stateMetrics.GetMetricsBase().GetAttInclusionSlot(valIdx),
					maxRewards.BaseReward,
					maxRewards.InSyncCommittee,
					maxRewards.ProposerSlot, // TODO: there can be several proposer slots, deprecate
					flags[altair.TimelySourceFlagIndex],
					flags[altair.TimelyTargetFlagIndex],
					flags[altair.TimelyHeadFlagIndex],
					stateMetrics.GetMetricsBase().NextState.GetValStatus(valIdx))

				if s.Metrics.Validator {
					batch.Queue(model.UpsertValidator,
						validatorDBRow.ValidatorIndex,
						validatorDBRow.Slot,
						validatorDBRow.Epoch,
						validatorDBRow.ValidatorBalance,
						validatorDBRow.Reward,
						validatorDBRow.MaxReward,
						validatorDBRow.AttestationReward,
						validatorDBRow.SyncCommitteeReward,
						validatorDBRow.BaseReward,
						validatorDBRow.InSyncCommittee,
						validatorDBRow.MissingSource,
						validatorDBRow.MissingTarget,
						validatorDBRow.MissingHead,
						validatorDBRow.Status)

					if batch.Len() > postgresql.MAX_BATCH_QUEUE {
						wlog.Debugf("Sending batch to be stored...")
						s.dbClient.WriteChan <- batch
						batch = pgx.Batch{}
					}
				}

				summaryMet.AddMetrics(maxRewards, stateMetrics, valIdx, validatorDBRow)

			}

			if s.Metrics.PoolSummary && valTask.PoolName != "" {
				// only send summary batch in case pools were introduced by the user and we have a name to identify it
				summaryMet.Aggregate()
				// create and send summary batch
				summaryBatch := pgx.Batch{}
				summaryBatch.Queue(model.UpsertPoolSummary,
					valTask.PoolName,
					stateMetrics.GetMetricsBase().NextState.Epoch,
					summaryMet.AvgReward,
					summaryMet.AvgMaxReward,
					summaryMet.AvgAttMaxReward,
					summaryMet.AvgSyncMaxReward,
					summaryMet.AvgBaseReward,
					summaryMet.MissingSourceCount,
					summaryMet.MissingTargetCount,
					summaryMet.MissingHeadCount,
					summaryMet.NumActiveVals,
					summaryMet.NumSyncVals)

				wlog.Debugf("Sending pool summary batch (%s) to be stored...", valTask.PoolName)
				s.dbClient.WriteChan <- summaryBatch
			}
			wlog.Debugf("Validator group processed, worker freed for next group. Took %f seconds", time.Since(snapshot).Seconds())

		case <-s.ctx.Done():
			log.Info("context has died, closing state worker routine")
			return
		default:
			// if there is something to persist and no more incoming tasks, flush validator batch
			if batch.Len() > 0 && len(s.ValTaskChan) == 0 {
				wlog.Debugf("Sending batch to be stored (no more tasks)...")
				s.dbClient.WriteChan <- batch
				batch = pgx.Batch{}
			}
		}

	}
	wlog.Infof("Validator worker finished, no more tasks to process")
}

type SummaryMetrics struct {
	AvgReward          float64
	AvgMaxReward       float64
	AvgAttMaxReward    float64
	AvgSyncMaxReward   float64
	AvgBaseReward      float64
	MissingSourceCount uint64
	MissingTargetCount uint64
	MissingHeadCount   uint64

	NumActiveVals      uint64
	NumNonProposerVals uint64
	NumSyncVals        uint64
}

func NewSummaryMetrics() SummaryMetrics {
	return SummaryMetrics{}
}

func (s *SummaryMetrics) AddMetrics(
	maxRewards state_metrics.ValidatorSepRewards,
	stateMetrics state_metrics.StateMetrics,
	valIdx uint64,
	validatorDBRow model.ValidatorRewards) {
	if maxRewards.ProposerSlot == -1 {
		// only do rewards statistics in case the validator is not a proposer
		// right now we cannot measure the max reward for a proposer

		// process batch metrics
		s.AvgReward += float64(stateMetrics.GetMetricsBase().EpochReward(valIdx))
		s.AvgMaxReward += float64(maxRewards.MaxReward)
		s.NumNonProposerVals += 1
	}
	if maxRewards.InSyncCommittee {
		s.NumSyncVals += 1
		s.AvgSyncMaxReward += float64(maxRewards.SyncCommittee)
	}

	s.AvgBaseReward += float64(maxRewards.BaseReward)

	// in case of Phase0 AttestationRewards and InlusionDelay is filled
	// in case of Altair, only FlagIndexReward is filled
	// TODO: we might need to do the same for single validator rewards
	s.AvgAttMaxReward += float64(maxRewards.Attestation)

	if fork_state.IsActive(*stateMetrics.GetMetricsBase().NextState.Validators[valIdx],
		phase0.Epoch(stateMetrics.GetMetricsBase().NextState.Epoch)) {
		s.NumActiveVals += 1

		if validatorDBRow.MissingSource {
			s.MissingSourceCount += 1
		}

		if validatorDBRow.MissingTarget {
			s.MissingTargetCount += 1
		}

		if validatorDBRow.MissingHead {
			s.MissingHeadCount += 1
		}
	}
}

func (s *SummaryMetrics) Aggregate() {
	// calculate averages
	s.AvgReward = s.AvgReward / float64(s.NumNonProposerVals)
	s.AvgMaxReward = s.AvgMaxReward / float64(s.NumNonProposerVals)

	s.AvgBaseReward = s.AvgBaseReward / float64(s.NumActiveVals)
	s.AvgAttMaxReward = s.AvgAttMaxReward / float64(s.NumActiveVals)
	s.AvgSyncMaxReward = s.AvgSyncMaxReward / float64(s.NumSyncVals)

	// sanitize in case of division by 0
	if s.NumActiveVals == 0 {
		s.AvgBaseReward = 0
		s.AvgAttMaxReward = 0
	}

	if s.NumNonProposerVals == 0 {
		// al validators are proposers, therefore average rewards cannot be calculated
		// (we still cannot calulate proposer max rewards)
		s.AvgReward = 0
		s.AvgMaxReward = 0
	}

	// avoid division by 0
	if s.NumSyncVals == 0 {
		s.AvgSyncMaxReward = 0
	}
}
