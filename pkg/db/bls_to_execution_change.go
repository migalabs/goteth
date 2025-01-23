package db

import (
	"github.com/ClickHouse/ch-go/proto"
	"github.com/migalabs/goteth/pkg/spec"
)

var (
	blsToExecutionChangeTable       = "t_bls_to_execution_changes"
	insertBLSToExecutionChangeQuery = `
	INSERT INTO %s (
		f_slot,
		f_epoch,
		f_validator_index,
		f_from_bls_pubkey,
		f_to_execution_address
		)
		VALUES`

	deleteBLSToExecutionChangesQuery = `
		DELETE FROM %s
		WHERE f_slot = $1;`
)

func blsToExecutionChangeInput(blsToExecutionChanges []spec.BLSToExecutionChange) proto.Input {
	// one object per column
	var (
		f_slot                 proto.ColUInt64
		f_epoch                proto.ColUInt64
		f_validator_index      proto.ColUInt64
		f_from_bls_pubkey      proto.ColStr
		f_to_execution_address proto.ColStr
	)

	for _, withdrawal := range blsToExecutionChanges {

		f_slot.Append(uint64(withdrawal.Slot))
		f_epoch.Append(uint64(withdrawal.Epoch))
		f_validator_index.Append(uint64(withdrawal.ValidatorIndex))
		f_from_bls_pubkey.Append(withdrawal.FromBLSPublicKey.String())
		f_to_execution_address.Append(withdrawal.ToExecutionAddress.String())
	}

	return proto.Input{
		{Name: "f_slot", Data: f_slot},
		{Name: "f_epoch", Data: f_epoch},
		{Name: "f_validator_index", Data: f_validator_index},
		{Name: "f_from_bls_pubkey", Data: f_from_bls_pubkey},
		{Name: "f_to_execution_address", Data: f_to_execution_address},
	}
}

func (p *DBService) PersistBLSToExecutionChanges(data []spec.BLSToExecutionChange) error {
	persistObj := PersistableObject[spec.BLSToExecutionChange]{
		input: blsToExecutionChangeInput,
		table: blsToExecutionChangeTable,
		query: insertBLSToExecutionChangeQuery,
	}

	for _, item := range data {
		persistObj.Append(item)
	}

	err := p.Persist(persistObj.ExportPersist())
	if err != nil {
		log.Errorf("error persisting blsToExecutionChange: %s", err.Error())
	}
	return err
}
