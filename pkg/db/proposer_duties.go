package db

import (
	"fmt"

	"github.com/ClickHouse/ch-go/proto"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/migalabs/goteth/pkg/spec"
)

/*

This file together with the model, has all the needed methods to interact with the epoch_metrics table of the database

*/

// // Postgres intregration variables
var (
	proposerDutiesTable       = "t_proposer_duties"
	insertProposerDutiesQuery = `
	INSERT INTO %s (
		f_val_idx,
		f_proposer_slot,
		f_proposed)
		VALUES
	`
	// if there is a confilct the line already exists

	deleteProposerDutiesQuery = `
	DELETE FROM %s
	WHERE f_proposer_slot/32 = $1;
`
)

type InsertProposerDuties struct {
	duties []spec.ProposerDuty
}

func (d InsertProposerDuties) Table() string {
	return proposerDutiesTable
}

func (d *InsertProposerDuties) Append(newDuty spec.ProposerDuty) {
	d.duties = append(d.duties, newDuty)
}

func (d InsertProposerDuties) Columns() int {
	return len(d.Input().Columns())
}

func (d InsertProposerDuties) Rows() int {
	return len(d.duties)
}

func (d InsertProposerDuties) Query() string {
	return fmt.Sprintf(insertProposerDutiesQuery, proposerDutiesTable)
}
func (d InsertProposerDuties) Input() proto.Input {
	// one object per column
	var (
		f_val_idx       proto.ColUInt64
		f_proposer_slot proto.ColUInt64
		f_proposed      proto.ColBool
	)

	for _, duty := range d.duties {
		f_val_idx.Append(uint64(duty.ValIdx))
		f_proposer_slot.Append(uint64(duty.ProposerSlot))
		f_proposed.Append(duty.Proposed)
	}

	return proto.Input{

		{Name: "f_val_idx", Data: f_val_idx},
		{Name: "f_proposer_slot", Data: f_proposer_slot},
		{Name: "f_proposed", Data: f_proposed},
	}
}

type DeleteProposerDuties struct {
	epoch phase0.Epoch
}

func (d DeleteProposerDuties) Query() string {
	return fmt.Sprintf(deleteProposerDutiesQuery, proposerDutiesTable)
}

func (d DeleteProposerDuties) Table() string {
	return proposerDutiesTable
}

func (d DeleteProposerDuties) Args() []any {
	return []any{d.epoch}
}
