package model

// Postgres intregration variables
var (
	CreateValidatorRewardsTable = `
	CREATE TABLE IF NOT EXISTS t_validator_rewards_summary(
		f_val_idx INT,
		f_slot INT,
		f_balance BIGINT,
		CONSTRAINT PK_ValidatorSlot PRIMARY KEY (f_val_idx,f_slot));`

	InsertNewLineTable = `
	INSERT INTO t_validator_rewards_summary (f_val_idx, f_slot, f_balance)
	VALUES ($1, $2, $3);
	`

	SelectByVal = `
	SELECT f_val_idx, f_slot, f_balance FROM t_validator_rewards_summary
	WHERE f_val_idx=$1 AND f_slot=$2;
	`
)

type ValidatorRewards struct {
	ValidatorIndex   uint64
	Slot             uint64
	ValidatorBalance uint64
}

func NewValidatorRewards(iValIdx uint64, iSlot uint64, iValBal uint64) ValidatorRewards {
	return ValidatorRewards{
		ValidatorIndex:   iValIdx,
		Slot:             iSlot,
		ValidatorBalance: iValBal,
	}
}

func NewValidatorRewardsFromSingleEpochMetrics(iMetrics SingleEpochMetrics) ValidatorRewards {
	return NewValidatorRewards(iMetrics.ValidatorIdx, iMetrics.Slot, iMetrics.ValidatorBalance)
}

func NewEmptyValidatorRewards() ValidatorRewards {
	return ValidatorRewards{}
}
