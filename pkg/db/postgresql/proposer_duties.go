package postgresql

/*

This file together with the model, has all the needed methods to interact with the epoch_metrics table of the database

*/

import (
	"github.com/cortze/eth-cl-state-analyzer/pkg/db/postgresql/model"
	"github.com/pkg/errors"
)

// in case the table did not exist
func (p *PostgresDBService) createProposerDutiesTable() error {
	// create the tables
	_, err := p.psqlPool.Exec(p.ctx, model.CreateProposerDutiesTable)
	if err != nil {
		return errors.Wrap(err, "error creating proposer duties table")
	}
	return nil
}
