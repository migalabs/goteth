package postgresql

/*

This file together with the model, has all the needed methods to interact with the epoch_metrics table of the database

*/

import (
	"github.com/cortze/eth-cl-state-analyzer/pkg/db/postgresql/model"
	"github.com/cortze/eth-cl-state-analyzer/pkg/state_metrics/fork_state"
	"github.com/pkg/errors"
)

var ()

// in case the table did not exist
func (p *PostgresDBService) createStatusTable() error {
	// create the tables
	_, err := p.psqlPool.Exec(p.ctx, model.CreateStatusTable)
	if err != nil {
		return errors.Wrap(err, "error creating status table")
	}

	// insert status
	_, err = p.psqlPool.Exec(p.ctx, model.UpsertStatus, fork_state.QUEUE_STATUS, "in queue to activation")
	if err != nil {
		return errors.Wrap(err, "error inserting queue status")
	}

	_, err = p.psqlPool.Exec(p.ctx, model.UpsertStatus, fork_state.ACTIVE_STATUS, "active")
	if err != nil {
		return errors.Wrap(err, "error inserting active status")
	}

	_, err = p.psqlPool.Exec(p.ctx, model.UpsertStatus, fork_state.EXIT_STATUS, "exit")
	if err != nil {
		return errors.Wrap(err, "error inserting exit status")
	}

	_, err = p.psqlPool.Exec(p.ctx, model.UpsertStatus, fork_state.SLASHED_STATUS, "slashed")
	if err != nil {
		return errors.Wrap(err, "error inserting slashed status")
	}

	return nil
}
