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
	reorgsTable       = "t_reorgs"
	insertReorgsQuery = `
	INSERT INTO %s (
		f_slot,
		f_depth,
		f_old_head_block_root,
		f_new_head_block_root,
		f_old_head_state_root,
		f_new_head_state_root)
		VALUES`
)

type InsertReorgs struct {
	reorgs []api.ChainReorgEvent
}

func (d InsertReorgs) Table() string {
	return reorgsTable
}

func (d *InsertReorgs) Append(newReorg api.ChainReorgEvent) {
	d.reorgs = append(d.reorgs, newReorg)
}

func (d InsertReorgs) Columns() int {
	return len(d.Input().Columns())
}

func (d InsertReorgs) Rows() int {
	return len(d.reorgs)
}

func (d InsertReorgs) Query() string {
	return fmt.Sprintf(insertReorgsQuery, reorgsTable)
}
func (d InsertReorgs) Input() proto.Input {
	// one object per column
	var (
		f_slot                proto.ColUInt64
		f_depth               proto.ColUInt64
		f_old_head_block_root proto.ColStr
		f_new_head_block_root proto.ColStr
		f_old_head_state_root proto.ColStr
		f_new_head_state_root proto.ColStr
	)

	for _, reorg := range d.reorgs {

		f_slot.Append(uint64(reorg.Slot))
		f_depth.Append(reorg.Depth)
		f_old_head_block_root.Append(reorg.OldHeadBlock.String())
		f_new_head_block_root.Append(reorg.NewHeadBlock.String())
		f_old_head_state_root.Append(reorg.OldHeadState.String())
		f_new_head_state_root.Append(reorg.NewHeadState.String())
	}

	return proto.Input{

		{Name: "f_slot", Data: f_slot},
		{Name: "f_depth", Data: f_depth},
		{Name: "f_old_head_block_root", Data: f_old_head_block_root},
		{Name: "f_new_head_block_root", Data: f_new_head_block_root},
		{Name: "f_old_head_state_root", Data: f_old_head_state_root},
		{Name: "f_new_head_state_root", Data: f_new_head_state_root},
	}
}
