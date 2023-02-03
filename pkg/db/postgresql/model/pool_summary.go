package model

// Postgres intregration variables
var (
	CreatePoolSummaryTable = `
	CREATE TABLE IF NOT EXISTS t_pool_summary(
		f_pool_name TEXT,
		f_epoch INT,
		f_avg_reward INT,
		f_avg_max_reward INT,
		f_avg_max_att_reward INT,
		f_avg_max_sync_reward INT,
		f_avg_base_reward INT,
		f_sum_missing_source BOOL,
		f_sum_missing_target BOOL, 
		f_sum_missing_head BOOL,
		f_num_vals INT,
		CONSTRAINT PK_ValidatorSlot PRIMARY KEY (f_pool_name,f_epoch));`

	UpsertPoolSummary = `
	INSERT INTO t_validator_rewards_summary (	
		f_epoch,
		f_pool_name,
		f_avg_reward,
		f_avg_max_reward,
		f_avg_max_att_reward,
		f_avg_max_sync_reward,
		f_avg_base_reward,
		f_sum_missing_source,
		f_sum_missing_target  
		f_sum_missing_head,
		f_num_vals)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	ON CONFLICT ON CONSTRAINT PK_ValidatorSlot
		DO 
			UPDATE SET 
			f_epoch = excluded.f_epoch,,
			f_pool_name = excluded.f_pool_name,
			f_avg_reward = excluded.f_avg_reward,
			f_avg_max_reward = excluded.f_avg_max_reward,
			f_avg_max_att_reward = excluded.f_avg_max_att_reward,
			f_avg_max_sync_reward = excluded.f_avg_max_sync_reward,
			f_avg_base_reward = excluded.f_avg_base_reward,
			f_sum_missing_source = excluded.f_sum_missing_source,
			f_sum_missing_target  = excluded.f_sum_missing_target  
			f_sum_missing_head = excluded.f_sum_missing_head,
			f_num_vals = excluded.f_num_val;
	`
)

type PoolSummary struct {
	PoolName             string
	Epoch                int
	Reward               int64
	MaxReward            uint64
	AttestationReward    uint64
	InclusionDelayReward uint64
	FlagIndexReward      uint64
	SyncCommitteeReward  uint64
	BaseReward           uint64
	MissingSource        bool
	MissingTarget        bool
	MissingHead          bool
	NumVals              int
}

func NewPoolSummary(
	iPoolName string,
	iEpoch uint64,
	iReward int64,
	iMaxReward uint64,
	iMaxAttReward uint64,
	iMaxInDelayReward uint64,
	iMaxFlagReward uint64,
	iMaxSyncComReward uint64,
	iBaseReward uint64,
	iMissingSource bool,
	iMissingTarget bool,
	iMissingHead bool,
	iNumVal int) PoolSummary {
	return PoolSummary{
		PoolName:             iPoolName,
		Epoch:                int(iEpoch),
		Reward:               iReward,
		MaxReward:            iMaxReward,
		AttestationReward:    iMaxAttReward,
		InclusionDelayReward: iMaxInDelayReward,
		FlagIndexReward:      iMaxFlagReward,
		SyncCommitteeReward:  iMaxSyncComReward,
		BaseReward:           iBaseReward,
		MissingSource:        iMissingSource,
		MissingTarget:        iMissingTarget,
		MissingHead:          iMissingHead,
		NumVals:              iNumVal,
	}
}
