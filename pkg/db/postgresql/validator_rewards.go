package postgresql

import (
	"context"

	"github.com/cortze/eth2-state-analyzer/pkg/db/postgresql/model"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/pkg/errors"
)

func (p *PostgresDBService) createRewardsTable(ctx context.Context, pool *pgxpool.Pool) error {
	// create the tables
	_, err := pool.Exec(ctx, model.CreateValidatorRewardsTable)
	if err != nil {
		return errors.Wrap(err, "error creating rewards table")
	}
	return nil
}
