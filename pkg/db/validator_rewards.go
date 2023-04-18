package db

import (
	"github.com/cortze/eth-cl-state-analyzer/pkg/db/model"
	"github.com/pkg/errors"
)

// Postgres intregration variables
var (
	CreateValidatorRewardsTable = `
	CREATE TABLE IF NOT EXISTS t_validator_rewards_summary(
		f_val_idx INT,
		f_epoch INT,
		f_balance_eth REAL,
		f_reward BIGINT,
		f_max_reward INT,
		f_max_att_reward INT,
		f_max_sync_reward INT,
		f_att_slot INT,
		f_base_reward INT,
		f_in_sync_committee BOOL,
		f_missing_source BOOL,
		f_missing_target BOOL, 
		f_missing_head BOOL,
		f_status SMALLINT,
		CONSTRAINT t_validator_rewards_summary_pkey PRIMARY KEY (f_val_idx,f_epoch));`

	UpsertValidator = `
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
		DO 
			UPDATE SET 
				f_epoch = excluded.f_epoch, 
				f_balance_eth = excluded.f_balance_eth,
				f_reward = excluded.f_reward,
				f_max_reward = excluded.f_max_reward,
				f_att_slot = excluded.f_att_slot,
				f_base_reward = excluded.f_base_reward,
				f_in_sync_committee = excluded.f_in_sync_committee,
				f_missing_source = excluded.f_missing_source,
				f_missing_target = excluded.f_missing_target,
				f_missing_head = excluded.f_missing_head,
				f_status = excluded.f_status;
	`

	Drop = `
		DROP FROM t_validator_rewards_summary
		WHERE f_val_idx = $1 AND f_epoch = $2;
	`
)

func insertValidator(inputValidator model.ValidatorRewards) (string, []interface{}) {
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
	return UpsertValidator, resultArgs
}

func ValidatorOperation(inputValidator model.ValidatorRewards) (string, []interface{}) {

	q, args := insertValidator(inputValidator)
	return q, args
}

func (p *PostgresDBService) createRewardsTable() error {
	// create the tables
	_, err := p.psqlPool.Exec(p.ctx, CreateValidatorRewardsTable)
	if err != nil {
		return errors.Wrap(err, "error creating rewards table")
	}
	return nil
}
