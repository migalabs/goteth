package db

import (
	"fmt"

	"github.com/ClickHouse/ch-go/proto"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/migalabs/goteth/pkg/spec"
)

// Postgres intregration variables
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

	DeleteValidatorStatus = `
		DELETE FROM %s
		WHERE f_epoch < $1`
)

type InsertValidatorLastStatuses struct {
	validatorStatuses []spec.ValidatorLastStatus
}

func (d InsertValidatorLastStatuses) Table() string {
	return valLastStatusTable
}
func (d *InsertValidatorLastStatuses) Append(newStatus spec.ValidatorLastStatus) {
	d.validatorStatuses = append(d.validatorStatuses, newStatus)
}

func (d InsertValidatorLastStatuses) Columns() int {
	return len(d.Input().Columns())
}

func (d InsertValidatorLastStatuses) Rows() int {
	return len(d.validatorStatuses)
}

func (d InsertValidatorLastStatuses) Query() string {
	return fmt.Sprintf(insertValidatorLastStatusesQuery, valLastStatusTable)
}
func (d InsertValidatorLastStatuses) Input() proto.Input {
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

	for _, status := range d.validatorStatuses {

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

type DeleteValLastStatus struct {
	Epoch phase0.Epoch
}

func (d DeleteValLastStatus) Query() string {
	return fmt.Sprintf(DeleteValidatorStatus, valLastStatusTable)
}

func (d DeleteValLastStatus) Table() string {
	return valLastStatusTable
}

func (d DeleteValLastStatus) Args() []any {
	return []any{d.Epoch}
}
