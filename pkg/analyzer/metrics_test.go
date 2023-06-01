package analyzer

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/cortze/eth-cl-state-analyzer/pkg/clientapi"
	"github.com/cortze/eth-cl-state-analyzer/pkg/spec"
	local_spec "github.com/cortze/eth-cl-state-analyzer/pkg/spec"
	"github.com/cortze/eth-cl-state-analyzer/pkg/spec/metrics"
	"github.com/magiconair/properties/assert"
)

func BuildStateAnalyzer() (StateAnalyzer, error) {

	ctx := context.Background()

	// generate the httpAPI client
	cli, err := clientapi.NewAPIClient(ctx, "http://localhost:5052", 50*time.Second)
	if err != nil {
		return StateAnalyzer{}, err
	}

	return StateAnalyzer{
		ctx: context.Background(),
		cli: cli,
	}, nil
}

func BuildBlockAnalyzer() (BlockAnalyzer, error) {

	ctx := context.Background()

	// generate the httpAPI client
	cli, err := clientapi.NewAPIClient(ctx, "http://localhost:5052", 50*time.Second)
	if err != nil {
		return BlockAnalyzer{}, err
	}

	return BlockAnalyzer{
		ctx: context.Background(),
		cli: cli,
	}, nil
}

func BuildEpochTask(analyzer StateAnalyzer, slot phase0.Slot) (EpochTask, error) {

	// Review slot is well positioned

	epoch := slot / spec.SlotsPerEpoch

	slot = ((epoch + 1) * spec.SlotsPerEpoch) - 1

	fmt.Printf("downloading state at slot: %d\n", slot-spec.SlotsPerEpoch)
	newState, err := analyzer.RequestBeaconState(slot - spec.SlotsPerEpoch)
	if err != nil {
		return EpochTask{}, fmt.Errorf("could not download state: %s", err)

	}
	prevState, err := local_spec.GetCustomState(*newState, analyzer.cli.Api)
	if err != nil {
		return EpochTask{}, fmt.Errorf("could not parse state: %s", err)
	}

	fmt.Printf("downloading state at slot: %d\n", slot)
	newState, err = analyzer.RequestBeaconState(slot)
	if err != nil {
		return EpochTask{}, fmt.Errorf("could not download state: %s", err)
	}
	currentState, err := local_spec.GetCustomState(*newState, analyzer.cli.Api)
	if err != nil {
		return EpochTask{}, fmt.Errorf("could not parse state: %s", err)
	}

	fmt.Printf("downloading state at slot: %d\n", slot+spec.SlotsPerEpoch)
	newState, err = analyzer.RequestBeaconState(slot + spec.SlotsPerEpoch)
	if err != nil {
		return EpochTask{}, fmt.Errorf("could not download state: %s", err)
	}
	nextState, err := local_spec.GetCustomState(*newState, analyzer.cli.Api)
	if err != nil {
		return EpochTask{}, fmt.Errorf("could not parse state: %s", err)
	}

	return EpochTask{
		PrevState: prevState,
		State:     currentState,
		NextState: nextState,
	}, nil

}

func TestPhase0Epoch(t *testing.T) {

	analyzer, err := BuildStateAnalyzer()
	if err != nil {
		fmt.Errorf("could not build analyzer: %s", err)
		return
	}

	epochTask, err := BuildEpochTask(analyzer, 320031) // epoch 10000
	if err != nil {
		fmt.Errorf("could not build epoch task: %s", err)
		return
	}

	// returns the state in a custom struct for Phase0, Altair of Bellatrix
	stateMetrics, err := metrics.StateMetricsByForkVersion(epochTask.NextState, epochTask.State, epochTask.PrevState, analyzer.cli.Api)

	assert.Equal(t, stateMetrics.GetMetricsBase().CurrentState.NumActiveVals, uint(60849))
	assert.Equal(t,
		stateMetrics.GetMetricsBase().CurrentState.MissedBlocks,
		[]phase0.Slot{320011, 320023})

}

func TestAltairEpoch(t *testing.T) {

	analyzer, err := BuildStateAnalyzer()
	if err != nil {
		fmt.Errorf("could not build analyzer: %s", err)
		return
	}

	epochTask, err := BuildEpochTask(analyzer, 2375711) // epoch 74240
	if err != nil {
		fmt.Errorf("could not build epoch task: %s", err)
		return
	}

	// returns the state in a custom struct for Phase0, Altair of Bellatrix
	stateMetrics, err := metrics.StateMetricsByForkVersion(epochTask.NextState, epochTask.State, epochTask.PrevState, analyzer.cli.Api)

	assert.Equal(t, stateMetrics.GetMetricsBase().CurrentState.NumActiveVals, uint(250226))
	assert.Equal(t,
		stateMetrics.GetMetricsBase().CurrentState.MissedBlocks,
		[]phase0.Slot{2375681, 2375682, 2375683, 2375688, 2375692, 2375699, 2375704})
	assert.Equal(t,
		stateMetrics.GetMetricsBase().CurrentState.AttestingBalance[1],
		phase0.Gwei(7979389000000000))
	assert.Equal(t,
		stateMetrics.GetMetricsBase().CurrentState.TotalActiveBalance,
		phase0.Gwei(8007160000000000))
}

func TestAltairRewards(t *testing.T) {

	analyzer, err := BuildStateAnalyzer()
	if err != nil {
		fmt.Errorf("could not build analyzer: %s", err)
		return
	}

	epochTask, err := BuildEpochTask(analyzer, 2375711) // epoch 74240
	if err != nil {
		fmt.Errorf("could not build epoch task: %s", err)
		return
	}

	// returns the state in a custom struct for Phase0, Altair of Bellatrix
	stateMetrics, err := metrics.StateMetricsByForkVersion(epochTask.NextState, epochTask.State, epochTask.PrevState, analyzer.cli.Api)

	// Test when everything runs normal
	rewards, err := stateMetrics.GetMaxReward(1250)
	assert.Equal(t,
		rewards.Reward,
		int64(19222))

	assert.Equal(t,
		rewards.AttestationReward,
		phase0.Gwei(19222))

	assert.Equal(t,
		rewards.AttSlot,
		phase0.Slot(2375659))
	assert.Equal(t,
		rewards.BaseReward,
		phase0.Gwei(22880))
	assert.Equal(t,
		rewards.MaxReward,
		phase0.Gwei(19222))

	assert.Equal(t,
		rewards.SyncCommitteeReward,
		phase0.Gwei(0))
	assert.Equal(t,
		rewards.InSyncCommittee,
		false)
	assert.Equal(t,
		rewards.MissingHead,
		false)
	assert.Equal(t,
		rewards.MissingSource,
		false)
	assert.Equal(t,
		rewards.MissingTarget,
		false)
	assert.Equal(t,
		rewards.ValidatorBalance,
		phase0.Gwei(34446677741))

	// Test when validator did not perform duties and was in a sync committee
	rewards, err = stateMetrics.GetMaxReward(60027)
	assert.Equal(t,
		rewards.Reward,
		int64(-254518))

	assert.Equal(t,
		rewards.AttestationReward,
		phase0.Gwei(17674))

	assert.Equal(t,
		rewards.AttSlot,
		phase0.Slot(2375772))
	assert.Equal(t,
		rewards.BaseReward,
		phase0.Gwei(22880))
	assert.Equal(t,
		rewards.MaxReward,
		phase0.Gwei(257892))

	assert.Equal(t,
		rewards.SyncCommitteeReward,
		phase0.Gwei(240218))
	assert.Equal(t,
		rewards.InSyncCommittee,
		true)
	assert.Equal(t,
		rewards.MissingHead,
		true)
	assert.Equal(t,
		rewards.MissingSource,
		true)
	assert.Equal(t,
		rewards.MissingTarget,
		true)
	assert.Equal(t,
		rewards.ValidatorBalance,
		phase0.Gwei(33675494489))
}
