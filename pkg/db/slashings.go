package db

import (
	"github.com/ClickHouse/ch-go/proto"
	"github.com/migalabs/goteth/pkg/spec"
)

var (
	slashingsTable       = "t_slashings"
	insertSlashingsQuery = `
	INSERT INTO %s (
		f_slashed_validator_index,
		f_slashed_by_validator_index,
		f_slashing_reason,
		f_slot,
		f_epoch
		)
		VALUES`
)

func slashingsInput(slashings []spec.AgnosticSlashing) proto.Input {
	// one object per column
	var (
		f_slashed_validator_index    proto.ColUInt64
		f_slashed_by_validator_index proto.ColUInt64
		f_slashing_reason            proto.ColStr
		f_slot                       proto.ColUInt64
		f_epoch                      proto.ColUInt64
	)

	for _, slashing := range slashings {
		f_slashed_validator_index.Append(uint64(slashing.SlashedValidator))
		f_slashed_by_validator_index.Append(uint64(slashing.SlashedBy))
		f_slashing_reason.Append(string(slashing.SlashingReason))
		f_slot.Append(uint64(slashing.Slot))
		f_epoch.Append(uint64(slashing.Epoch))
	}

	return proto.Input{
		{Name: "f_slashed_validator_index", Data: f_slashed_validator_index},
		{Name: "f_slashed_by_validator_index", Data: f_slashed_by_validator_index},
		{Name: "f_slashing_reason", Data: f_slashing_reason},
		{Name: "f_slot", Data: f_slot},
		{Name: "f_epoch", Data: f_epoch},
	}
}

func (p *DBService) PersistSlashings(data []spec.AgnosticSlashing) error {
	persistObj := PersistableObject[spec.AgnosticSlashing]{
		input: slashingsInput,
		table: slashingsTable,
		query: insertSlashingsQuery,
	}

	for _, item := range data {
		persistObj.Append(item)
	}

	err := p.Persist(persistObj.ExportPersist())
	if err != nil {
		log.Errorf("error persisting block rewards: %s", err.Error())
	}
	return err
}
