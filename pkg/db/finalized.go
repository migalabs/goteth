package db

/*

This file together with the model, has all the needed methods to interact with the epoch_metrics table of the database

*/

import (
	api "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/migalabs/goteth/pkg/spec"
)

// Postgres intregration variables
var (
	InsertFinalizedQuery = `
	INSERT INTO t_finalized_checkpoint (
		f_id,
		f_block_root,
		f_state_root,
		f_epoch)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT ON CONSTRAINT t_finalized_checkpoint_pkey
		DO 
			UPDATE SET 
			f_block_root = excluded.f_block_root,
			f_state_root = excluded.f_state_root,
			f_epoch = excluded.f_epoch;
	`
)

func InsertCheckpoint(inputCheckpoint CheckpointType) (string, []interface{}) {
	resultArgs := make([]interface{}, 0)

	resultArgs = append(resultArgs, 0) // hardcoded, ID is always 0
	resultArgs = append(resultArgs, inputCheckpoint.Block.String())
	resultArgs = append(resultArgs, inputCheckpoint.State.String())
	resultArgs = append(resultArgs, inputCheckpoint.Epoch)

	return InsertFinalizedQuery, resultArgs
}

type CheckpointType api.FinalizedCheckpointEvent

func (s CheckpointType) Type() spec.ModelType {
	return spec.FinalizedCheckpointModel
}

func ChepointTypeFromCheckpoint(input api.FinalizedCheckpointEvent) CheckpointType {
	return CheckpointType{
		Block: input.Block,
		State: input.State,
		Epoch: input.Epoch,
	}
}
