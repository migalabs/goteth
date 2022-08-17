package postgresql

/*

This file together with the model, has all the needed methods to interact with the epoch_metrics table of the database

*/

import (
	"context"

	"github.com/cortze/eth2-state-analyzer/pkg/model"
	"github.com/cortze/eth2-state-analyzer/pkg/utils"
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

func (p *PostgresDBService) InsertNewEpochRow(iEpochObj model.EpochMetrics) error {

	_, err := p.psqlPool.Exec(p.ctx, model.InsertNewEpochLineTable, iEpochObj.Epoch, iEpochObj.Slot, iEpochObj.PrevNumAttestations, iEpochObj.PrevNumValidators, iEpochObj.TotalBalance, iEpochObj.TotalEffectiveBalance, iEpochObj.MissingSource, iEpochObj.MissingTarget, iEpochObj.MissingHead, iEpochObj.MissedBlocks)

	if err != nil {
		return errors.Wrap(err, "error inserting row in epoch metrics table")
	}
	return nil
}

// to be checked if we need it
func (p *PostgresDBService) UpdatePrevEpochMetrics(iEpochObj model.EpochMetrics) error {

	if iEpochObj.Slot > utils.SlotBase {
		log.Debugf("updating row %d from epoch metrics", iEpochObj.Slot-utils.SlotBase)
		_, err := p.psqlPool.Exec(p.ctx, model.UpdateRow, iEpochObj.PrevNumAttestations, iEpochObj.PrevNumValidators, iEpochObj.Slot-utils.SlotBase, iEpochObj.TotalBalance, iEpochObj.TotalEffectiveBalance, iEpochObj.MissingSource, iEpochObj.MissingTarget, iEpochObj.MissingHead)
		if err != nil {
			return errors.Wrap(err, "error updating row in epoch metrics table")
		}
		return nil
	} else {
		log.Debugf("not updating row as we are in the first epoch")
		return nil
	}

}

func (p *PostgresDBService) GetEpochRow(iEpoch uint64) (model.EpochMetrics, error) {

	row := p.psqlPool.QueryRow(p.ctx, model.SelectByEpoch, iEpoch)
	epochRow := model.NewEmptyEpochMetrics()

	err := row.Scan(&epochRow.Epoch, &epochRow.Slot, &epochRow.PrevNumAttestations, &epochRow.PrevNumValidators)

	if err != nil {
		return model.NewEmptyEpochMetrics(), errors.Wrap(err, "error retrieving row from epoch metrics table")
	}
	return epochRow, nil
}
