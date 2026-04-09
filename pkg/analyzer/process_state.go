package analyzer

import (
	"fmt"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/migalabs/goteth/pkg/db"
	"github.com/migalabs/goteth/pkg/relay"
	"github.com/migalabs/goteth/pkg/spec"
	"github.com/migalabs/goteth/pkg/spec/metrics"
)

var (
	epochProcesserTag = "epoch="
)

// We always provide the epoch we transition to
// To process the transition from epoch 9 to 10, we provide 10 and we retrieve 8, 9, 10
func (s *ChainAnalyzer) ProcessStateTransitionMetrics(epoch phase0.Epoch) {

	if !s.metrics.Epoch {
		return
	}

	routineKey := fmt.Sprintf("%s%d", epochProcesserTag, epoch)
	s.processerBook.Acquire(routineKey) // resgiter we are about to process metrics for epoch

	// Wait for previous epoch processing to complete before reading its blocks.
	// ProcessAttestations writes block.ManualReward on nextState blocks, and the
	// next epoch's processBlockRewards reads from those same shared block objects
	// (via ChainCache). Without this barrier the reader can see a partially
	// accumulated ManualReward, producing incorrect f_cl_manual_reward values
	// (see https://github.com/migalabs/goteth/issues/242).
	if epoch >= 1 {
		prevEpochKey := fmt.Sprintf("%s%d", epochProcesserTag, epoch-1)
		s.processerBook.WaitUntilInactive(prevEpochKey)
	}

	// Retrieve states to process metrics

	prevState := &spec.AgnosticState{}
	currentState := &spec.AgnosticState{}
	nextState := &spec.AgnosticState{}

	var err error

	// this state may never be downloaded if it is below initSlot
	if epoch >= 2 && epoch-2 >= phase0.Epoch(s.initSlot/spec.SlotsPerEpoch) {
		prevState, err = s.downloadCache.StateHistory.Wait(s.ctx, EpochTo[uint64](epoch)-2)
		if err != nil {
			s.processerBook.FreePage(routineKey)
			log.Errorf("context cancelled waiting for state at epoch %d: %s", epoch-2, err)
			return
		}
	}
	if epoch >= 1 && epoch-1 >= phase0.Epoch(s.initSlot/spec.SlotsPerEpoch) {
		currentState, err = s.downloadCache.StateHistory.Wait(s.ctx, EpochTo[uint64](epoch)-1)
		if err != nil {
			s.processerBook.FreePage(routineKey)
			log.Errorf("context cancelled waiting for state at epoch %d: %s", epoch-1, err)
			return
		}
	}
	nextState, err = s.downloadCache.StateHistory.Wait(s.ctx, EpochTo[uint64](epoch))
	if err != nil {
		s.processerBook.FreePage(routineKey)
		log.Errorf("context cancelled waiting for state at epoch %d: %s", epoch, err)
		return
	}

	bundle, err := metrics.StateMetricsByForkVersion(nextState, currentState, prevState, s.cli.Api)
	if err != nil {
		s.processerBook.FreePage(routineKey)
		log.Errorf("could not parse bundle metrics at epoch: %s", err)
		s.stop = true
		s.cancel()
		return
	}

	// If prevState, currentState and nextState are filled, we can process proposer duties, epoch metrics and validator rewards
	if !nextState.EmptyStateRoot() && !currentState.EmptyStateRoot() && !prevState.EmptyStateRoot() {
		s.processEpochDuties(bundle)
		s.processValLastStatus(bundle)
		s.processEpochMetrics(bundle)
		s.processBlockRewards(bundle) // block rewards depend on two previous epochs
		if s.metrics.ValidatorRewards {
			s.processEpochValRewards(bundle)
		}
		s.processSlashings(bundle)
		s.storeDepositsProcessed(bundle) // we store deposits processed from electra + in the database
		s.storeConsolidationRequests(bundle)
		s.storeWithdrawalRequests(bundle)
		s.storeDepositRequests(bundle)
		s.storeConsoidationsProcessed(bundle)
		s.processPoolMetrics(bundle.GetMetricsBase().PrevState.Epoch) // Calculated over prev state so we make sure that tables are filled
	}

	s.processerBook.FreePage(routineKey)

}

func (s *ChainAnalyzer) processSlashings(bundle metrics.StateMetrics) {
	slashings := bundle.GetMetricsBase().NextState.Slashings
	if len(slashings) == 0 {
		return
	}
	err := s.dbClient.PersistSlashings(slashings)
	if err != nil {
		log.Errorf("error persisting slashings: %s", err.Error())
	}
}

// storeDepositsProcessed stores the deposits processed from electra + in the database
func (s *ChainAnalyzer) storeDepositsProcessed(bundle metrics.StateMetrics) {
	depositsProcessed := bundle.GetMetricsBase().NextState.DepositsProcessed
	if len(depositsProcessed) == 0 {
		return
	}
	err := s.dbClient.PersistDeposits(depositsProcessed)
	if err != nil {
		log.Errorf("error persisting deposits processed: %s", err.Error())
	}
}

func (s *ChainAnalyzer) storeConsoidationsProcessed(bundle metrics.StateMetrics) {
	consolidationsProcessed := bundle.GetMetricsBase().NextState.ConsolidationsProcessed
	if len(consolidationsProcessed) == 0 {
		return
	}
	err := s.dbClient.PersistConsolidationsProcessed(consolidationsProcessed)
	if err != nil {
		log.Errorf("error persisting consolidationsProcessed: %s", err.Error())
	}
}

func (s *ChainAnalyzer) storeWithdrawalRequests(bundle metrics.StateMetrics) {
	withdrawalRequests := bundle.GetMetricsBase().NextState.WithdrawalRequests
	if len(withdrawalRequests) == 0 {
		return
	}
	err := s.dbClient.PersistWithdrawalRequests(withdrawalRequests)
	if err != nil {
		log.Errorf("error persisting withdrawal requests: %s", err.Error())
	}
}

func (s *ChainAnalyzer) storeConsolidationRequests(bundle metrics.StateMetrics) {
	consolidationRequests := bundle.GetMetricsBase().NextState.ConsolidationRequests
	if len(consolidationRequests) == 0 {
		return
	}
	err := s.dbClient.PersistConsolidationRequests(consolidationRequests)
	if err != nil {
		log.Errorf("error persisting consolidation requests: %s", err.Error())
	}
}

func (s *ChainAnalyzer) storeDepositRequests(bundle metrics.StateMetrics) {
	depositRequests := bundle.GetMetricsBase().NextState.DepositRequests
	if len(depositRequests) == 0 {
		return
	}
	err := s.dbClient.PersistDepositRequests(depositRequests)
	if err != nil {
		log.Errorf("error persisting deposit requests: %s", err.Error())
	}
}

func (s *ChainAnalyzer) processEpochMetrics(bundle metrics.StateMetrics) {

	// we need sameEpoch and nextEpoch
	metricsBase := bundle.GetMetricsBase()
	epoch := metricsBase.ExportToEpoch()

	log.Debugf("persisting epoch metrics: epoch %d", epoch.Epoch)

	err := s.dbClient.PersistEpochs([]spec.Epoch{epoch})
	if err != nil {
		log.Errorf("error persisting epoch: %s", err.Error())
	}

}

func (s *ChainAnalyzer) processPoolMetrics(epoch phase0.Epoch) {

	log.Debugf("persisting pool summaries: epoch %d", epoch)

	err := s.dbClient.InsertPoolSummary(epoch)

	// we need sameEpoch and nextEpoch

	if err != nil {
		log.Fatalf("error persisting pool metrics: %s", err.Error())
	}

}

func (s *ChainAnalyzer) processEpochDuties(bundle metrics.StateMetrics) {

	nextState := bundle.GetMetricsBase().NextState

	// Build a map of slot → proposed from the actual blocks in the cache.
	// This correctly handles both missed blocks (Proposed=false from
	// CreateMissingBlock) and orphaned blocks that were reorged out,
	// unlike MissedBlocks which only detects slots with no block proposed.
	proposedBySlot := make(map[phase0.Slot]bool)
	for _, block := range nextState.Blocks {
		proposedBySlot[block.Slot] = block.Proposed
	}

	var duties []spec.ProposerDuty

	for _, item := range nextState.EpochStructs.ProposerDuties {
		duties = append(duties, spec.ProposerDuty{
			ValIdx:       item.ValidatorIndex,
			ProposerSlot: item.Slot,
			Proposed:     proposedBySlot[item.Slot],
		})
	}

	err := s.dbClient.PersistDuties(duties)
	if err != nil {
		log.Fatalf("error persisting proposer duties: %s", err.Error())
	}

}

func (s *ChainAnalyzer) processValLastStatus(bundle metrics.StateMetrics) {

	if s.downloadMode == "finalized" {
		var valStatusArr []spec.ValidatorLastStatus
		nextState := bundle.GetMetricsBase().NextState
		for i, validator := range nextState.Validators {
			valIdx := phase0.ValidatorIndex(i)
			newVal := spec.ValidatorLastStatus{
				ValIdx:                valIdx,
				Epoch:                 nextState.Epoch,
				CurrentBalance:        nextState.Balances[valIdx],
				EffectiveBalance:      validator.EffectiveBalance,
				CurrentStatus:         nextState.GetValStatus(valIdx),
				Slashed:               validator.Slashed,
				ActivationEpoch:       validator.ActivationEpoch,
				WithdrawalEpoch:       validator.WithdrawableEpoch,
				ExitEpoch:             validator.ExitEpoch,
				PublicKey:             validator.PublicKey,
				WithdrawalCredentials: validator.WithdrawalCredentials,
			}
			valStatusArr = append(valStatusArr, newVal)
		}
		if len(valStatusArr) > 0 { // persist everything

			err := s.dbClient.PersistValLastStatus(valStatusArr)
			if err != nil {
				log.Errorf("error persisting validator last status: %s", err.Error())
			}
			err = s.dbClient.DeleteValLastStatus(bundle.GetMetricsBase().NextState.Epoch)
			if err != nil {
				log.Errorf("error deleting validator last status: %s", err.Error())
			}
		}
	}
}

func (s *ChainAnalyzer) processEpochValRewards(bundle metrics.StateMetrics) {
	var insertValsObj []spec.ValidatorRewards
	log.Debugf("persising validator metrics: epoch %d", bundle.GetMetricsBase().NextState.Epoch)
	nextState := bundle.GetMetricsBase().NextState
	prevState := bundle.GetMetricsBase().PrevState
	// process each validator
	for i, validator := range nextState.Validators {
		valIdx := phase0.ValidatorIndex(i)

		// get max reward at given epoch using the formulas
		maxRewards, err := bundle.GetMaxReward(valIdx)
		if err != nil {
			log.Errorf("Error obtaining max reward: %s", err.Error())
			continue
		}

		// Check validator status conditions
		isActive := spec.IsActive(*validator, prevState.Epoch)
		isSlashed := validator.Slashed
		isExited := validator.ExitEpoch <= prevState.Epoch
		// Only process validators that are active, or slashed and not exited, or in sync committee
		if !isActive && (!isSlashed || isExited) && !maxRewards.InSyncCommittee {
			continue
		}

		insertValsObj = append(insertValsObj, maxRewards)
	}
	if len(insertValsObj) > 0 { // persist everything
		err := s.dbClient.PersistValidatorRewards(insertValsObj)
		if err != nil {
			log.Fatalf("error persisting validator rewards: %s", err.Error())
		}
	}

	if s.rewardsAggregationEpochs > 1 {
		s.validatorsRewardsAggregationsMu.Lock()

		epoch := bundle.GetMetricsBase().NextState.Epoch

		// Only aggregate if:
		//  1. The epoch belongs to the current window (rejects stale epochs
		//     from already-flushed windows reprocessed by AdvanceFinalized).
		//  2. The epoch hasn't been aggregated yet (rejects duplicate calls
		//     for the same epoch, e.g. AdvanceFinalized reprocessing an epoch
		//     that the normal flow already handled). Using a set of seen
		//     epochs instead of a counter prevents the cumulative window
		//     shift described in #255.
		inWindow := epoch >= s.startEpochAggregation && epoch <= s.endEpochAggregation
		_, alreadySeen := s.aggregatedEpochsInWindow[epoch]

		if inWindow && !alreadySeen {
			for _, maxRewards := range insertValsObj {
				valIdx := maxRewards.ValidatorIndex
				if _, ok := s.validatorsRewardsAggregations[valIdx]; !ok {
					s.validatorsRewardsAggregations[valIdx] = spec.NewValidatorRewardsAggregation(valIdx, s.startEpochAggregation, s.endEpochAggregation)
				}
				s.validatorsRewardsAggregations[valIdx].Aggregate(maxRewards)
			}

			s.aggregatedEpochsInWindow[epoch] = true

			if len(s.aggregatedEpochsInWindow) >= s.rewardsAggregationEpochs {
				if len(s.validatorsRewardsAggregations) > 0 {
					err := s.dbClient.PersistValidatorRewardsAggregation(s.validatorsRewardsAggregations)
					if err != nil {
						log.Fatalf("error persisting validator rewards aggregation: %s", err.Error())
					}
				}
				s.validatorsRewardsAggregations = make(map[phase0.ValidatorIndex]*spec.ValidatorRewardsAggregation)
				s.startEpochAggregation = s.endEpochAggregation + 1
				s.endEpochAggregation = s.endEpochAggregation + phase0.Epoch(s.rewardsAggregationEpochs)
				s.aggregatedEpochsInWindow = make(map[phase0.Epoch]bool)
			}
		}

		s.validatorsRewardsAggregationsMu.Unlock()
	}

}

func (s *ChainAnalyzer) processBlockRewards(bundle metrics.StateMetrics) {

	blockRewards := make([]db.BlockReward, 0)

	mevBids, err := s.relayCli.GetDeliveredBidsPerSlotRange(bundle.GetMetricsBase().CurrentState.Slot, spec.SlotsPerEpoch)
	if err != nil {
		log.Errorf("error getting mev bids: %s", err.Error())
	}

	for _, block := range bundle.GetMetricsBase().CurrentState.Blocks {
		// Wait for ProcessBlock to finish appending transactions before reading
		// them in BlockGasFees(). Without this, AgnosticTransactions may be empty
		// and f_reward_fees/f_burnt_fees are written as 0 (see #249).
		slotKey := fmt.Sprintf("%s%d", slotProcesserTag, block.Slot)
		s.processerBook.WaitUntilInactive(slotKey)

		blockRewards = append(blockRewards, s.getSingleBlockRewards(*block, mevBids))
	}

	s.dbClient.PersistBlockRewards(blockRewards)

}

func (s *ChainAnalyzer) getSingleBlockRewards(
	block spec.AgnosticBlock,
	mevBids *relay.RelayBidsPerSlot) db.BlockReward {
	slot := block.Slot
	bids := mevBids.GetBidsAtSlot(slot)
	clManualReward := block.ManualReward
	clApiReward := phase0.Gwei(block.Reward.Data.Total)
	var err error

	// obtain
	burntFees := uint64(0)
	rewardFees := uint64(0)
	bidCommision := uint64(0)
	relayAddresses := make([]string, 0)
	builderPubkeys := make([]string, 0)

	rewardFees, burntFees, err = block.BlockGasFees()
	if err != nil {
		log.Warnf("block at slot %d gas fees not calculated: %s", slot, err)
	}

	if len(bids) > 0 {

		blockHash := block.ExecutionPayload.BlockHash

		for address, bid := range bids {
			bidBlockHash := bid.BlockHash

			if blockHash == bidBlockHash {
				bidCommision = bid.Value.Uint64()
				relayAddresses = append(relayAddresses, address)
				builderPubkeys = append(builderPubkeys, bid.BuilderPubkey.String())
			}
		}
	}
	return db.BlockReward{
		Slot:           slot,
		CLManualReward: clManualReward,
		CLApiReward:    clApiReward,
		RewardFees:     rewardFees,
		BurntFees:      burntFees,
		Relays:         relayAddresses,
		BidCommision:   bidCommision,
		BuilderPubkeys: builderPubkeys,
	}
}
