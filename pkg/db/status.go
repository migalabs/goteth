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
	CreateStatusTable = `
	CREATE TABLE IF NOT EXISTS t_status(
		f_id INT,
		f_status TEXT PRIMARY KEY);`

	UpsertStatus = `
	INSERT INTO t_status (
		f_id, 
		f_status)
		VALUES ($1, $2)
		ON CONFLICT ON CONSTRAINT t_status_pkey
		DO NOTHING
	`
)

// in case the table did not exist
func (p *PostgresDBService) createStatusTable() error {
	// create the tables
	_, err := p.psqlPool.Exec(p.ctx, CreateStatusTable)
	if err != nil {
		return errors.Wrap(err, "error creating status table")
	}

	// insert status
	_, err = p.psqlPool.Exec(p.ctx, UpsertStatus, model.QUEUE_STATUS, "in queue to activation")
	if err != nil {
		return errors.Wrap(err, "error inserting queue status")
	}

	_, err = p.psqlPool.Exec(p.ctx, UpsertStatus, model.ACTIVE_STATUS, "active")
	if err != nil {
		return errors.Wrap(err, "error inserting active status")
	}

	_, err = p.psqlPool.Exec(p.ctx, UpsertStatus, model.EXIT_STATUS, "exit")
	if err != nil {
		return errors.Wrap(err, "error inserting exit status")
	}

	_, err = p.psqlPool.Exec(p.ctx, UpsertStatus, model.SLASHED_STATUS, "slashed")
	if err != nil {
		return errors.Wrap(err, "error inserting slashed status")
	}

	return nil
}
