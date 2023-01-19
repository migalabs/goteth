package model

import "strings"

// Postgres intregration variables
var (
	CreateBlockMetricsTable = `
	CREATE TABLE IF NOT EXISTS t_block_metrics(
		f_timestamp INT,
		f_epoch INT,
		f_slot INT,
		f_graffiti TEXT,
		f_proposer_index INT,
		f_proposed BOOL,
		f_attestations INT,
		f_deposits INT,
		f_proposer_slashings INT,
		f_att_slashings INT,
		f_voluntary_exits INT,
		f_sync_bits INT,
		f_el_fee_recp TEXT,
		f_el_gas_limit INT,
		f_el_gas_used INT,
		f_el_base_fee_per_gas INT,
		f_el_block_hash TEXT,
		f_el_transactions INT,
		CONSTRAINT PK_Slot PRIMARY KEY (f_slot));`

	UpsertBlock = `
	INSERT INTO t_block_metrics (
		f_timestamp,
		f_epoch, 
		f_slot,
		f_graffiti,
		f_proposer_index,
		f_proposed,
		f_attestations,
		f_deposits,
		f_proposer_slashings,
		f_att_slashings,
		f_voluntary_exits,
		f_sync_bits,
		f_el_fee_recp,
		f_el_gas_limit,
		f_el_gas_used,
		f_el_base_fee_per_gas,
		f_el_block_hash,
		f_el_transactions)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
		ON CONFLICT ON CONSTRAINT PK_Slot
		DO
			UPDATE SET
				f_epoch = excluded.f_epoch,
				f_graffiti = excluded.f_graffiti,
				f_proposer_index = excluded.f_proposer_index,
				f_proposed = excluded.f_proposed,
				f_attestations = excluded.f_attestations,
				f_deposits = excluded.f_deposits,
				f_proposer_slashings = excluded.f_proposer_slashings,
				f_att_slashings = excluded.f_att_slashings,
				f_voluntary_exits = excluded.f_voluntary_exits,
				f_sync_bits = excluded.f_sync_bits,
				f_el_fee_recp = excluded.f_el_fee_recp,
				f_el_gas_limit = excluded.f_el_gas_limit,
				f_el_gas_used = excluded.f_el_gas_used,
				f_el_base_fee_per_gas = excluded.f_el_base_fee_per_gas,
				f_el_block_hash = excluded.f_el_block_hash,
				f_el_transactions = excluded.f_el_transactions;
	`
	SelectLastSlot = `
	SELECT f_slot
	FROM t_block_metrics
	ORDER BY f_slot DESC
	LIMIT 1`
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
	iGraffiti [32]byte,
	iProposerIndex uint64,
	iProposed bool) BlockMetrics {

	graffiti := strings.ReplaceAll(string(iGraffiti[:]), "\u0000", "")

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
