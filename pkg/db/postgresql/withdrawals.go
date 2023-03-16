package postgresql

import (
	"github.com/cortze/eth-cl-state-analyzer/pkg/db/postgresql/model"
	"github.com/pkg/errors"
)

func (p *PostgresDBService) createWithdrawalsTable() error {
	// create the tables
	_, err := p.psqlPool.Exec(p.ctx, model.CreateWithdrawalsTable)
	if err != nil {
		return errors.Wrap(err, "error creating withdrawals table")
	}
	return nil
}
