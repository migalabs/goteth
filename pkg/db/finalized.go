package db

/*

This file together with the model, has all the needed methods to interact with the epoch_metrics table of the database

*/

import (
	"fmt"

	"github.com/ClickHouse/ch-go/proto"
	api "github.com/attestantio/go-eth2-client/api/v1"
)

// Postgres intregration variables
var (
	finalizedTable       = "t_finalized_checkpoint"
	insertFinalizedQuery = `
	INSERT INTO %s (
		f_id,
		f_block_root,
		f_state_root,
		f_epoch)
		VALUES`
)

type InsertFinalized struct {
	checkpoints []api.FinalizedCheckpointEvent
}

func (d InsertFinalized) Table() string {
	return finalizedTable
}

func (d *InsertFinalized) Append(newCheckpoint api.FinalizedCheckpointEvent) {
	d.checkpoints = append(d.checkpoints, newCheckpoint)
}

func (d InsertFinalized) Columns() int {
	return len(d.Input().Columns())
}

func (d InsertFinalized) Rows() int {
	return len(d.checkpoints)
}

func (d InsertFinalized) Query() string {
	return fmt.Sprintf(insertFinalizedQuery, finalizedTable)
}
func (d InsertFinalized) Input() proto.Input {
	// one object per column
	var (
		f_block proto.ColStr
		f_state proto.ColStr
		f_epoch proto.ColUInt64
	)

	for _, checkpoint := range d.checkpoints {
		f_block.Append(checkpoint.Block.String())
		f_state.Append(checkpoint.State.String())
		f_epoch.Append(uint64(checkpoint.Epoch))
	}

	return proto.Input{

		{Name: "f_block", Data: f_block},
		{Name: "f_state", Data: f_state},
		{Name: "f_epoch", Data: f_epoch},
	}
}
