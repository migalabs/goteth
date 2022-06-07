package postgresql

import (
	"context"

	"github.com/pkg/errors"

	"github.com/jackc/pgx/v4/pgxpool"
)

// Postgres intregration variables
var (
	createValidatorRewardsTable = `
	CREATE TABLE IF NOT EXISTS t_validator_rewards_summary(
		f_val_idx TEXT,
		f_slot TEXT,
		f_balance TEXT,
		CONSTRAINT PK_ValidatorSlot PRIMARY KEY (f_val_idx,f_slot));`

	// insertNewLineTable = `
	// INSERT INTO t_validator_rewards_summary (f_val_idx, f_slot, f_balance)
	// VALUES ($1, $2, $3)
	// `
)

type ValidatorRewards struct {
	validatorIndex   uint64
	slot             uint64
	validatorBalance uint64
}

func NewValidatorRewards(i_valIdx uint64, i_slot uint64, i_valBal uint64) ValidatorRewards {
	return ValidatorRewards{
		validatorIndex:   i_valIdx,
		slot:             i_slot,
		validatorBalance: i_valBal,
	}
}

func NewEmptyValidatorRewards() ValidatorRewards {
	return ValidatorRewards{}
}

func (p *ValidatorRewards) init(ctx context.Context, pool *pgxpool.Pool) error {
	// create the tables
	err := p.createRewardsTable(ctx, pool)
	if err != nil {
		return err
	}
	return nil
}

func (p *ValidatorRewards) createRewardsTable(ctx context.Context, pool *pgxpool.Pool) error {
	// create the tables
	_, err := pool.Exec(ctx, createValidatorRewardsTable)
	if err != nil {
		return errors.Wrap(err, "error creating peer-message-metrics table")
	}
	return nil
}

// func (p *ValidatorRewards) InsertNewData(model.SingleEpochMetrics) error {
// 	// create the tables
// 	_, err := p.pool.Exec(p.ctx, createValidatorRewardsTable)
// 	if err != nil {
// 		return errors.Wrap(err, "error creating peer-message-metrics table")
// 	}
// 	return nil
// }
