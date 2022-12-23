package postgresql

/*

This file together with the model, has all the needed methods to interact with the epoch_metrics table of the database

*/

import (
	"github.com/cortze/eth2-state-analyzer/pkg/db/postgresql/model"
	"github.com/pkg/errors"
)

// in case the table did not exist
func (p *PostgresDBService) createBlockMetricsTable() error {
	// create the tables
	_, err := p.psqlPool.Exec(p.ctx, model.CreateBlockMetricsTable)
	if err != nil {
		return errors.Wrap(err, "error creating block metrics table")
	}
	return nil
}
