package db

import (
	"github.com/ClickHouse/ch-go/proto"
	"github.com/migalabs/goteth/pkg/spec"
)

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

func proposerDutiesInput(duties []spec.ProposerDuty) proto.Input {
	// one object per column
	var (
		f_val_idx       proto.ColUInt64
		f_proposer_slot proto.ColUInt64
		f_proposed      proto.ColBool
	)

	for _, duty := range duties {
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

func (p *DBService) PersistDuties(data []spec.ProposerDuty) error {
	persistObj := PersistableObject[spec.ProposerDuty]{
		input: proposerDutiesInput,
		table: proposerDutiesTable,
		query: insertProposerDutiesQuery,
	}

	for _, item := range data {
		persistObj.Append(item)
	}

	err := p.Persist(persistObj.ExportPersist())
	if err != nil {
		log.Errorf("error persisting proposer duties: %s", err.Error())
	}
	return err
}
