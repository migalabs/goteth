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
		CONSTRAINT PK_Slot PRIMARY KEY (f_slot));`

	UpsertBlock = `
	INSERT INTO t_block_metrics (
		f_epoch, 
		f_slot,
		f_graffiti,
		f_proposer_index)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT ON CONSTRAINT PK_Slot
		DO
			UPDATE SET
				f_epoch = excluded.f_epoch,
				f_graffiti = excluded.f_graffiti,
				f_proposer_index = excluded.f_proposer_index;
	`
)

type BlockMetrics struct {
	Epoch         uint64
	Slot          uint64
	Graffiti      string
	ProposerIndex uint64
}

func NewBlockMetrics(iEpoch uint64,
	iSlot uint64,
	iGraffiti []byte,
	iProposerIndex uint64) BlockMetrics {

	graffiti := strings.ReplaceAll(string(iGraffiti), "\u0000", "")

	return BlockMetrics{
		Epoch:         iEpoch,
		Slot:          iSlot,
		Graffiti:      graffiti,
		ProposerIndex: iProposerIndex,
	}
}

func NewEmptyBlockMetrics() BlockMetrics {
	return BlockMetrics{}
}
