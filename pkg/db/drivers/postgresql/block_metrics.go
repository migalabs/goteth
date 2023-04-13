package postgresql

/*

This file together with the model, has all the needed methods to interact with the epoch_metrics table of the database

*/

import (
	"fmt"
	"strings"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/cortze/eth-cl-state-analyzer/pkg/db/model"
	"github.com/pkg/errors"
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
		DO NOTHING;
	`
	SelectLastSlot = `
	SELECT f_slot
	FROM t_block_metrics
	ORDER BY f_slot DESC
	LIMIT 1`
)

func insertBlock(inputBlock model.ForkBlockContentBase) (string, []interface{}) {
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

	return UpsertBlock, resultArgs
}

func BlockOperation(inputBlock model.ForkBlockContentBase, op string) (string, []interface{}, error) {

	if op == model.INSERT_OP {
		q, args := insertBlock(inputBlock)
		return q, args, nil
	}

	return "", nil, fmt.Errorf("block operation not permitted: %s", op)
}

// in case the table did not exist
func (p *PostgresDBService) createBlockMetricsTable() error {
	// create the tables
	_, err := p.psqlPool.Exec(p.ctx, CreateBlockMetricsTable)
	if err != nil {
		return errors.Wrap(err, "error creating block metrics table")
	}
	return nil
}

// in case the table did not exist
func (p *PostgresDBService) ObtainLastSlot() (phase0.Slot, error) {
	// create the tables
	rows, err := p.psqlPool.Query(p.ctx, SelectLastSlot)
	if err != nil {
		return 0, errors.Wrap(err, "error obtianing last block from database")
	}
	slot := -1
	rows.Next()
	rows.Scan(&slot)
	return phase0.Slot(slot), nil
}
