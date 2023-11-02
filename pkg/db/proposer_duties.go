package db

/*

This file together with the model, has all the needed methods to interact with the epoch_metrics table of the database

*/

import (
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/migalabs/goteth/pkg/spec"
)

// Postgres intregration variables
var (
	InsertProposerDuty = `
	INSERT INTO t_proposer_duties (
		f_val_idx, 
		f_proposer_slot,
		f_proposed)
		VALUES ($1, $2, $3)
		ON CONFLICT DO NOTHING;
	`
	// if there is a confilct the line already exists

	DropProposerDutiesQuery = `
	DELETE FROM t_proposer_duties
	WHERE f_proposer_slot/32 = $1;
`
)

func insertProposerDuty(inputDuty spec.ProposerDuty) (string, []interface{}) {
	resultArgs := make([]interface{}, 0)
	resultArgs = append(resultArgs, inputDuty.ValIdx)
	resultArgs = append(resultArgs, inputDuty.ProposerSlot)
	resultArgs = append(resultArgs, inputDuty.Proposed)

	return InsertProposerDuty, resultArgs
}

func ProposerDutyOperation(inputDuty spec.ProposerDuty) (string, []interface{}) {

	q, args := insertProposerDuty(inputDuty)
	return q, args

}

type ProposerDutiesDropType phase0.Epoch

func (s ProposerDutiesDropType) Type() spec.ModelType {
	return spec.ProposerDutyDropModel
}

func DropProposerDuties(epoch ProposerDutiesDropType) (string, []interface{}) {
	resultArgs := make([]interface{}, 0)
	resultArgs = append(resultArgs, epoch)
	return DropProposerDutiesQuery, resultArgs
}
