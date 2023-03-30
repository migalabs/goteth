package model

import "github.com/cortze/eth-cl-state-analyzer/pkg/state_metrics/fork_state"

// Postgres intregration variables
var (
	CreateLastValidatorStatusTable = `
	CREATE TABLE IF NOT EXISTS t_validator_last_status(
		f_val_idx INT PRIMARY KEY,
		f_epoch INT,
		f_balance_eth REAL,
		f_status SMALLINT);`

	UpsertValidatorLastStatus = `
	INSERT INTO t_validator_last_status (	
		f_val_idx, 
		f_epoch, 
		f_balance_eth, 
		f_status)
	VALUES ($1, $2, $3, $4)
	ON CONFLICT ON CONSTRAINT t_validator_last_status_pkey
		DO 
			UPDATE SET 
				f_epoch = excluded.f_epoch, 
				f_balance_eth = excluded.f_balance_eth,
				f_status = excluded.f_status;
	`
)

type ValidatorLastStatus struct {
	ValidatorIndex   uint64
	Epoch            int
	ValidatorBalance float32
	Status           int
}

func NewValidatorLastStatus(
	iValIdx uint64,
	iEpoch uint64,
	iValBal uint64,
	iStatus int) ValidatorLastStatus {
	return ValidatorLastStatus{
		ValidatorIndex:   iValIdx,
		Epoch:            int(iEpoch),
		ValidatorBalance: float32(iValBal) / float32(fork_state.EFFECTIVE_BALANCE_INCREMENT),
		Status:           iStatus,
	}
}

func NewEmptyValidatorLastStatus() ValidatorLastStatus {
	return ValidatorLastStatus{}
}
