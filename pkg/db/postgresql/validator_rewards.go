package postgresql

import (
	"github.com/cortze/eth-cl-state-analyzer/pkg/db/postgresql/model"
	"github.com/pkg/errors"
)

func (p *PostgresDBService) createRewardsTable() error {
	// create the tables
	_, err := p.psqlPool.Exec(p.ctx, model.CreateValidatorRewardsTable)
	if err != nil {
		return errors.Wrap(err, "error creating rewards table")
	}
	return nil
}
