package db

import (
	"github.com/ClickHouse/ch-go/proto"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/migalabs/goteth/pkg/spec"
)

var (
	valLastStatusTable               = "t_validator_last_status"
	insertValidatorLastStatusesQuery = `
	INSERT INTO %s (	
		f_val_idx, 
		f_epoch, 
		f_balance_eth, 
		f_status,
		f_slashed,
		f_activation_epoch,
		f_withdrawal_epoch,
		f_exit_epoch,
		f_public_key)
	VALUES`

	deleteValidatorStatus = `
		DELETE FROM %s
		WHERE f_epoch < $1`
)

func valStatusInput(validatorStatuses []spec.ValidatorLastStatus) proto.Input {
	// one object per column
	var (
		f_epoch            proto.ColUInt64
		f_balance_eth      proto.ColFloat32
		f_status           proto.ColUInt8
		f_slashed          proto.ColBool
		f_activation_epoch proto.ColUInt64
		f_withdrawal_epoch proto.ColUInt64
		f_exit_epoch       proto.ColUInt64
		f_public_key       proto.ColStr
	)

	for _, status := range validatorStatuses {

		f_epoch.Append(uint64(status.Epoch))
		f_balance_eth.Append(status.BalanceToEth())
		f_status.Append(uint8(status.CurrentStatus))
		f_slashed.Append(status.Slashed)
		f_activation_epoch.Append(uint64(status.ActivationEpoch))
		f_withdrawal_epoch.Append(uint64(status.WithdrawalEpoch))
		f_exit_epoch.Append(uint64(status.ExitEpoch))
		f_public_key.Append(status.PublicKey.String())
	}

	return proto.Input{

		{Name: "f_epoch", Data: f_epoch},
		{Name: "f_balance_eth", Data: f_balance_eth},
		{Name: "f_status", Data: f_status},
		{Name: "f_slashed", Data: f_slashed},
		{Name: "f_activation_epoch", Data: f_activation_epoch},
		{Name: "f_withdrawal_epoch", Data: f_withdrawal_epoch},
		{Name: "f_exit_epoch", Data: f_exit_epoch},
		{Name: "f_public_key", Data: f_public_key},
	}
}

func (p *DBService) PersistValLastStatus(data []spec.ValidatorLastStatus) error {
	persistObj := PersistableObject[spec.ValidatorLastStatus]{
		input: valStatusInput,
		table: valLastStatusTable,
		query: insertValidatorLastStatusesQuery,
	}

	for _, item := range data {
		persistObj.Append(item)
	}

	err := p.Persist(persistObj.ExportPersist())
	if err != nil {
		log.Errorf("error persisting validator last status: %s", err.Error())
	}
	return err
}

func (p *DBService) DeleteValLastStatus(epoch phase0.Epoch) error {

	deleteObj := DeletableObject{
		query: deleteValidatorStatus,
		table: valLastStatusTable,
		args:  []any{epoch},
	}

	err := p.Delete(deleteObj)
	if err != nil {
		log.Errorf("error deleting validator last status: %s", err.Error())
	}

	return err
}
