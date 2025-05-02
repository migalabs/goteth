package db

import (
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
