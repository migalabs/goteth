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

	rewards, err := stateMetrics.GetMaxReward(1250)
	assert.Equal(t,
		int64(19222),
		rewards.Reward)

	assert.Equal(t,
		phase0.Gwei(19222),
		rewards.AttestationReward)

	assert.Equal(t,
		phase0.Slot(2375659),
		rewards.AttSlot)
	assert.Equal(t,
		phase0.Gwei(22880),
		rewards.BaseReward)
	assert.Equal(t,
		phase0.Gwei(19222),
		rewards.MaxReward)

	assert.Equal(t,
		phase0.Gwei(0),
		rewards.SyncCommitteeReward)
	assert.Equal(t,
		false,
		rewards.InSyncCommittee)
	assert.Equal(t,
		false,
		rewards.MissingHead)
	assert.Equal(t,
		false,
		rewards.MissingSource)
	assert.Equal(t,
		false,
		rewards.MissingTarget)
	assert.Equal(t,
		phase0.Gwei(34446677741),
		rewards.ValidatorBalance)
}
