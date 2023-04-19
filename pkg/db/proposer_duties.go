package db

/*

This file together with the model, has all the needed methods to interact with the epoch_metrics table of the database

*/

import (
	"github.com/cortze/eth-cl-state-analyzer/pkg/db/model"
	"github.com/pkg/errors"
)

// Postgres intregration variables
var (
	CreateProposerDutiesTable = `
	CREATE TABLE IF NOT EXISTS t_proposer_duties(
		f_val_idx INT,
		f_proposer_slot INT,
		f_proposed BOOL,
		CONSTRAINT PK_Val_Slot PRIMARY KEY (f_val_idx, f_proposer_slot));`

	InsertProposerDuty = `
	INSERT INTO t_proposer_duties (
		f_val_idx, 
		f_proposer_slot,
		f_proposed)
		VALUES ($1, $2, $3)
		ON CONFLICT DO NOTHING;
	`
	// if there is a confilct the line already exists
)

func insertProposerDuty(inputDuty model.ProposerDuty) (string, []interface{}) {
	resultArgs := make([]interface{}, 0)
	resultArgs = append(resultArgs, inputDuty.ValIdx)
	resultArgs = append(resultArgs, inputDuty.ProposerSlot)
	resultArgs = append(resultArgs, inputDuty.Proposed)

	return InsertProposerDuty, resultArgs
}

func ProposerDutyOperation(inputDuty model.ProposerDuty) (string, []interface{}) {

	q, args := insertProposerDuty(inputDuty)
	return q, args

}

// in case the table did not exist
func (p *PostgresDBService) createProposerDutiesTable() error {
	// create the tables
	_, err := p.psqlPool.Exec(p.ctx, CreateProposerDutiesTable)
	if err != nil {
		return errors.Wrap(err, "error creating proposer duties table")
	}
	return nil
}
