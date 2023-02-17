package model

import (
	"strings"

	"github.com/attestantio/go-eth2-client/spec/altair"
	"github.com/attestantio/go-eth2-client/spec/bellatrix"
	"github.com/attestantio/go-eth2-client/spec/phase0"
)

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
		f_el_block_number INT,
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
		f_el_transactions,
		f_el_block_number)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
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
				f_el_transactions = excluded.f_el_transactions,
				f_el_block_number = excluded.f_el_block_number;
	`
	SelectLastSlot = `
	SELECT f_slot
	FROM t_block_metrics
	ORDER BY f_slot DESC
	LIMIT 1`
)

type BlockMetrics struct {
	ELTimestamp       uint64
	Epoch             uint64
	Slot              uint64
	Graffiti          string
	ProposerIndex     uint64
	Proposed          bool
	Attestatons       uint64
	Deposits          uint64
	ProposerSlashings uint64
	AttSlashings      uint64
	VoluntaryExits    uint64
	SyncBits          uint64
	ELFeeRecp         string
	ELGasLimit        uint64
	ELGasUsed         uint64
	ELBaseFeePerGas   uint64
	ELBlockHash       string
	ELTransactions    uint64
	BlockNumber       uint64
}

func NewBlockMetrics(
	iELTimeStamp uint64,
	iEpoch uint64,
	iSlot uint64,
	iGraffiti [32]byte,
	iProposerIndex uint64,
	iProposed bool,
	iAttestatons []*phase0.Attestation,
	iDeposits []*phase0.Deposit,
	iProposerSlashings []*phase0.ProposerSlashing,
	iAttSlashings []*phase0.AttesterSlashing,
	iVoluntaryExits []*phase0.SignedVoluntaryExit,
	iSyncBits *altair.SyncAggregate,
	iELFeeRecp bellatrix.ExecutionAddress,
	iELGasLimit uint64,
	iELGasUsed uint64,
	iELBaseFeePerGas [32]byte,
	iELBlockHash phase0.Hash32,
	iELTransactions []bellatrix.Transaction,
	iBlockNumber uint64) BlockMetrics {

	graffiti := strings.ReplaceAll(string(iGraffiti[:]), "\u0000", "")

	return BlockMetrics{
		ELTimestamp:       iELTimeStamp,
		Epoch:             iEpoch,
		Slot:              iSlot,
		Graffiti:          graffiti,
		ProposerIndex:     iProposerIndex,
		Proposed:          iProposed,
		Attestatons:       uint64(len(iAttestatons)),
		Deposits:          uint64(len(iDeposits)),
		ProposerSlashings: uint64(len(iProposerSlashings)),
		AttSlashings:      uint64(len(iAttSlashings)),
		VoluntaryExits:    uint64(len(iVoluntaryExits)),
		SyncBits:          iSyncBits.SyncCommitteeBits.Count(),
		ELFeeRecp:         iELFeeRecp.String(),
		ELGasLimit:        iELGasLimit,
		ELGasUsed:         iELGasUsed,
		ELBaseFeePerGas:   0,
		ELBlockHash:       iELBlockHash.String(),
		ELTransactions:    uint64(len(iELTransactions)),
		BlockNumber:       iBlockNumber,
	}
}

func NewEmptyBlockMetrics() BlockMetrics {
	return BlockMetrics{}
}
