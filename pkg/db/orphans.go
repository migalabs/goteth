package db

/*

This file together with the model, has all the needed methods to interact with the epoch_metrics table of the database

*/

import (
	"strings"

	"github.com/cortze/eth-cl-state-analyzer/pkg/spec"
)

// Postgres intregration variables
var (
	InsertOrphan = `
	INSERT INTO t_orphans (
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
		f_el_block_number,
		f_size_bytes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)
		ON CONFLICT ON CONSTRAINT t_orphans_pkey
		DO NOTHING;
	`
)

func insertOrphan(inputBlock OrphanBlock) (string, []interface{}) {
	resultArgs := make([]interface{}, 0)

	resultArgs = append(resultArgs, inputBlock.ExecutionPayload.Timestamp)
	resultArgs = append(resultArgs, inputBlock.Slot/32)
	resultArgs = append(resultArgs, inputBlock.Slot)
	resultArgs = append(resultArgs, strings.ReplaceAll(string(inputBlock.Graffiti[:]), "\u0000", ""))
	resultArgs = append(resultArgs, inputBlock.ProposerIndex)
	resultArgs = append(resultArgs, inputBlock.Proposed)
	resultArgs = append(resultArgs, len(inputBlock.Attestations))
	resultArgs = append(resultArgs, len(inputBlock.Deposits))
	resultArgs = append(resultArgs, len(inputBlock.ProposerSlashings))
	resultArgs = append(resultArgs, len(inputBlock.AttesterSlashings))
	resultArgs = append(resultArgs, len(inputBlock.VoluntaryExits))
	resultArgs = append(resultArgs, inputBlock.SyncAggregate.SyncCommitteeBits.Count())
	resultArgs = append(resultArgs, inputBlock.ExecutionPayload.FeeRecipient.String())
	resultArgs = append(resultArgs, inputBlock.ExecutionPayload.GasLimit)
	resultArgs = append(resultArgs, inputBlock.ExecutionPayload.GasUsed)
	resultArgs = append(resultArgs, inputBlock.ExecutionPayload.BaseFeeToInt())
	resultArgs = append(resultArgs, inputBlock.ExecutionPayload.BlockHash.String())
	resultArgs = append(resultArgs, len(inputBlock.ExecutionPayload.Transactions))
	resultArgs = append(resultArgs, inputBlock.ExecutionPayload.BlockNumber)
	resultArgs = append(resultArgs, inputBlock.Size)

	return InsertOrphan, resultArgs
}

func OrphanOperation(inputBlock OrphanBlock) (string, []interface{}) {

	q, args := insertOrphan(inputBlock)
	return q, args

}

type OrphanBlock spec.AgnosticBlock

func (s OrphanBlock) Type() spec.ModelType {
	return spec.OrphanModel
}
