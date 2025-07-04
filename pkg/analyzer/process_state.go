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

	// Retrieve states to process metrics

	prevState := &spec.AgnosticState{}
	currentState := &spec.AgnosticState{}
	nextState := &spec.AgnosticState{}

	// this state may never be downloaded if it is below initSlot
	if epoch >= 2 && epoch-2 >= phase0.Epoch(s.initSlot/spec.SlotsPerEpoch) {
		prevState = s.downloadCache.StateHistory.Wait(EpochTo[uint64](epoch) - 2)
	}
	if epoch >= 1 && epoch-1 >= phase0.Epoch(s.initSlot/spec.SlotsPerEpoch) {
		currentState = s.downloadCache.StateHistory.Wait(EpochTo[uint64](epoch) - 1)
	}
	nextState = s.downloadCache.StateHistory.Wait(EpochTo[uint64](epoch))

	bundle, err := metrics.StateMetricsByForkVersion(nextState, currentState, prevState, s.cli.Api)
	if err != nil {
		s.processerBook.FreePage(routineKey)
		log.Errorf("could not parse bundle metrics at epoch: %s", err)
		s.stop = true
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

	missedBlocks := bundle.GetMetricsBase().NextState.MissedBlocks

	var duties []spec.ProposerDuty

	for _, item := range bundle.GetMetricsBase().NextState.EpochStructs.ProposerDuties {

		newDuty := spec.ProposerDuty{
			ValIdx:       item.ValidatorIndex,
			ProposerSlot: item.Slot,
			Proposed:     true,
		}
		for _, item := range missedBlocks {
			if newDuty.ProposerSlot == item { // we found the proposer slot in the missed blocks
				newDuty.Proposed = false
			}
		}
		duties = append(duties, newDuty)
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
		isActive := spec.IsActive(*validator, nextState.Epoch)
		isSlashed := validator.Slashed
		isExited := validator.ExitEpoch <= nextState.Epoch
		// Only process validators that are active, or slashed and not exited, or in sync committee
		if !isActive && (!isSlashed || isExited) && !maxRewards.InSyncCommittee {
			continue
		}

		if s.rewardsAggregationEpochs > 1 {
			// if validator is not in s.validatorsRewardsAggregations, we need to create it
			if _, ok := s.validatorsRewardsAggregations[valIdx]; !ok {
				s.validatorsRewardsAggregations[valIdx] = spec.NewValidatorRewardsAggregation(valIdx, s.startEpochAggregation, s.endEpochAggregation)
			}
			s.validatorsRewardsAggregations[valIdx].Aggregate(maxRewards)
		}
		insertValsObj = append(insertValsObj, maxRewards)
	}
	if len(insertValsObj) > 0 { // persist everything
		err := s.dbClient.PersistValidatorRewards(insertValsObj)
		if err != nil {
			log.Fatalf("error persisting validator rewards: %s", err.Error())
		}
	}

	if s.rewardsAggregationEpochs > 1 && nextState.Epoch == s.endEpochAggregation {
		if len(s.validatorsRewardsAggregations) > 0 {
			err := s.dbClient.PersistValidatorRewardsAggregation(s.validatorsRewardsAggregations)
			if err != nil {
				log.Fatalf("error persisting validator rewards aggregation: %s", err.Error())
			}
		}
		s.validatorsRewardsAggregations = make(map[phase0.ValidatorIndex]*spec.ValidatorRewardsAggregation)
		s.startEpochAggregation = s.endEpochAggregation + 1
		s.endEpochAggregation = s.endEpochAggregation + phase0.Epoch(s.rewardsAggregationEpochs)
	}

}

func (s *ChainAnalyzer) processBlockRewards(bundle metrics.StateMetrics) {

	blockRewards := make([]db.BlockReward, 0)

	mevBids, err := s.relayCli.GetDeliveredBidsPerSlotRange(bundle.GetMetricsBase().CurrentState.Slot, spec.SlotsPerEpoch)
	if err != nil {
		log.Errorf("error getting mev bids: %s", err.Error())
	}

	for _, block := range bundle.GetMetricsBase().CurrentState.Blocks {
		blockRewards = append(blockRewards, s.getSingleBlockRewards(*block, mevBids))
	}

	s.dbClient.PersistBlockRewards(blockRewards)

}

func (s *ChainAnalyzer) getSingleBlockRewards(
	block spec.AgnosticBlock,
	mevBids relay.RelayBidsPerSlot) db.BlockReward {
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
