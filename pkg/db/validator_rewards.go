package db

import (
	"github.com/ClickHouse/ch-go/proto"
	"github.com/attestantio/go-eth2-client/spec/phase0"

	"github.com/migalabs/goteth/pkg/spec"
)

var (
	valRewardsTable             = "t_validator_rewards_summary"
	insertValidatorRewardsQuery = `
	INSERT INTO %s (	
		f_val_idx, 
		f_epoch, 
		f_balance_eth, 
		f_effective_balance,
		f_withdrawal_prefix,
		f_reward, 
		f_max_reward,
		f_max_att_reward,
		f_max_sync_reward,
		f_att_slot,
		f_attestation_included,
		f_base_reward,
		f_in_sync_committee,
		f_missing_source,
		f_missing_target,
		f_missing_head,
		f_status,
		f_block_api_reward,
		f_block_experimental_reward,
		f_inclusion_delay) VALUES`

	deleteValidatorRewardsInEpochQuery = `
		DELETE FROM %s
		WHERE f_epoch = $1;
	`

	deleteValidatorRewardsUntilEpochQuery = `
		DELETE FROM %s
		WHERE f_epoch <= $1;
	`
)

func rewardsInput(vals []spec.ValidatorRewards) proto.Input {
	// one object per column
	var (
		f_val_idx                   proto.ColUInt64
		f_epoch                     proto.ColUInt64
		f_balance_eth               proto.ColFloat32
		f_effective_balance         proto.ColUInt64
		f_withdrawal_prefix         proto.ColUInt8
		f_reward                    proto.ColInt64
		f_max_reward                proto.ColUInt64
		f_max_att_reward            proto.ColUInt64
		f_max_sync_reward           proto.ColUInt64
		f_att_slot                  proto.ColUInt64
		f_attestation_included      proto.ColBool
		f_base_reward               proto.ColUInt64
		f_in_sync_committee         proto.ColBool
		f_missing_source            proto.ColBool
		f_missing_target            proto.ColBool
		f_missing_head              proto.ColBool
		f_status                    proto.ColUInt8
		f_block_api_reward          proto.ColUInt64
		f_block_experimental_reward proto.ColUInt64
		f_inclusion_delay           proto.ColUInt8
	)

	for _, val := range vals {
		f_val_idx.Append(uint64(val.ValidatorIndex))
		f_epoch.Append(uint64(val.Epoch))
		f_balance_eth.Append(float32(val.BalanceToEth()))
		f_effective_balance.Append(uint64(val.EffectiveBalance))
		f_withdrawal_prefix.Append(uint8(val.WithdrawalPrefix))
		f_reward.Append(int64(val.Reward))
		f_max_reward.Append(uint64(val.MaxReward))
		f_max_att_reward.Append(uint64(val.AttestationReward))
		f_max_sync_reward.Append(uint64(val.SyncCommitteeReward))
		f_att_slot.Append(uint64(val.AttSlot))
		f_attestation_included.Append(val.AttestationIncluded)
		f_base_reward.Append(uint64(val.BaseReward))
		f_in_sync_committee.Append(val.InSyncCommittee)
		f_missing_source.Append(val.MissingSource)
		f_missing_target.Append(val.MissingTarget)
		f_missing_head.Append(val.MissingHead)
		f_status.Append(uint8(val.Status))
		f_block_api_reward.Append(uint64(val.ProposerApiReward))
		f_block_experimental_reward.Append(uint64(val.ProposerManualReward))
		f_inclusion_delay.Append(uint8(val.InclusionDelay))
	}

	return proto.Input{
		{Name: "f_val_idx", Data: f_val_idx},
		{Name: "f_epoch", Data: f_epoch},
		{Name: "f_balance_eth", Data: f_balance_eth},
		{Name: "f_effective_balance", Data: f_effective_balance},
		{Name: "f_withdrawal_prefix", Data: f_withdrawal_prefix},
		{Name: "f_reward", Data: f_reward},
		{Name: "f_max_reward", Data: f_max_reward},
		{Name: "f_max_att_reward", Data: f_max_att_reward},
		{Name: "f_max_sync_reward", Data: f_max_sync_reward},
		{Name: "f_att_slot", Data: f_att_slot},
		{Name: "f_attestation_included", Data: f_attestation_included},
		{Name: "f_base_reward", Data: f_base_reward},
		{Name: "f_in_sync_committee", Data: f_in_sync_committee},
		{Name: "f_missing_source", Data: f_missing_source},
		{Name: "f_missing_target", Data: f_missing_target},
		{Name: "f_missing_head", Data: f_missing_head},
		{Name: "f_status", Data: f_status},
		{Name: "f_block_api_reward", Data: f_block_api_reward},
		{Name: "f_block_experimental_reward", Data: f_block_experimental_reward},
		{Name: "f_inclusion_delay", Data: f_inclusion_delay},
	}
}

func (p *DBService) PersistValidatorRewards(data []spec.ValidatorRewards) error {
	persistObj := PersistableObject[spec.ValidatorRewards]{
		input: rewardsInput,
		table: valRewardsTable,
		query: insertValidatorRewardsQuery,
	}

	for _, item := range data {
		persistObj.Append(item)
	}

	err := p.Persist(persistObj.ExportPersist())
	if err != nil {
		log.Errorf("error persisting validator rewards: %s", err.Error())
	}
	return err
}

func (p *DBService) DeleteValidatorRewardsUntil(epoch phase0.Epoch) error {

	deleteObj := DeletableObject{
		query: deleteValidatorRewardsUntilEpochQuery,
		table: valRewardsTable,
		args:  []any{epoch},
	}

	err := p.Delete(deleteObj)
	if err != nil {
		log.Errorf("error deleting validator rewards: %s", err.Error())
	}

	return err
}
