package model

import "github.com/cortze/eth2-state-analyzer/pkg/custom_spec"

// Postgres intregration variables
var (
	CreateValidatorRewardsTable = `
	CREATE TABLE IF NOT EXISTS t_validator_rewards_summary(
		f_val_idx INT,
		f_slot INT,
		f_epoch INT,
		f_balance_eth REAL,
		f_reward INT,
		f_max_reward INT,
		f_att_slot INT,
		f_att_inclusion_slot INT,
		f_base_reward INT,
		f_in_sync_committee BOOL,
		f_proposer_slot INT,
		f_missing_source BOOL,
		f_missing_target BOOL, 
		f_missing_head BOOL,
		CONSTRAINT PK_ValidatorSlot PRIMARY KEY (f_val_idx,f_slot));`

	InsertNewValidatorLineTable = `
	INSERT INTO t_validator_rewards_summary (	
		f_val_idx, 
		f_slot, 
		f_epoch, 
		f_balance_eth, 
		f_reward, 
		f_max_reward, 
		f_att_slot, 
		f_att_inclusion_slot, 
		f_base_reward,
		f_in_sync_committee,
		f_proposer_slot,
		f_missing_source,
		f_missing_target,
		f_missing_head)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14);
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
	Slot                 int
	Epoch                int
	ValidatorBalance     float32
	Reward               int
	MaxReward            int
	AttestationReward    int
	InclusionDelayReward int
	FlagIndexReward      int
	SyncCommitteeReward  int
	BaseReward           int
	AttSlot              int
	InclusionDelay       int
	InSyncCommittee      bool
	ProposerSlot         int
	MissingSource        bool
	MissingTarget        bool
	MissingHead          bool
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
	iSyncCommittee bool,
	iProposerSlot float64,
	iMissingSource bool,
	iMissingTarget bool,
	iMissingHead bool) ValidatorRewards {
	return ValidatorRewards{
		ValidatorIndex:       iValIdx,
		Slot:                 int(iSlot),
		Epoch:                int(iEpoch),
		ValidatorBalance:     float32(iValBal) / float32(custom_spec.EFFECTIVE_BALANCE_INCREMENT),
		Reward:               int(iReward),
		MaxReward:            int(iMaxReward),
		AttestationReward:    int(iMaxAttReward),
		InclusionDelayReward: int(iMaxInDelayReward),
		FlagIndexReward:      int(iMaxFlagReward),
		SyncCommitteeReward:  int(iMaxSyncComReward),
		AttSlot:              int(iAttSlot),
		InclusionDelay:       int(iInclusionDelay),
		BaseReward:           int(iBaseReward),
		InSyncCommittee:      iSyncCommittee,
		ProposerSlot:         int(iProposerSlot),
		MissingSource:        iMissingSource,
		MissingTarget:        iMissingTarget,
		MissingHead:          iMissingHead,
	}
}

func NewEmptyValidatorRewards() ValidatorRewards {
	return ValidatorRewards{}
}
