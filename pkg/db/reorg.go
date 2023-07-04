package db

/*

This file together with the model, has all the needed methods to interact with the epoch_metrics table of the database

*/

import (
	api "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/cortze/eth-cl-state-analyzer/pkg/spec"
)

// Postgres intregration variables
var (
	InsertReorgQuery = `
	INSERT INTO t_reorgs (
		f_slot,
		f_depth,
		f_old_head_block_root,
		f_new_head_block_root,
		f_old_head_state_root,
		f_new_head_state_root
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT ON CONSTRAINT t_reorgs_pkey
		DO NOTHING;
	`
)

func InsertReorg(inputReorg ReorgType) (string, []interface{}) {
	resultArgs := make([]interface{}, 0)

	resultArgs = append(resultArgs, inputReorg.Slot)
	resultArgs = append(resultArgs, inputReorg.Depth)
	resultArgs = append(resultArgs, inputReorg.OldHeadBlock)
	resultArgs = append(resultArgs, inputReorg.NewHeadBlock)
	resultArgs = append(resultArgs, inputReorg.OldHeadState)
	resultArgs = append(resultArgs, inputReorg.NewHeadState)

	return InsertReorgQuery, resultArgs
}

type ReorgType api.ChainReorgEvent

func (s ReorgType) Type() spec.ModelType {
	return spec.ReorgModel
}

func ReorgTypeFromReorg(input api.ChainReorgEvent) ReorgType {
	return ReorgType{
		Slot:         input.Slot,
		Depth:        input.Depth,
		OldHeadBlock: input.OldHeadBlock,
		NewHeadBlock: input.NewHeadBlock,
		OldHeadState: input.OldHeadState,
		NewHeadState: input.NewHeadState,
	}
}
