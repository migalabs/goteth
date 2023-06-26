package analyzer

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/cortze/eth-cl-state-analyzer/pkg/clientapi"
	"github.com/cortze/eth-cl-state-analyzer/pkg/spec"
	"github.com/cortze/eth-cl-state-analyzer/pkg/spec/metrics"
	"github.com/stretchr/testify/assert"
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

func BuildBlockAnalyzerWithEL() (BlockAnalyzer, error) {

	ctx := context.Background()

	// generate the httpAPI client
	cli, err := clientapi.NewAPIClient(ctx, "http://localhost:5052", 50*time.Second, clientapi.WithELEndpoint("http://localhost:8545"))
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
	prevState, err := analyzer.cli.RequestBeaconState(slot - spec.SlotsPerEpoch)
	if err != nil {
		return EpochTask{}, fmt.Errorf("could not download state: %s", err)

	}

	fmt.Printf("downloading state at slot: %d\n", slot)
	currentState, err := analyzer.cli.RequestBeaconState(slot)
	if err != nil {
		return EpochTask{}, fmt.Errorf("could not download state: %s", err)
	}

	fmt.Printf("downloading state at slot: %d\n", slot+spec.SlotsPerEpoch)
	nextState, err := analyzer.cli.RequestBeaconState(slot + spec.SlotsPerEpoch)
	if err != nil {
		return EpochTask{}, fmt.Errorf("could not download state: %s", err)
	}

	return EpochTask{
		PrevState: *prevState,
		State:     *currentState,
		NextState: *nextState,
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

	epochTask, err := BuildEpochTask(analyzer, 6565759) // epoch 205179
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
		int64(12322))

	assert.Equal(t,
		rewards.AttestationReward,
		phase0.Gwei(12322))

	assert.Equal(t,
		rewards.AttSlot,
		phase0.Slot(6565698))
	assert.Equal(t,
		rewards.BaseReward,
		phase0.Gwei(14816))
	assert.Equal(t,
		rewards.MaxReward,
		phase0.Gwei(12322))

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
		phase0.Gwei(36586892613))

	// Test when validator did perform duties and was in a sync committee
	rewards, err = stateMetrics.GetMaxReward(325479)
	assert.Equal(t,
		rewards.Reward,
		int64(517882))

	assert.Equal(t,
		rewards.AttestationReward,
		phase0.Gwei(12322))

	assert.Equal(t,
		rewards.AttSlot,
		phase0.Slot(6565704))
	assert.Equal(t,
		rewards.BaseReward,
		phase0.Gwei(14816))
	assert.Equal(t,
		rewards.MaxReward,
		phase0.Gwei(517882))

	assert.Equal(t,
		rewards.SyncCommitteeReward,
		phase0.Gwei(505560))
	assert.Equal(t,
		rewards.InSyncCommittee,
		true)
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
		phase0.Gwei(32078071168))
}

func TestAltairNegativeRewards(t *testing.T) {

	analyzer, err := BuildStateAnalyzer()
	if err != nil {
		fmt.Errorf("could not build analyzer: %s", err)
		return
	}

	epochTask, err := BuildEpochTask(analyzer, 6565823) // epoch 205181
	if err != nil {
		fmt.Errorf("could not build epoch task: %s", err)
		return
	}

	// returns the state in a custom struct for Phase0, Altair of Bellatrix
	stateMetrics, err := metrics.StateMetricsByForkVersion(epochTask.NextState, epochTask.State, epochTask.PrevState, analyzer.cli.Api)

	// Test negative rewards
	rewards, err := stateMetrics.GetMaxReward(9097)
	assert.Equal(t,
		rewards.Reward,
		int64(-9260))

	assert.Equal(t,
		rewards.AttestationReward,
		phase0.Gwei(12122))

	assert.Equal(t,
		rewards.AttSlot,
		phase0.Slot(6565786))
	assert.Equal(t,
		rewards.BaseReward,
		phase0.Gwei(14816))
	assert.Equal(t,
		rewards.MaxReward,
		phase0.Gwei(12122))

	assert.Equal(t,
		rewards.SyncCommitteeReward,
		phase0.Gwei(0))
	assert.Equal(t,
		rewards.InSyncCommittee,
		false)
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
		phase0.Gwei(36855786132))
}

func TestCapellaBlock(t *testing.T) {

	blockAnalyzer, err := BuildBlockAnalyzer()

	if err != nil {
		fmt.Errorf("could not build analyzer: %s", err)
		return
	}

	// Test regular case

	block, proposed, err := blockAnalyzer.cli.RequestBeaconBlock(6564725)

	assert.Equal(t, proposed, true)

	assert.Equal(t, strings.ReplaceAll(string(block.Graffiti[:]), "\u0000", ""), "Stakewise_aonif")
	assert.Equal(t, len(block.Attestations), 65)
	assert.Equal(t, len(block.Deposits), 0)
	assert.Equal(t, block.ProposerIndex, phase0.ValidatorIndex(646459))
	assert.Equal(t, block.Proposed, true)
	assert.Equal(t, block.Slot, phase0.Slot(6564725))
	assert.Equal(t, block.ExecutionPayload.BlockHash.String(), "0xdbdb4d20266578de916de5b052f500c9d92633b7d9017e9193e4b4f90c086c89")
	assert.Equal(t, block.ExecutionPayload.BlockNumber, uint64(17384171))
	assert.Equal(t, block.ExecutionPayload.FeeRecipient.String(), strings.ToLower("0x6b333B20fBae3c5c0969dd02176e30802e2fbBdB"))
	assert.Equal(t, block.ExecutionPayload.GasLimit, uint64(30000000))
	assert.Equal(t, block.ExecutionPayload.GasUsed, uint64(22774075))
	assert.Equal(t, block.ExecutionPayload.Timestamp, uint64(1685600723))
	assert.Equal(t, len(block.ExecutionPayload.Transactions), 222)
	assert.Equal(t, len(block.ExecutionPayload.Withdrawals), 16)

	transactions, err := blockAnalyzer.cli.RequestTransactionDetails(block)
	if err != nil {
		fmt.Errorf("could not retrieve transaction details: %s", err)
		return
	}
	assert.Equal(t, transactions[0].Hash.String(), "0xa8ee3de535f01a6df2e117af8d7142ea811ffeeda3a1b4e604ad357db2924ec4")

	// Test missed

	block, proposed, err = blockAnalyzer.cli.RequestBeaconBlock(6564753)

	assert.Equal(t, proposed, false)

	assert.Equal(t, strings.ReplaceAll(string(block.Graffiti[:]), "\u0000", ""), "")
	assert.Equal(t, len(block.Attestations), 0)
	assert.Equal(t, len(block.Deposits), 0)
	assert.Equal(t, block.ProposerIndex, phase0.ValidatorIndex(565236))
	assert.Equal(t, block.Proposed, false)
	assert.Equal(t, block.Slot, phase0.Slot(6564753))
	assert.Equal(t, block.ExecutionPayload.BlockHash.String(), "0x0000000000000000000000000000000000000000000000000000000000000000")
	assert.Equal(t, block.ExecutionPayload.BlockNumber, uint64(0))
	assert.Equal(t, block.ExecutionPayload.FeeRecipient.String(), strings.ToLower("0x0000000000000000000000000000000000000000"))
	assert.Equal(t, block.ExecutionPayload.GasLimit, uint64(0))
	assert.Equal(t, block.ExecutionPayload.GasUsed, uint64(0))
	assert.Equal(t, block.ExecutionPayload.Timestamp, uint64(0))
	assert.Equal(t, len(block.ExecutionPayload.Transactions), 0)
	assert.Equal(t, len(block.ExecutionPayload.Withdrawals), 0)

}

func TestBellatrixBlock(t *testing.T) {

	blockAnalyzer, err := BuildBlockAnalyzer()

	if err != nil {
		fmt.Errorf("could not build analyzer: %s", err)
		return
	}

	// Test regular case

	block, proposed, err := blockAnalyzer.cli.RequestBeaconBlock(4709993)

	assert.Equal(t, proposed, true)

	assert.Equal(t, strings.ReplaceAll(string(block.Graffiti[:]), "\u0000", ""), "Lighthouse/v3.1.0-aa022f4")
	assert.Equal(t, len(block.Attestations), 128)
	assert.Equal(t, len(block.Deposits), 0)
	assert.Equal(t, block.ProposerIndex, phase0.ValidatorIndex(361543))
	assert.Equal(t, block.Proposed, true)
	assert.Equal(t, block.Slot, phase0.Slot(4709993))
	assert.Equal(t, block.ExecutionPayload.BlockHash.String(), "0xdf8c3becf096dd2f50ae3848a6c3c9bc6b6b68eb5c00051d76b8dff82919e2db")
	assert.Equal(t, block.ExecutionPayload.BlockNumber, uint64(15547218))
	assert.Equal(t, block.ExecutionPayload.FeeRecipient.String(), strings.ToLower("0xb64a30399f7F6b0C154c2E7Af0a3ec7B0A5b131a"))
	assert.Equal(t, block.ExecutionPayload.GasLimit, uint64(30000000))
	assert.Equal(t, block.ExecutionPayload.GasUsed, uint64(29993683))
	assert.Equal(t, block.ExecutionPayload.Timestamp, uint64(1663343939))
	assert.Equal(t, len(block.ExecutionPayload.Transactions), 441)
	assert.Equal(t, len(block.ExecutionPayload.Withdrawals), 0)

	transactions, err := blockAnalyzer.cli.RequestTransactionDetails(block)

	if err != nil {
		fmt.Errorf("could not retrieve transaction details: %s", err)
		return
	}
	assert.Equal(t, transactions[0].Hash.String(), "0x2ec2cc18a5db329ef76b71db72f47611b49d51b780f5e0140c455320d1278d41")

	// Test missed

	block, proposed, err = blockAnalyzer.cli.RequestBeaconBlock(4709992)

	assert.Equal(t, proposed, false)

	assert.Equal(t, strings.ReplaceAll(string(block.Graffiti[:]), "\u0000", ""), "")
	assert.Equal(t, len(block.Attestations), 0)
	assert.Equal(t, len(block.Deposits), 0)
	assert.Equal(t, block.ProposerIndex, phase0.ValidatorIndex(43851))
	assert.Equal(t, block.Proposed, false)
	assert.Equal(t, block.Slot, phase0.Slot(4709992))
	assert.Equal(t, block.ExecutionPayload.BlockHash.String(), "0x0000000000000000000000000000000000000000000000000000000000000000")
	assert.Equal(t, block.ExecutionPayload.BlockNumber, uint64(0))
	assert.Equal(t, block.ExecutionPayload.FeeRecipient.String(), strings.ToLower("0x0000000000000000000000000000000000000000"))
	assert.Equal(t, block.ExecutionPayload.GasLimit, uint64(0))
	assert.Equal(t, block.ExecutionPayload.GasUsed, uint64(0))
	assert.Equal(t, block.ExecutionPayload.Timestamp, uint64(0))
	assert.Equal(t, len(block.ExecutionPayload.Transactions), 0)
	assert.Equal(t, len(block.ExecutionPayload.Withdrawals), 0)
}

func TestAltairBlock(t *testing.T) {

	blockAnalyzer, err := BuildBlockAnalyzer()

	if err != nil {
		fmt.Errorf("could not build analyzer: %s", err)
		return
	}

	// Test regular case

	block, proposed, err := blockAnalyzer.cli.RequestBeaconBlock(4636687)

	assert.Equal(t, proposed, true)

	assert.Equal(t, strings.ReplaceAll(string(block.Graffiti[:]), "\u0000", ""), "")
	assert.Equal(t, len(block.Attestations), 128)
	assert.Equal(t, len(block.Deposits), 0)
	assert.Equal(t, block.ProposerIndex, phase0.ValidatorIndex(319226))
	assert.Equal(t, block.Proposed, true)
	assert.Equal(t, block.Slot, phase0.Slot(4636687))
	assert.Equal(t, block.ExecutionPayload.BlockHash.String(), "0x0000000000000000000000000000000000000000000000000000000000000000")
	assert.Equal(t, block.ExecutionPayload.BlockNumber, uint64(0))
	assert.Equal(t, block.ExecutionPayload.FeeRecipient.String(), strings.ToLower("0x0000000000000000000000000000000000000000"))
	assert.Equal(t, block.ExecutionPayload.GasLimit, uint64(0))
	assert.Equal(t, block.ExecutionPayload.GasUsed, uint64(0))
	assert.Equal(t, block.ExecutionPayload.Timestamp, uint64(0))
	assert.Equal(t, len(block.ExecutionPayload.Transactions), 0)
	assert.Equal(t, len(block.ExecutionPayload.Withdrawals), 0)

	// Test missed

	block, proposed, err = blockAnalyzer.cli.RequestBeaconBlock(4709992)

	assert.Equal(t, proposed, false)

	assert.Equal(t, strings.ReplaceAll(string(block.Graffiti[:]), "\u0000", ""), "")
	assert.Equal(t, len(block.Attestations), 0)
	assert.Equal(t, len(block.Deposits), 0)
	assert.Equal(t, block.ProposerIndex, phase0.ValidatorIndex(43851))
	assert.Equal(t, block.Proposed, false)
	assert.Equal(t, block.Slot, phase0.Slot(4709992))
	assert.Equal(t, block.ExecutionPayload.BlockHash.String(), "0x0000000000000000000000000000000000000000000000000000000000000000")
	assert.Equal(t, block.ExecutionPayload.BlockNumber, uint64(0))
	assert.Equal(t, block.ExecutionPayload.FeeRecipient.String(), strings.ToLower("0x0000000000000000000000000000000000000000"))
	assert.Equal(t, block.ExecutionPayload.GasLimit, uint64(0))
	assert.Equal(t, block.ExecutionPayload.GasUsed, uint64(0))
	assert.Equal(t, block.ExecutionPayload.Timestamp, uint64(0))
	assert.Equal(t, len(block.ExecutionPayload.Transactions), 0)
	assert.Equal(t, len(block.ExecutionPayload.Withdrawals), 0)
}

func TestPhase0Block(t *testing.T) {

	blockAnalyzer, err := BuildBlockAnalyzer()

	if err != nil {
		fmt.Errorf("could not build analyzer: %s", err)
		return
	}

	// Test regular case

	block, proposed, err := blockAnalyzer.cli.RequestBeaconBlock(2372310)

	assert.Equal(t, proposed, true)

	assert.Equal(t, strings.ReplaceAll(string(block.Graffiti[:]), "\u0000", ""), "")
	assert.Equal(t, len(block.Attestations), 128)
	assert.Equal(t, len(block.Deposits), 0)
	assert.Equal(t, block.ProposerIndex, phase0.ValidatorIndex(162042))
	assert.Equal(t, block.Proposed, true)
	assert.Equal(t, block.Slot, phase0.Slot(2372310))
	assert.Equal(t, block.ExecutionPayload.BlockHash.String(), "0x0000000000000000000000000000000000000000000000000000000000000000")
	assert.Equal(t, block.ExecutionPayload.BlockNumber, uint64(0))
	assert.Equal(t, block.ExecutionPayload.FeeRecipient.String(), strings.ToLower("0x0000000000000000000000000000000000000000"))
	assert.Equal(t, block.ExecutionPayload.GasLimit, uint64(0))
	assert.Equal(t, block.ExecutionPayload.GasUsed, uint64(0))
	assert.Equal(t, block.ExecutionPayload.Timestamp, uint64(0))
	assert.Equal(t, len(block.ExecutionPayload.Transactions), 0)
	assert.Equal(t, len(block.ExecutionPayload.Withdrawals), 0)

	// Test missed

	block, proposed, err = blockAnalyzer.cli.RequestBeaconBlock(2372309)

	assert.Equal(t, proposed, false)

	assert.Equal(t, strings.ReplaceAll(string(block.Graffiti[:]), "\u0000", ""), "")
	assert.Equal(t, len(block.Attestations), 0)
	assert.Equal(t, len(block.Deposits), 0)
	assert.Equal(t, block.ProposerIndex, phase0.ValidatorIndex(31106))
	assert.Equal(t, block.Proposed, false)
	assert.Equal(t, block.Slot, phase0.Slot(2372309))
	assert.Equal(t, block.ExecutionPayload.BlockHash.String(), "0x0000000000000000000000000000000000000000000000000000000000000000")
	assert.Equal(t, block.ExecutionPayload.BlockNumber, uint64(0))
	assert.Equal(t, block.ExecutionPayload.FeeRecipient.String(), strings.ToLower("0x0000000000000000000000000000000000000000"))
	assert.Equal(t, block.ExecutionPayload.GasLimit, uint64(0))
	assert.Equal(t, block.ExecutionPayload.GasUsed, uint64(0))
	assert.Equal(t, block.ExecutionPayload.Timestamp, uint64(0))
	assert.Equal(t, len(block.ExecutionPayload.Transactions), 0)
	assert.Equal(t, len(block.ExecutionPayload.Withdrawals), 0)
}

func TestTransactionHasEffectiveGasPriceWhenELIsProvided(t *testing.T) {
	blockAnalyzer, err := BuildBlockAnalyzerWithEL()

	if err != nil {
		fmt.Errorf("could not build analyzer: %s", err)
		return
	}

	// Test regular case
	block, proposed, err := blockAnalyzer.cli.RequestBeaconBlock(5610381) //block number 16442285
	assert.Equal(t, proposed, true)

	transactions, err := blockAnalyzer.cli.RequestTransactionDetails(block)

	if err != nil {
		fmt.Errorf("could not retrieve transaction details: %s", err)
		return
	}

	// Test retrieved transaction gas price is the same with that expected to be effective gas price
	assert.Equal(t, transactions[10].Hash.String(), "0x737319e9325ddbb754908d0874dac9f95fbb6c1f49cf5f88c389b98ebb61d36c")
	assert.Equal(t, transactions[10].GasPrice, phase0.Gwei(94245102754))
}

func TestTransactionNotEffectiveGasPriceWhenELNotProvided(t *testing.T) {
	blockAnalyzer, err := BuildBlockAnalyzer()

	if err != nil {
		fmt.Errorf("could not build analyzer: %s", err)
		return
	}

	// Test regular case
	block, proposed, err := blockAnalyzer.cli.RequestBeaconBlock(5610381) //block number 16442285
	assert.Equal(t, proposed, true)

	transactions, err := blockAnalyzer.cli.RequestTransactionDetails(block)

	if err != nil {
		fmt.Errorf("could not retrieve transaction details: %s", err)
		return
	}

	// Test retrieved transaction gas price is not same with that expected to be effective gas price
	assert.Equal(t, transactions[10].Hash.String(), "0x737319e9325ddbb754908d0874dac9f95fbb6c1f49cf5f88c389b98ebb61d36c")
	assert.NotEqual(t, transactions[10].GasPrice, phase0.Gwei(94245102754))
}

func TestBlockSizeIsSetWhenELIsProvided(t *testing.T) {
	blockAnalyzer, err := BuildBlockAnalyzerWithEL()

	if err != nil {
		fmt.Errorf("could not build analyzer: %s", err)
		return
	}

	block, _, err := blockAnalyzer.cli.RequestBeaconBlock(5610381) //block number 16442285
	assert.Equal(t, block.Size, uint32(69157))
}

func TestBlockSizeNotSetWhenELNotProvided(t *testing.T) {
	blockAnalyzer, err := BuildBlockAnalyzer()

	if err != nil {
		fmt.Errorf("could not build analyzer: %s", err)
		return
	}

	block, _, err := blockAnalyzer.cli.RequestBeaconBlock(5610381) //block number 16442285
	assert.Equal(t, block.Size, uint32(0))
}
