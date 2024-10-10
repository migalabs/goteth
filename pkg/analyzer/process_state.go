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

		s.processPoolMetrics(bundle.GetMetricsBase().CurrentState.Epoch)
		s.processEpochMetrics(bundle)
		s.processBlockRewards(bundle) // block rewards depend on two previous epochs
		if s.metrics.ValidatorRewards {
			s.processEpochValRewards(bundle)
		}
	}

	s.processerBook.FreePage(routineKey)

}

func (s *ChainAnalyzer) processEpochMetrics(bundle metrics.StateMetrics) {

	// we need sameEpoch and nextEpoch

	epoch := bundle.GetMetricsBase().ExportToEpoch()

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
		for valIdx, validator := range bundle.GetMetricsBase().NextState.Validators {

			newVal := spec.ValidatorLastStatus{
				ValIdx:          phase0.ValidatorIndex(valIdx),
				Epoch:           bundle.GetMetricsBase().NextState.Epoch,
				CurrentBalance:  bundle.GetMetricsBase().NextState.Balances[valIdx],
				CurrentStatus:   bundle.GetMetricsBase().NextState.GetValStatus(phase0.ValidatorIndex(valIdx)),
				Slashed:         validator.Slashed,
				ActivationEpoch: validator.ActivationEpoch,
				WithdrawalEpoch: validator.WithdrawableEpoch,
				ExitEpoch:       validator.ExitEpoch,
				PublicKey:       validator.PublicKey,
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

	// process each validator
	for valIdx := range bundle.GetMetricsBase().NextState.Validators {
		if valIdx >= len(bundle.GetMetricsBase().NextState.Validators) {
			continue // validator is not in the chain yet
		}
		valIdx := phase0.ValidatorIndex(valIdx)
		// get max reward at given epoch using the formulas
		maxRewards, err := bundle.GetMaxReward(valIdx)
		if err != nil {
			log.Errorf("Error obtaining max reward: %s", err.Error())
			continue
		}
		if s.rewardsAggregationEpochs == 1 {
			continue
		}
		// if validator is not in s.validatorsRewardsAggregations, we need to create it
		if _, ok := s.validatorsRewardsAggregations[phase0.ValidatorIndex(valIdx)]; !ok {
			s.validatorsRewardsAggregations[phase0.ValidatorIndex(valIdx)] = spec.NewValidatorRewardsAggregation(valIdx, s.startEpochAggregation, s.endEpochAggregation)
		}
		s.validatorsRewardsAggregations[valIdx].Aggregate(maxRewards)
		insertValsObj = append(insertValsObj, maxRewards)
	}
	if len(insertValsObj) > 0 { // persist everything
		err := s.dbClient.PersistValidatorRewards(insertValsObj)
		if err != nil {
			log.Fatalf("error persisting validator rewards: %s", err.Error())
		}
	}

	if s.rewardsAggregationEpochs > 1 && bundle.GetMetricsBase().NextState.Epoch == s.endEpochAggregation {
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

	mevBids, err := s.relayCli.GetDeliveredBidsPerSlotRange(bundle.GetMetricsBase().NextState.Slot, spec.SlotsPerEpoch)
	if err != nil {
		log.Errorf("error getting mev bids: %s", err.Error())
	}

	for _, block := range bundle.GetMetricsBase().NextState.Blocks {
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
