package model

// Postgres intregration variables
var (
	CreateValidatorRewardsTable = `
	CREATE TABLE IF NOT EXISTS t_validator_rewards_summary(
		f_val_idx INT,
		f_slot INT,
		f_epoch INT,
		f_balance BIGINT,
		f_reward BIGINT,
		f_max_reward BIGINT,
		CONSTRAINT PK_ValidatorSlot PRIMARY KEY (f_val_idx,f_slot));`

	InsertNewValidatorLineTable = `
	INSERT INTO t_validator_rewards_summary (f_val_idx, f_slot, f_epoch, f_balance, f_reward, f_max_reward)
	VALUES ($1, $2, $3, $4, $5, $6);
	`

	SelectByValSlot = `
	SELECT f_val_idx, f_slot, f_epoch, f_balance, f_reward, f_max_reward FROM t_validator_rewards_summary
	WHERE f_val_idx=$1 AND f_slot=$2;
	`
)

type ValidatorRewards struct {
	ValidatorIndex   uint64
	Slot             uint64
	Epoch            uint64
	ValidatorBalance uint64
	Reward           uint64
	MaxReward        uint64
}

func NewValidatorRewards(iValIdx uint64, iSlot uint64, iEpoch uint64, iValBal uint64, iReward uint64, iMaxReward uint64) ValidatorRewards {
	return ValidatorRewards{
		ValidatorIndex:   iValIdx,
		Slot:             iSlot,
		Epoch:            iEpoch,
		ValidatorBalance: iValBal,
		Reward:           iReward,
		MaxReward:        iMaxReward,
	}
}

func NewValidatorRewardsFromSingleEpochMetrics(iMetrics SingleEpochMetrics) ValidatorRewards {
	return NewValidatorRewards(iMetrics.ValidatorIdx, iMetrics.Slot, iMetrics.Epoch, iMetrics.ValidatorBalance, iMetrics.Reward, iMetrics.MaxReward)
}

func NewEmptyValidatorRewards() ValidatorRewards {
	return ValidatorRewards{}
}
