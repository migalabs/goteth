package db

import (
	"fmt"

	"github.com/ClickHouse/ch-go/proto"
	"github.com/attestantio/go-eth2-client/spec/phase0"

	"github.com/migalabs/goteth/pkg/spec"
)

var (
	valRewardsAggregationTable             = "t_validator_rewards_aggregation"
	insertValidatorRewardsAggregationQuery = `
	INSERT INTO %s (	
		f_val_idx, 
		f_start_epoch,
		f_end_epoch, 
		f_reward, 
		f_max_reward,
		f_max_att_reward,
		f_max_sync_reward,
		f_base_reward,
		f_in_sync_committee_count,
		f_sync_committee_participations_included,
		f_attestations_included,
		f_missing_source_count,
		f_missing_target_count,
		f_missing_head_count,
		f_block_api_reward,
		f_block_experimental_reward,
		f_inclusion_delay_sum) VALUES`

	deleteValidatorRewardsAggregationUntilEpochQuery = `
		DELETE FROM %s
		WHERE f_start_epoch <= $1;
	`

	// Both read the primary key in order, so they only touch the last granules.
	selectLastAggregationStartQuery = `
		SELECT f_start_epoch
		FROM %s
		ORDER BY f_start_epoch DESC
		LIMIT 1`

	selectLastAggregationStartBeforeQuery = `
		SELECT f_start_epoch
		FROM %s
		WHERE f_start_epoch < %d
		ORDER BY f_start_epoch DESC
		LIMIT 1`

	// FINAL: re-inserted or trimmed windows (same key) must count once, with their latest end.
	selectAggregationWindowsQuery = `
		SELECT f_start_epoch, max(f_end_epoch) AS f_last_end, uniqExact(f_val_idx) AS f_validators
		FROM %s FINAL
		WHERE f_start_epoch IN (%d, %d)
		GROUP BY f_start_epoch`

	// A last window with fewer validators than this share of the window before it is taken as
	// partially persisted and is aggregated again (replaced by key). A window is one insert, so
	// this is only a safety net.
	aggregationCompleteRowsRatio = 0.99
)

func rewardsAggregationInput(vals []spec.ValidatorRewardsAggregation) proto.Input {
	// one object per column
	var (
		f_val_idx                                proto.ColUInt64
		f_start_epoch                            proto.ColUInt64
		f_end_epoch                              proto.ColUInt64
		f_reward                                 proto.ColInt64
		f_max_reward                             proto.ColUInt64
		f_max_att_reward                         proto.ColUInt64
		f_max_sync_reward                        proto.ColUInt64
		f_base_reward                            proto.ColUInt64
		f_in_sync_committee_count                proto.ColUInt16
		f_sync_committee_participations_included proto.ColUInt16
		f_attestations_included                  proto.ColUInt16
		f_missing_source_count                   proto.ColUInt16
		f_missing_target_count                   proto.ColUInt16
		f_missing_head_count                     proto.ColUInt16
		f_block_api_reward                       proto.ColUInt64
		f_block_experimental_reward              proto.ColUInt64
		f_inclusion_delay_sum                    proto.ColUInt32
	)

	for _, val := range vals {
		f_val_idx.Append(uint64(val.ValidatorIndex))
		f_start_epoch.Append(uint64(val.StartEpoch))
		f_end_epoch.Append(uint64(val.EndEpoch))
		f_reward.Append(int64(val.Reward))
		f_max_reward.Append(uint64(val.MaxReward))
		f_max_att_reward.Append(uint64(val.MaxAttestationReward))
		f_max_sync_reward.Append(uint64(val.MaxSyncCommitteeReward))
		f_base_reward.Append(uint64(val.BaseReward))
		f_in_sync_committee_count.Append(val.InSyncCommitteeCount)
		f_sync_committee_participations_included.Append(val.SyncCommitteeParticipationsIncluded)
		f_attestations_included.Append(val.AttestationsIncluded)
		f_missing_source_count.Append(val.MissingSourceCount)
		f_missing_target_count.Append(val.MissingTargetCount)
		f_missing_head_count.Append(val.MissingHeadCount)
		f_block_api_reward.Append(uint64(val.ProposerApiReward))
		f_block_experimental_reward.Append(uint64(val.ProposerManualReward))
		f_inclusion_delay_sum.Append(uint32(val.InclusionDelaySum))
	}

	return proto.Input{
		{Name: "f_val_idx", Data: f_val_idx},
		{Name: "f_start_epoch", Data: f_start_epoch},
		{Name: "f_end_epoch", Data: f_end_epoch},
		{Name: "f_reward", Data: f_reward},
		{Name: "f_max_reward", Data: f_max_reward},
		{Name: "f_max_att_reward", Data: f_max_att_reward},
		{Name: "f_max_sync_reward", Data: f_max_sync_reward},
		{Name: "f_base_reward", Data: f_base_reward},
		{Name: "f_in_sync_committee_count", Data: f_in_sync_committee_count},
		{Name: "f_sync_committee_participations_included", Data: f_sync_committee_participations_included},
		{Name: "f_attestations_included", Data: f_attestations_included},
		{Name: "f_missing_source_count", Data: f_missing_source_count},
		{Name: "f_missing_target_count", Data: f_missing_target_count},
		{Name: "f_missing_head_count", Data: f_missing_head_count},
		{Name: "f_block_api_reward", Data: f_block_api_reward},
		{Name: "f_block_experimental_reward", Data: f_block_experimental_reward},
		{Name: "f_inclusion_delay_sum", Data: f_inclusion_delay_sum},
	}
}

func (p *DBService) PersistValidatorRewardsAggregation(data map[phase0.ValidatorIndex]*spec.ValidatorRewardsAggregation) error {
	persistObj := PersistableObject[spec.ValidatorRewardsAggregation]{
		input: rewardsAggregationInput,
		table: valRewardsAggregationTable,
		query: insertValidatorRewardsAggregationQuery,
	}

	for _, item := range data {
		persistObj.Append(*item)
	}

	err := p.Persist(persistObj.ExportPersist())
	if err != nil {
		log.Errorf("error persisting validator rewards: %s", err.Error())
	}
	return err
}

// RetrieveNextAggregationStart returns the first epoch that no persisted rewards aggregation
// window covers, so a restarted analyzer can continue the window sequence instead of starting a
// new one wherever it resumes. found is false when the table is empty. If the last window has
// clearly fewer validators than the one before it, its start is returned so it is aggregated again.
func (p *DBService) RetrieveNextAggregationStart() (next phase0.Epoch, found bool, err error) {
	var last []struct {
		F_start_epoch uint64 `ch:"f_start_epoch"`
	}
	if err = p.highSelect(fmt.Sprintf(selectLastAggregationStartQuery, valRewardsAggregationTable), &last); err != nil {
		return 0, false, err
	}
	if len(last) == 0 {
		return 0, false, nil
	}
	lastStart := last[0].F_start_epoch

	var prev []struct {
		F_start_epoch uint64 `ch:"f_start_epoch"`
	}
	if err = p.highSelect(fmt.Sprintf(selectLastAggregationStartBeforeQuery, valRewardsAggregationTable, lastStart), &prev); err != nil {
		return 0, false, err
	}
	prevStart := lastStart
	if len(prev) > 0 {
		prevStart = prev[0].F_start_epoch
	}

	var windows []struct {
		F_start_epoch uint64 `ch:"f_start_epoch"`
		F_last_end    uint64 `ch:"f_last_end"`
		F_validators  uint64 `ch:"f_validators"`
	}
	if err = p.highSelect(fmt.Sprintf(selectAggregationWindowsQuery, valRewardsAggregationTable, lastStart, prevStart), &windows); err != nil {
		return 0, false, err
	}
	var lastEnd, lastRows, prevRows uint64
	for _, w := range windows {
		if w.F_start_epoch == lastStart {
			lastEnd, lastRows = w.F_last_end, w.F_validators
		}
		if w.F_start_epoch == prevStart && prevStart != lastStart {
			prevRows = w.F_validators
		}
	}
	if lastRows == 0 {
		return 0, false, fmt.Errorf("aggregation window %d has no rows", lastStart)
	}
	return nextAggregationStart(lastStart, lastEnd, lastRows, prevRows), true, nil
}

// nextAggregationStart is the decision behind RetrieveNextAggregationStart.
func nextAggregationStart(lastStart, lastEnd, lastRows, prevRows uint64) phase0.Epoch {
	if prevRows > 0 && float64(lastRows) < aggregationCompleteRowsRatio*float64(prevRows) {
		log.Warnf("last rewards aggregation window %d-%d has %d validators, the previous one %d: aggregating it again",
			lastStart, lastEnd, lastRows, prevRows)
		return phase0.Epoch(lastStart)
	}
	return phase0.Epoch(lastEnd + 1)
}

func (p *DBService) DeleteValidatorRewardsAggregationUntil(epoch phase0.Epoch) error {

	deleteObj := DeletableObject{
		query: deleteValidatorRewardsAggregationUntilEpochQuery,
		table: valRewardsAggregationTable,
		args:  []any{epoch},
	}

	err := p.Delete(deleteObj)
	if err != nil {
		log.Errorf("error deleting validator rewards: %s", err.Error())
	}

	return err
}
