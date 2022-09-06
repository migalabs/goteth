package model

// Postgres intregration variables
var (
	CreateValidatorRewardsTable = `
	CREATE TABLE IF NOT EXISTS t_validator_rewards_summary(
		f_val_idx INT,
		f_slot INT,
		f_epoch INT,
		f_balance BIGINT,
		f_reward NUMERIC,
		f_max_reward NUMERIC,
		f_max_att_reward NUMERIC,
		f_max_incl_delay_reward NUMERIC,
		f_max_flag_index_reward NUMERIC,
		f_max_sync_committee_reward NUMERIC,
		f_att_slot BIGINT,
		f_att_inclusion_slot BIGINT,
		f_base_reward NUMERIC,
		f_missing_source BOOL,
		f_missing_target BOOL, 
		f_missing_head BOOL,
		CONSTRAINT PK_ValidatorSlot PRIMARY KEY (f_val_idx,f_slot));`

	InsertNewValidatorLineTable = `
	INSERT INTO t_validator_rewards_summary (	
		f_val_idx, 
		f_slot, 
		f_epoch, 
		f_balance, 
		f_reward, 
		f_max_reward, 
		f_max_att_reward,
		f_max_incl_delay_reward,
		f_max_flag_index_reward,
		f_max_sync_committee_reward,
		f_att_slot, 
		f_att_inclusion_slot, 
		f_base_reward,
		f_missing_source,
		f_missing_target,
		f_missing_head)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16);
	`

	UpdateValidatorLineTable = `
	UPDATE t_validator_rewards_summary
	SET f_reward=$3
	WHERE f_val_idx=$1 AND f_slot=$2
	`

	VALIDATOR_QUERIES = [...]string{InsertNewEpochLineTable, UpdateValidatorLineTable}
)

type ValidatorRewards struct {
	ValidatorIndex       uint64
	Slot                 uint64
	Epoch                uint64
	ValidatorBalance     uint64
	Reward               int64
	MaxReward            float64
	AttestationReward    float64
	InclusionDelayReward float64
	FlagIndexReward      float64
	SyncCommitteeReward  float64
	BaseReward           float64
	AttSlot              int64
	InclusionDelay       int64

	MissingSource bool
	MissingTarget bool
	MissingHead   bool
}

func NewValidatorRewards(
	iValIdx uint64,
	iSlot uint64,
	iEpoch uint64,
	iValBal uint64,
	iReward int64,
	iMaxReward float64,
	iMaxAttReward float64,
	iMaxInDelayReward float64,
	iMaxFlagReward float64,
	iMaxSyncComReward float64,
	iAttSlot int64,
	iInclusionDelay int64,
	iBaseReward float64,
	iMissingSource bool,
	iMissingTarget bool,
	iMissingHead bool) ValidatorRewards {
	return ValidatorRewards{
		ValidatorIndex:       iValIdx,
		Slot:                 iSlot,
		Epoch:                iEpoch,
		ValidatorBalance:     iValBal,
		Reward:               iReward,
		MaxReward:            iMaxReward,
		AttestationReward:    iMaxAttReward,
		InclusionDelayReward: iMaxInDelayReward,
		FlagIndexReward:      iMaxFlagReward,
		SyncCommitteeReward:  iMaxSyncComReward,
		AttSlot:              iAttSlot,
		InclusionDelay:       iInclusionDelay,
		BaseReward:           iBaseReward,
		MissingSource:        iMissingSource,
		MissingTarget:        iMissingTarget,
		MissingHead:          iMissingHead,
	}
}

func NewEmptyValidatorRewards() ValidatorRewards {
	return ValidatorRewards{}
}
