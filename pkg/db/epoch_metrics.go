package db

import (
	"fmt"

	"github.com/ClickHouse/ch-go/proto"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/migalabs/goteth/pkg/spec"
)

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
		f_source_att_effective_balance_eth,
		f_target_att_effective_balance_eth,
		f_head_att_effective_balance_eth,
		f_total_effective_balance_eth,
		f_missing_source,
		f_missing_target,
		f_missing_head,
		f_timestamp,
		f_num_slashed_vals,
		f_num_active_vals,
		f_num_exited_vals,
		f_num_in_activation_vals,
		f_sync_committee_participation		
		)
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

func epochsInput(epochs []spec.Epoch) proto.Input {
	// one object per column
	var (
		f_epoch                            proto.ColUInt64
		f_slot                             proto.ColUInt64
		f_num_att                          proto.ColUInt64
		f_num_att_vals                     proto.ColUInt64
		f_num_vals                         proto.ColUInt64
		f_total_balance_eth                proto.ColFloat32
		f_att_effective_balance_eth        proto.ColUInt64
		f_source_att_effective_balance_eth proto.ColUInt64
		f_target_att_effective_balance_eth proto.ColUInt64
		f_head_att_effective_balance_eth   proto.ColUInt64
		f_total_effective_balance_eth      proto.ColUInt64
		f_missing_source                   proto.ColUInt64
		f_missing_target                   proto.ColUInt64
		f_missing_head                     proto.ColUInt64
		f_timestamp                        proto.ColUInt64
		f_num_slashed_vals                 proto.ColUInt64
		f_num_active_vals                  proto.ColUInt64
		f_num_exited_vals                  proto.ColUInt64
		f_num_in_activation_vals           proto.ColUInt64
		f_sync_committee_participation     proto.ColUInt64
	)

	for _, epoch := range epochs {
		f_epoch.Append(uint64(epoch.Epoch))
		f_slot.Append(uint64(epoch.Slot))
		f_num_att.Append(uint64(epoch.NumAttestations))
		f_num_att_vals.Append(uint64(epoch.NumAttValidators))
		f_num_vals.Append(uint64(epoch.NumValidators))
		f_total_balance_eth.Append(float32(epoch.TotalBalance))
		f_att_effective_balance_eth.Append(uint64(epoch.AttEffectiveBalance))
		f_source_att_effective_balance_eth.Append(uint64(epoch.SourceAttEffectiveBalance))
		f_target_att_effective_balance_eth.Append(uint64(epoch.TargetAttEffectiveBalance))
		f_head_att_effective_balance_eth.Append(uint64(epoch.HeadAttEffectiveBalance))
		f_total_effective_balance_eth.Append(uint64(epoch.TotalEffectiveBalance))
		f_missing_source.Append(uint64(epoch.MissingSource))
		f_missing_target.Append(uint64(epoch.MissingTarget))
		f_missing_head.Append(uint64(epoch.MissingHead))
		f_timestamp.Append(uint64(epoch.Timestamp))
		f_num_slashed_vals.Append(uint64(epoch.NumSlashedVals))
		f_num_active_vals.Append(uint64(epoch.NumActiveVals))
		f_num_exited_vals.Append(uint64(epoch.NumExitedVals))
		f_num_in_activation_vals.Append(uint64(epoch.NumInActivationVals))
		f_sync_committee_participation.Append(epoch.SyncCommitteeParticipation)
	}

	return proto.Input{
		{Name: "f_epoch", Data: f_epoch},
		{Name: "f_slot", Data: f_slot},
		{Name: "f_num_att", Data: f_num_att},
		{Name: "f_num_att_vals", Data: f_num_att_vals},
		{Name: "f_num_vals", Data: f_num_vals},
		{Name: "f_total_balance_eth", Data: f_total_balance_eth},
		{Name: "f_att_effective_balance_eth", Data: f_att_effective_balance_eth},
		{Name: "f_source_att_effective_balance_eth", Data: f_source_att_effective_balance_eth},
		{Name: "f_target_att_effective_balance_eth", Data: f_target_att_effective_balance_eth},
		{Name: "f_head_att_effective_balance_eth", Data: f_head_att_effective_balance_eth},
		{Name: "f_total_effective_balance_eth", Data: f_total_effective_balance_eth},
		{Name: "f_missing_source", Data: f_missing_source},
		{Name: "f_missing_target", Data: f_missing_target},
		{Name: "f_missing_head", Data: f_missing_head},
		{Name: "f_timestamp", Data: f_timestamp},
		{Name: "f_num_slashed_vals", Data: f_num_slashed_vals},
		{Name: "f_num_active_vals", Data: f_num_active_vals},
		{Name: "f_num_exited_vals", Data: f_num_exited_vals},
		{Name: "f_num_in_activation_vals", Data: f_num_in_activation_vals},
		{Name: "f_sync_committee_participation", Data: f_sync_committee_participation},
	}
}

func (p *DBService) PersistEpochs(data []spec.Epoch) error {
	persistObj := PersistableObject[spec.Epoch]{
		input: epochsInput,
		table: epochsTable,
		query: insertEpochQuery,
	}

	for _, item := range data {
		persistObj.Append(item)
	}

	err := p.Persist(persistObj.ExportPersist())
	if err != nil {
		log.Errorf("error persisting epoch: %s", err.Error())
	}
	return err
}

func (p *DBService) RetrieveLastEpoch() (phase0.Epoch, error) {

	var dest []struct {
		F_epoch uint64 `ch:"f_epoch"`
	}

	err := p.highSelect(
		fmt.Sprintf(selectLastEpochQuery, epochsTable),
		&dest)

	if len(dest) > 0 {
		return phase0.Epoch(dest[0].F_epoch), err
	}
	return 0, err

}

// delete metrics that use the state at epoch x
func (s *DBService) DeleteStateMetrics(epoch phase0.Epoch) error {
	var err error

	// epochs are written at currentState using current state and nextState
	err = s.Delete(DeletableObject{
		query: deleteEpochsQuery,
		table: epochsTable,
		args:  []any{epoch - 1},
	}) // when deleteState -> nextState

	if err != nil {
		return err
	}
	err = s.Delete(DeletableObject{
		query: deleteEpochsQuery,
		table: epochsTable,
		args:  []any{epoch},
	}) // when deleteState -> currentState
	if err != nil {
		return err
	}

	// proposer duties are writter using nextState
	err = s.Delete(DeletableObject{
		query: deleteProposerDutiesQuery,
		table: proposerDutiesTable,
		args:  []any{epoch},
	})
	if err != nil {
		return err
	}

	// valRewards are written at nextState using prevState, currentState and nextState
	err = s.Delete(DeletableObject{
		query: deleteValidatorRewardsInEpochQuery,
		table: valRewardsTable,
		args:  []any{epoch + 2},
	}) // when deleteState -> prevState
	if err != nil {
		return err
	}
	err = s.Delete(DeletableObject{
		query: deleteValidatorRewardsInEpochQuery,
		table: valRewardsTable,
		args:  []any{epoch + 1},
	}) // when deleteState -> currentState
	if err != nil {
		return err
	}
	err = s.Delete(DeletableObject{
		query: deleteValidatorRewardsInEpochQuery,
		table: valRewardsTable,
		args:  []any{epoch},
	}) // when deleteState -> nextState
	if err != nil {
		return err
	}
	return nil

}
