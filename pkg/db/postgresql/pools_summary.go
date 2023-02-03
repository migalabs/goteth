package postgresql

import (
	"github.com/cortze/eth-cl-state-analyzer/pkg/db/postgresql/model"
	"github.com/pkg/errors"
)

func (p *PostgresDBService) createPoolsTable() error {
	// create the tables
	_, err := p.psqlPool.Exec(p.ctx, model.CreatePoolSummaryTable)
	if err != nil {
		return errors.Wrap(err, "error creating pools table")
	}
	return nil
}
