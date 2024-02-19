package db

import (
	"github.com/ClickHouse/ch-go/proto"
	api "github.com/attestantio/go-eth2-client/api/v1"
)

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

func reorgsInput(reorgs []api.ChainReorgEvent) proto.Input {
	// one object per column
	var (
		f_slot                proto.ColUInt64
		f_depth               proto.ColUInt64
		f_old_head_block_root proto.ColStr
		f_new_head_block_root proto.ColStr
		f_old_head_state_root proto.ColStr
		f_new_head_state_root proto.ColStr
	)

	for _, reorg := range reorgs {

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

func (p *DBService) PersistReorgs(data []api.ChainReorgEvent) error {
	persistObj := PersistableObject[api.ChainReorgEvent]{
		input: reorgsInput,
		table: reorgsTable,
		query: insertReorgsQuery,
	}

	for _, item := range data {
		persistObj.Append(item)
	}

	err := p.Persist(persistObj.ExportPersist())
	if err != nil {
		log.Errorf("error persisting reorgs: %s", err.Error())
	}
	return err
}
