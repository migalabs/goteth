package db

import (
	"github.com/migalabs/goteth/pkg/spec"
)

// Postgres intregration variables
var (
	UpsertPoolSummary = `
	INSERT INTO t_pool_summary (
		f_pool_name,
		f_epoch,
		f_reward,
		f_max_reward,
		f_max_att_reward,
		f_max_sync_reward,
		f_base_reward,
		f_sum_missing_source,
		f_sum_missing_target,
		f_sum_missing_head,
		f_num_active_vals,
		f_sync_vals)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	ON CONFLICT ON CONSTRAINT PK_EpochPool
		DO 
			UPDATE SET 
			f_reward = excluded.f_reward,
			f_max_reward = excluded.f_max_reward,
			f_max_att_reward = excluded.f_max_att_reward,
			f_max_sync_reward = excluded.f_max_sync_reward,
			f_base_reward = excluded.f_base_reward,
			f_sum_missing_source = excluded.f_sum_missing_source,
			f_sum_missing_target  = excluded.f_sum_missing_target,  
			f_sum_missing_head = excluded.f_sum_missing_head,
			f_num_active_vals = excluded.f_num_active_vals,
			f_sync_vals = excluded.f_sync_vals;
	`
)

func insertPool(inputPool spec.PoolSummary) (string, []interface{}) {
	resultArgs := make([]interface{}, 0)

	if len(inputPool.ValidatorList) > 0 {
		reward := float32(0)
		maxReward := float32(0)
		baseReward := float32(0)
		attMaxReward := float32(0)
		syncMaxReward := float32(0)
		missingSource := 0
		missingTarget := 0
		missingHead := 0

		numSyncVals := 0
		numActiveVals := 0
		// numProposers := 0

		// calculate averages
		for _, item := range inputPool.ValidatorList {
			reward += float32(item.Reward)
			maxReward += float32(item.MaxReward)
			baseReward += float32(item.BaseReward)
			attMaxReward += float32(item.AttestationReward)
			syncMaxReward += float32(item.SyncCommitteeReward)

			if item.MissingSource {
				missingSource += 1
			}
			if item.MissingTarget {
				missingTarget += 1
			}
			if item.MissingHead {
				missingHead += 1
			}

			if item.Status == spec.ACTIVE_STATUS {
				numActiveVals += 1
			}
			if item.InSyncCommittee {
				numSyncVals += 1
			}
		}

		resultArgs = append(resultArgs, inputPool.PoolName)
		resultArgs = append(resultArgs, inputPool.Epoch)
		resultArgs = append(resultArgs, reward)
		resultArgs = append(resultArgs, maxReward)
		resultArgs = append(resultArgs, attMaxReward)
		resultArgs = append(resultArgs, syncMaxReward)
		resultArgs = append(resultArgs, baseReward)
		resultArgs = append(resultArgs, missingSource)
		resultArgs = append(resultArgs, missingTarget)
		resultArgs = append(resultArgs, missingHead)
		resultArgs = append(resultArgs, numActiveVals)
		resultArgs = append(resultArgs, numSyncVals)
	}

	return UpsertPoolSummary, resultArgs
}

func PoolOperation(inputPool spec.PoolSummary) (string, []interface{}) {

	q, args := insertPool(inputPool)
	return q, args

}
