package db

/*

This file together with the model, has all the needed methods to interact with the epoch_metrics table of the database

*/

import (
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/cortze/eth-cl-state-analyzer/pkg/spec"

	"github.com/pkg/errors"
)

// Postgres intregration variables
var (
	UpsertEpoch = `
	INSERT INTO t_epoch_metrics_summary (
		f_epoch, 
		f_slot, 
		f_num_att, 
		f_num_att_vals, 
		f_num_vals, 
		f_total_balance_eth,
		f_att_effective_balance_eth,  
		f_total_effective_balance_eth, 
		f_missing_source, 
		f_missing_target, 
		f_missing_head)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT ON CONSTRAINT PK_Epoch
		DO 
			UPDATE SET 
				f_num_att = excluded.f_num_att, 
				f_num_att_vals = excluded.f_num_att_vals,
				f_num_vals = excluded.f_num_vals,
				f_total_balance_eth = excluded.f_total_balance_eth,
				f_att_effective_balance_eth = excluded.f_att_effective_balance_eth,
				f_total_effective_balance_eth = excluded.f_total_effective_balance_eth,
				f_missing_source = excluded.f_missing_source,
				f_missing_target = excluded.f_missing_target,
				f_missing_head = excluded.f_missing_head;
	`
	SelectLastEpoch = `
		SELECT f_epoch
		FROM t_epoch_metrics_summary
		ORDER BY f_epoch DESC
		LIMIT 1`

	DropEpochsQuery = `
	DELETE FROM t_epoch_metrics_summary
	WHERE f_epoch >= $1;
`
)

func insertEpoch(inputEpoch spec.Epoch) (string, []interface{}) {
	resultArgs := make([]interface{}, 0)
	resultArgs = append(resultArgs, inputEpoch.Epoch)
	resultArgs = append(resultArgs, inputEpoch.Slot)
	resultArgs = append(resultArgs, inputEpoch.NumAttestations)
	resultArgs = append(resultArgs, inputEpoch.NumAttValidators)
	resultArgs = append(resultArgs, inputEpoch.NumValidators)
	resultArgs = append(resultArgs, inputEpoch.TotalBalance)
	resultArgs = append(resultArgs, inputEpoch.AttEffectiveBalance)
	resultArgs = append(resultArgs, inputEpoch.TotalEffectiveBalance)
	resultArgs = append(resultArgs, inputEpoch.MissingSource)
	resultArgs = append(resultArgs, inputEpoch.MissingTarget)
	resultArgs = append(resultArgs, inputEpoch.MissingHead)

	return UpsertEpoch, resultArgs
}

func EpochOperation(inputEpoch spec.Epoch) (string, []interface{}) {

	q, args := insertEpoch(inputEpoch)
	return q, args
}

// in case the table did not exist
func (p *PostgresDBService) ObtainLastEpoch() (phase0.Epoch, error) {
	// create the tables
	rows, err := p.psqlPool.Query(p.ctx, SelectLastEpoch)
	if err != nil {
		rows.Close()
		return phase0.Epoch(0), errors.Wrap(err, "error obtaining last epoch from database")
	}
	epoch := phase0.Epoch(0)
	rows.Next()
	rows.Scan(&epoch)
	rows.Close()
	return phase0.Epoch(epoch), nil
}

type EpochDropType phase0.Epoch

func (s EpochDropType) Type() spec.ModelType {
	return spec.EpochDropModel
}

func DropEpochs(epoch EpochDropType) (string, []interface{}) {
	resultArgs := make([]interface{}, 0)
	resultArgs = append(resultArgs, epoch)
	return DropEpochsQuery, resultArgs
}
