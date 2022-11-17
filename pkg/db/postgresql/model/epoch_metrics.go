package model

import (
	"fmt"

	"github.com/cortze/eth2-state-analyzer/pkg/reward_metrics/fork_state"
)

// Postgres intregration variables
var (
	CreateEpochMetricsTable = `
	CREATE TABLE IF NOT EXISTS t_epoch_metrics_summary(
		f_epoch INT,
		f_slot INT,
		f_num_att INT,
		f_num_att_vals INT,
		f_num_vals INT,
		f_total_balance_eth REAL,
		f_att_effective_balance_eth REAL,
		f_total_effective_balance_eth REAL,
		f_missing_source INT, 
		f_missing_target INT,
		f_missing_head INT,
		CONSTRAINT PK_Epoch PRIMARY KEY (f_slot));`

	UpsertEpoch = `
	INSERT INTO t_epoch_metrics_summary (
		f_epoch, 
		f_slot, 
		f_num_att, 
		f_num_att_vals, 
		f_num_vals, 
		f_total_balance_eth,
		f_att_effective_balance_eth,  
		f_total_effective_balance_eth, 
		f_missing_source, 
		f_missing_target, 
		f_missing_head)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT ON CONSTRAINT PK_Epoch
		DO 
			UPDATE SET 
				f_num_att = excluded.f_num_att, 
				f_num_att_vals = excluded.f_num_att_vals,
				f_num_vals = excluded.f_num_vals,
				f_total_balance_eth = excluded.f_total_balance_eth,
				f_att_effective_balance_eth = excluded.f_att_effective_balance_eth,
				f_total_effective_balance_eth = excluded.f_total_effective_balance_eth,
				f_missing_source = excluded.f_missing_source,
				f_missing_target = excluded.f_missing_target,
				f_missing_head = excluded.f_missing_head;
	`
)

type EpochMetrics struct {
	Epoch                 uint64
	Slot                  uint64
	PrevNumAttestations   int
	PrevNumAttValidators  int
	PrevNumValidators     int
	TotalBalance          float32
	AttEffectiveBalance   float32
	TotalEffectiveBalance float32

	MissingSource int
	MissingTarget int
	MissingHead   int

	MissedBlocks string
}

func NewEpochMetrics(iEpoch uint64,
	iSlot uint64,
	iNumAtt uint64,
	iNumAttVals uint64,
	iNumVals uint64,
	iTotBal uint64,
	iAttEfBal uint64,
	iTotEfBal uint64,
	iMissingSource uint64,
	iMissingTarget uint64,
	iMissingHead uint64,
	iMissedBlocks []uint64) EpochMetrics {

	missedBlocks := "["
	for _, item := range iMissedBlocks {
		missedBlocks += fmt.Sprintf("%d", item) + ","
	}
	missedBlocks += "]"
	return EpochMetrics{
		Epoch:                 iEpoch,
		Slot:                  iSlot,
		PrevNumAttestations:   int(iNumAtt),
		PrevNumAttValidators:  int(iNumAttVals),
		PrevNumValidators:     int(iNumVals),
		TotalBalance:          float32(iTotBal) / float32(fork_state.EFFECTIVE_BALANCE_INCREMENT),
		AttEffectiveBalance:   float32(iAttEfBal) / float32(fork_state.EFFECTIVE_BALANCE_INCREMENT),
		TotalEffectiveBalance: float32(iTotEfBal) / float32(fork_state.EFFECTIVE_BALANCE_INCREMENT),
		MissingSource:         int(iMissingSource),
		MissingTarget:         int(iMissingTarget),
		MissingHead:           int(iMissingHead),
		MissedBlocks:          missedBlocks,
	}
}

func NewEmptyEpochMetrics() EpochMetrics {
	return EpochMetrics{}
}
