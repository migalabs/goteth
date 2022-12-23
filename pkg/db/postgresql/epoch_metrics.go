package postgresql

/*

This file together with the model, has all the needed methods to interact with the epoch_metrics table of the database

*/

import (
	"github.com/cortze/eth2-state-analyzer/pkg/db/postgresql/model"
	"github.com/pkg/errors"
)

// in case the table did not exist
func (p *PostgresDBService) createEpochMetricsTable() error {
	// create the tables
	_, err := p.psqlPool.Exec(p.ctx, model.CreateEpochMetricsTable)
	if err != nil {
		return errors.Wrap(err, "error creating epoch metrics table")
	}
	return nil
}

// in case the table did not exist
func (p *PostgresDBService) ObtainLastEpoch() (int, error) {
	// create the tables
	rows, err := p.psqlPool.Query(p.ctx, model.SelectLastEpoch)
	if err != nil {
		return -1, errors.Wrap(err, "error creating epoch metrics table")
	}
	epoch := -1
	rows.Next()
	rows.Scan(&epoch)
	return epoch, nil
}
