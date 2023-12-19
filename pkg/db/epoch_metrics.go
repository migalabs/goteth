package db

import (
	"fmt"
	"time"

	"github.com/ClickHouse/ch-go/proto"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/migalabs/goteth/pkg/spec"
)

/*

This file together with the model, has all the needed methods to interact with the epoch_metrics table of the database

*/

// Postgres intregration variables
var (
	epochsTable      = "t_epoch_metrics_summary"
	insertEpochQuery = `
	INSERT INTO %s (
		f_epoch,
		f_slot,
		f_num_att,
		f_num_att_vals,
		f_num_vals,
		f_total_balance_eth,
		f_att_effective_balance_eth,
		f_total_effective_balance_eth,
		f_missing_source,
		f_missing_target,
		f_missing_head,
		f_timestamp,
		f_num_slashed_vals,
		f_num_active_vals,
		f_num_exited_vals,
		f_num_in_activation_vals)
		VALUES`

	selectLastEpochQuery = `
		SELECT f_epoch
		FROM %s
		ORDER BY f_epoch DESC
		LIMIT 1`

	deleteEpochsQuery = `
		DELETE FROM %s
		WHERE f_epoch = $1;
`
)

type InsertEpochs struct {
	epochs []spec.Epoch
}

func (d InsertEpochs) Table() string {
	return epochsTable
}

func (d *InsertEpochs) Append(newEpoch spec.Epoch) {
	d.epochs = append(d.epochs, newEpoch)
}

func (d InsertEpochs) Columns() int {
	return len(d.Input().Columns())
}

func (d InsertEpochs) Rows() int {
	return len(d.epochs)
}

func (d InsertEpochs) Query() string {
	return fmt.Sprintf(insertEpochQuery, epochsTable)
}
func (d InsertEpochs) Input() proto.Input {
	// one object per column
	var (
		f_timestamp                   proto.ColUInt64
		f_epoch                       proto.ColUInt64
		f_slot                        proto.ColUInt64
		f_num_att                     proto.ColUInt64
		f_num_att_vals                proto.ColUInt64
		f_num_vals                    proto.ColUInt64
		f_total_balance_eth           proto.ColFloat32
		f_att_effective_balance_eth   proto.ColFloat32
		f_total_effective_balance_eth proto.ColFloat32
		f_missing_source              proto.ColUInt64
		f_missing_target              proto.ColUInt64
		f_missing_head                proto.ColUInt64
		f_num_slashed_vals            proto.ColUInt64
		f_num_active_vals             proto.ColUInt64
		f_num_exited_vals             proto.ColUInt64
		f_num_in_activation_vals      proto.ColUInt64
	)

	for _, epoch := range d.epochs {
		f_timestamp.Append(uint64(epoch.Timestamp))
		f_epoch.Append(uint64(epoch.Epoch))
		f_slot.Append(uint64(epoch.Slot))
		f_num_att.Append(uint64(epoch.NumAttestations))
		f_num_att_vals.Append(uint64(epoch.NumAttValidators))
		f_num_vals.Append(uint64(epoch.NumValidators))
		f_total_balance_eth.Append(float32(epoch.TotalBalance))
		f_att_effective_balance_eth.Append(float32(epoch.AttEffectiveBalance))
		f_total_effective_balance_eth.Append(float32(epoch.TotalEffectiveBalance))
		f_missing_source.Append(uint64(epoch.MissingSource))
		f_missing_target.Append(uint64(epoch.MissingTarget))
		f_missing_head.Append(uint64(epoch.MissingHead))
		f_num_slashed_vals.Append(uint64(epoch.NumSlashedVals))
		f_num_active_vals.Append(uint64(epoch.NumActiveVals))
		f_num_exited_vals.Append(uint64(epoch.NumExitedVals))
		f_num_in_activation_vals.Append(uint64(epoch.NumInActivationVals))

	}

	return proto.Input{

		{Name: "f_timestamp", Data: f_timestamp},
		{Name: "f_epoch", Data: f_epoch},
		{Name: "f_slot", Data: f_slot},
		{Name: "f_num_att", Data: f_num_att},
		{Name: "f_num_att_vals", Data: f_num_att_vals},
		{Name: "f_num_vals", Data: f_num_vals},
		{Name: "f_total_balance_eth", Data: f_total_balance_eth},
		{Name: "f_att_effective_balance_eth", Data: f_att_effective_balance_eth},
		{Name: "f_total_effective_balance_eth", Data: f_total_effective_balance_eth},
		{Name: "f_missing_source", Data: f_missing_source},
		{Name: "f_missing_target", Data: f_missing_target},
		{Name: "f_missing_head", Data: f_missing_head},
		{Name: "f_num_slashed_vals", Data: f_num_slashed_vals},
		{Name: "f_num_active_vals", Data: f_num_active_vals},
		{Name: "f_num_exited_vals", Data: f_num_exited_vals},
		{Name: "f_num_in_activation_vals", Data: f_num_in_activation_vals},
	}
}

func (p *DBService) RetrieveLastEpoch() (phase0.Epoch, error) {

	var result phase0.Epoch
	query := fmt.Sprintf(selectLastEpochQuery, epochsTable)
	var err error
	var dest []struct {
		F_epoch uint64 `ch:"f_epoch"`
	}
	startTime := time.Now()

	p.highMu.Lock()
	err = p.highLevelClient.Select(p.ctx, &dest, query)
	p.highMu.Unlock()

	if err == nil && len(dest) > 0 {
		log.Infof("retrieved %d rows in %f seconds, query: %s", len(dest), time.Since(startTime).Seconds(), query)
		result = phase0.Epoch(dest[0].F_epoch)
	}

	return result, err
}

type DeleteEpoch struct {
	epoch phase0.Epoch
}

func (d DeleteEpoch) Query() string {
	return fmt.Sprintf(deleteEpochsQuery, epochsTable)
}

func (d DeleteEpoch) Table() string {
	return epochsTable
}

func (d DeleteEpoch) Args() []any {
	return []any{d.epoch}
}

// delete metrics that use the state at epoch x
func (s *DBService) DeleteStateMetrics(epoch phase0.Epoch) error {
	var err error

	// epochs are written at currentState using current state and nextState
	s.Delete(DeleteEpoch{epoch: epoch - 1}) // when deleteState -> nextState
	if err != nil {
		return err
	}
	s.Delete(DeleteEpoch{epoch: epoch}) // when deleteState -> currentState
	if err != nil {
		return err
	}

	// proposer duties are writter using nextState
	s.Delete(DeleteProposerDuties{epoch: epoch})
	if err != nil {
		return err
	}

	// valRewards are written at nextState using prevState, currentState and nextState
	s.Delete(DeleteValRewards{epoch: epoch + 2}) // when deleteState -> prevState
	if err != nil {
		return err
	}
	s.Delete(DeleteValRewards{epoch: epoch + 1}) // when deleteState -> currentState
	if err != nil {
		return err
	}
	s.Delete(DeleteValRewards{epoch: epoch}) // when deleteState -> nextState
	if err != nil {
		return err
	}
	return nil

}
