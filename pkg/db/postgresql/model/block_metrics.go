package model

import "strings"

// Postgres intregration variables
var (
	CreateBlockMetricsTable = `
	CREATE TABLE IF NOT EXISTS t_block_metrics(
		f_epoch INT,
		f_slot INT,
		f_graffiti TEXT,
		f_proposer_index INT,
		f_proposed BOOL,
		CONSTRAINT PK_Slot PRIMARY KEY (f_slot));`

	UpsertBlock = `
	INSERT INTO t_block_metrics (
		f_epoch, 
		f_slot,
		f_graffiti,
		f_proposer_index,
		f_proposed)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT ON CONSTRAINT PK_Slot
		DO
			UPDATE SET
				f_epoch = excluded.f_epoch,
				f_graffiti = excluded.f_graffiti,
				f_proposer_index = excluded.f_proposer_index,
				f_proposed = excluded.f_proposed;
	`
)

type BlockMetrics struct {
	Epoch         uint64
	Slot          uint64
	Graffiti      string
	ProposerIndex uint64
	Proposed      bool
}

func NewBlockMetrics(iEpoch uint64,
	iSlot uint64,
	iGraffiti []byte,
	iProposerIndex uint64,
	iProposed bool) BlockMetrics {

	graffiti := strings.ReplaceAll(string(iGraffiti), "\u0000", "")

	return BlockMetrics{
		Epoch:         iEpoch,
		Slot:          iSlot,
		Graffiti:      graffiti,
		ProposerIndex: iProposerIndex,
		Proposed:      iProposed,
	}
}

func NewEmptyBlockMetrics() BlockMetrics {
	return BlockMetrics{}
}
