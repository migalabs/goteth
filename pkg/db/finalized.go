package db

import (
	"github.com/ClickHouse/ch-go/proto"
	api "github.com/attestantio/go-eth2-client/api/v1"
)

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

func finalizedInput(checkpoints []api.FinalizedCheckpointEvent) proto.Input {
	// one object per column
	var (
		f_block_root proto.ColStr
		f_state_root proto.ColStr
		f_epoch      proto.ColUInt64
	)

	for _, checkpoint := range checkpoints {
		f_block_root.Append(checkpoint.Block.String())
		f_state_root.Append(checkpoint.State.String())
		f_epoch.Append(uint64(checkpoint.Epoch))
	}

	return proto.Input{

		{Name: "f_block_root", Data: f_block_root},
		{Name: "f_state_root", Data: f_state_root},
		{Name: "f_epoch", Data: f_epoch},
	}
}

func (p *DBService) PersistFinalized(data []api.FinalizedCheckpointEvent) error {
	persistObj := PersistableObject[api.FinalizedCheckpointEvent]{
		input: finalizedInput,
		table: finalizedTable,
		query: insertFinalizedQuery,
	}

	for _, item := range data {
		persistObj.Append(item)
	}

	err := p.Persist(persistObj.ExportPersist())
	if err != nil {
		log.Errorf("error persisting checkpoint: %s", err.Error())
	}
	return err
}
