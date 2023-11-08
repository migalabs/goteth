package db

import (
	"time"

	pgx "github.com/jackc/pgx/v4"
	"github.com/migalabs/goteth/pkg/spec"
)

// Postgres intregration variables
var (
	UpsertValidatorLastStatus = `
	INSERT INTO t_validator_last_status (	
		f_val_idx, 
		f_epoch, 
		f_balance_eth, 
		f_status,
		f_slashed,
		f_activation_epoch,
		f_withdrawal_epoch,
		f_exit_epoch,
		f_public_key)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	ON CONFLICT ON CONSTRAINT t_validator_last_status_pkey
		DO 
			UPDATE SET 
				f_epoch = excluded.f_epoch, 
				f_balance_eth = excluded.f_balance_eth,
				f_status = excluded.f_status,
				f_slashed = excluded.f_slashed,
				f_activation_epoch = excluded.f_activation_epoch,
				f_withdrawal_epoch = excluded.f_withdrawal_epoch,
				f_exit_epoch = excluded.f_exit_epoch,
				f_public_key = excluded.f_public_key;
	`

	DropOldValidatorStatus = `
		DELETE FROM t_validator_last_status
		WHERE f_epoch < $1`
)

func insertValidatorLastStatus(inputValidator spec.ValidatorLastStatus) (string, []interface{}) {

	return UpsertValidatorLastStatus, inputValidator.ToArray()
}

func ValidatorLastStatusOperation(inputValidator spec.ValidatorLastStatus) (string, []interface{}) {

	q, args := insertValidatorLastStatus(inputValidator)
	return q, args

}

func (p *PostgresDBService) CopyValLastStatus(rowSrc [][]interface{}) int64 {

	startTime := time.Now()

	count, err := p.psqlPool.CopyFrom(
		p.ctx,
		pgx.Identifier{"t_validator_last_status"},
		[]string{"f_val_idx",
			"f_epoch",
			"f_balance_eth",
			"f_status",
			"f_slashed",
			"f_activation_epoch",
			"f_withdrawal_epoch",
			"f_exit_epoch",
			"f_public_key"},
		pgx.CopyFromRows(rowSrc))

	if err != nil {
		wlog.Fatalf("could not copy val_status rows into db: %s", err.Error())
	}

	wlog.Infof("persisted val_status %d rows in %f seconds", count, time.Since(startTime).Seconds())

	return count
}
