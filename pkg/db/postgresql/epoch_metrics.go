package postgresql

/*

This file together with the model, has all the needed methods to interact with the epoch_metrics table of the database

*/

import (
	"context"

	"github.com/cortze/eth2-state-analyzer/pkg/db/postgresql/model"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/pkg/errors"
)

// in case the table did not exist
func (p *PostgresDBService) createEpochMetricsTable(ctx context.Context, pool *pgxpool.Pool) error {
	// create the tables
	_, err := pool.Exec(ctx, model.CreateEpochMetricsTable)
	if err != nil {
		return errors.Wrap(err, "error creating epoch metrics table")
	}
	return nil
}
