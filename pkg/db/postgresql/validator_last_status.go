package postgresql

import (
	"github.com/cortze/eth-cl-state-analyzer/pkg/db/postgresql/model"
	"github.com/pkg/errors"
)

func (p *PostgresDBService) createLastStatusTable() error {
	// create the tables
	_, err := p.psqlPool.Exec(p.ctx, model.CreateLastValidatorStatusTable)
	if err != nil {
		return errors.Wrap(err, "error creating validator last status table")
	}
	return nil
}
