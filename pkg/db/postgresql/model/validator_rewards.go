package model

import "github.com/cortze/eth-cl-state-analyzer/pkg/state_metrics/fork_state"

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
		f_max_att_reward INT,
		f_max_sync_reward INT,
		f_att_slot INT,
		f_base_reward INT,
		f_in_sync_committee BOOL,
		f_missing_source BOOL,
		f_missing_target BOOL, 
		f_missing_head BOOL,
		f_status SMALLINT,
		CONSTRAINT PK_ValidatorSlot PRIMARY KEY (f_val_idx,f_slot));`

	UpsertValidator = `
	INSERT INTO t_validator_rewards_summary (	
		f_val_idx, 
		f_slot, 
		f_epoch, 
		f_balance_eth, 
		f_reward, 
		f_max_reward,
		f_att_slot,
		f_base_reward,
		f_in_sync_committee,
		f_missing_source,
		f_missing_target,
		f_missing_head,
		f_status)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	ON CONFLICT ON CONSTRAINT PK_ValidatorSlot
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
)

type ValidatorRewards struct {
	ValidatorIndex       uint64
	Slot                 int
	Epoch                int
	ValidatorBalance     float32
	Reward               int64
	MaxReward            uint64
	AttestationReward    uint64
	InclusionDelayReward uint64
	FlagIndexReward      uint64
	SyncCommitteeReward  uint64
	BaseReward           uint64
	AttSlot              uint64
	InclusionDelay       int64
	InSyncCommittee      bool
	ProposerSlot         int64
	MissingSource        bool
	MissingTarget        bool
	MissingHead          bool
	Status               int
}

func NewValidatorRewards(
	iValIdx uint64,
	iSlot uint64,
	iEpoch uint64,
	iValBal uint64,
	iReward int64,
	iMaxReward uint64,
	iMaxAttReward uint64,
	iMaxInDelayReward uint64,
	iMaxFlagReward uint64,
	iMaxSyncComReward uint64,
	iAttSlot uint64,
	iInclusionDelay int64,
	iBaseReward uint64,
	iSyncCommittee bool,
	iProposerSlot int64,
	iMissingSource bool,
	iMissingTarget bool,
	iMissingHead bool,
	iStatus int) ValidatorRewards {
	return ValidatorRewards{
		ValidatorIndex:       iValIdx,
		Slot:                 int(iSlot),
		Epoch:                int(iEpoch),
		ValidatorBalance:     float32(iValBal) / float32(fork_state.EFFECTIVE_BALANCE_INCREMENT),
		Reward:               iReward,
		MaxReward:            iMaxReward,
		AttestationReward:    iMaxAttReward,
		InclusionDelayReward: iMaxInDelayReward,
		FlagIndexReward:      iMaxFlagReward,
		SyncCommitteeReward:  iMaxSyncComReward,
		AttSlot:              iAttSlot,
		InclusionDelay:       iInclusionDelay,
		BaseReward:           iBaseReward,
		InSyncCommittee:      iSyncCommittee,
		ProposerSlot:         iProposerSlot,
		MissingSource:        iMissingSource,
		MissingTarget:        iMissingTarget,
		MissingHead:          iMissingHead,
		Status:               iStatus,
	}
}

func NewEmptyValidatorRewards() ValidatorRewards {
	return ValidatorRewards{}
}
