package db

import (
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/cortze/eth-cl-state-analyzer/pkg/spec"
)

// Postgres intregration variables
var (
	InsertValidator = `
	INSERT INTO t_validator_rewards_summary (	
		f_val_idx, 
		f_epoch, 
		f_balance_eth, 
		f_reward, 
		f_max_reward,
		f_max_att_reward,
		f_max_sync_reward,
		f_att_slot,
		f_base_reward,
		f_in_sync_committee,
		f_missing_source,
		f_missing_target,
		f_missing_head,
		f_status)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	ON CONFLICT ON CONSTRAINT t_validator_rewards_summary_pkey
		DO NOTHING;
	`

	DropValidatorRewardsQuery = `
		DROP FROM t_validator_rewards_summary
		WHERE f_epoch >= $1;
	`
)

func insertValidator(inputValidator spec.ValidatorRewards) (string, []interface{}) {
	resultArgs := make([]interface{}, 0)
	resultArgs = append(resultArgs, inputValidator.ValidatorIndex)
	resultArgs = append(resultArgs, inputValidator.Epoch)
	resultArgs = append(resultArgs, inputValidator.BalanceToEth())
	resultArgs = append(resultArgs, inputValidator.Reward)
	resultArgs = append(resultArgs, inputValidator.MaxReward)
	resultArgs = append(resultArgs, inputValidator.AttestationReward)
	resultArgs = append(resultArgs, inputValidator.SyncCommitteeReward)
	resultArgs = append(resultArgs, inputValidator.AttSlot)
	resultArgs = append(resultArgs, inputValidator.BaseReward)
	resultArgs = append(resultArgs, inputValidator.InSyncCommittee)
	resultArgs = append(resultArgs, inputValidator.MissingSource)
	resultArgs = append(resultArgs, inputValidator.MissingTarget)
	resultArgs = append(resultArgs, inputValidator.MissingHead)
	resultArgs = append(resultArgs, inputValidator.Status)
	return InsertValidator, resultArgs
}

func ValidatorOperation(inputValidator spec.ValidatorRewards) (string, []interface{}) {

	q, args := insertValidator(inputValidator)
	return q, args
}

type ValidatorRewardsDropType phase0.Epoch

func (s ValidatorRewardsDropType) Type() spec.ModelType {
	return spec.ValidatorRewardDropModel
}

func DropValidatorRewards(epoch ValidatorRewardsDropType) (string, []interface{}) {
	resultArgs := make([]interface{}, 0)
	resultArgs = append(resultArgs, epoch)
	return DropValidatorRewardsQuery, resultArgs
}
