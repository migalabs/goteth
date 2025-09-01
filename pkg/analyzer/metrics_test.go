package analyzer

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/migalabs/goteth/pkg/clientapi"
	"github.com/migalabs/goteth/pkg/db"
	"github.com/migalabs/goteth/pkg/spec"
	"github.com/migalabs/goteth/pkg/spec/metrics"
	"github.com/stretchr/testify/assert"
)

func BuildChainAnalyzer() (ChainAnalyzer, error) {

	ctx := context.Background()

	dbMetrics := db.DBMetrics{
		Transactions:     true,
		Block:            true,
		Epoch:            true,
		ValidatorRewards: true,
		APIRewards:       true,
	}
	maxRequestRetries := 3
	// generate the httpAPI client
	cli, err := clientapi.NewAPIClient(
		ctx,
		"http://localhost:5052",
		maxRequestRetries,
		clientapi.WithELEndpoint("http://localhost:8545"),
		clientapi.WithDBMetrics(dbMetrics))
	if err != nil {
		return ChainAnalyzer{}, err
	}

	return ChainAnalyzer{
		ctx:     context.Background(),
		cli:     cli,
		metrics: dbMetrics,
	}, nil
}

func BuildChainAnalyzerWithEL() (ChainAnalyzer, error) {

	ctx := context.Background()
	maxRequestRetries := 3
	// generate the httpAPI client
	cli, err := clientapi.NewAPIClient(ctx, "http://localhost:5052", maxRequestRetries, clientapi.WithELEndpoint("http://localhost:8545"))
	if err != nil {
		return ChainAnalyzer{}, err
	}

	return ChainAnalyzer{
		ctx: context.Background(),
		cli: cli,
	}, nil
}

func BuildEpochTask(analyzer *ChainAnalyzer, slot phase0.Slot) (metrics.StateMetrics, error) {

	// Review slot is well positioned

	epoch := slot / spec.SlotsPerEpoch

	slot = ((epoch + 1) * spec.SlotsPerEpoch) - 1

	fmt.Printf("downloading state at slot: %d\n", slot-spec.SlotsPerEpoch)
	prevState, err := analyzer.cli.RequestBeaconState(slot - spec.SlotsPerEpoch)
	if err != nil {
		return metrics.Phase0Metrics{}, fmt.Errorf("could not download state: %s", err)

	}

	fmt.Printf("downloading state at slot: %d\n", slot)
	currentState, err := analyzer.cli.RequestBeaconState(slot)
	if err != nil {
		return metrics.Phase0Metrics{}, fmt.Errorf("could not download state: %s", err)
	}

	fmt.Printf("downloading state at slot: %d\n", slot+spec.SlotsPerEpoch)
	nextState, err := analyzer.cli.RequestBeaconState(slot + spec.SlotsPerEpoch)
	if err != nil {
		return metrics.Phase0Metrics{}, fmt.Errorf("could not download state: %s", err)
	}

	bundle, err := metrics.StateMetricsByForkVersion(nextState, currentState, prevState, analyzer.cli.Api)
	if err != nil {
		return metrics.Phase0Metrics{}, fmt.Errorf("could not build bundle: %s", err)
	}
	return bundle, nil

}

func TestPhase0Epoch(t *testing.T) {

	analyzer, err := BuildChainAnalyzer()
	if err != nil {
		t.Errorf("could not build analyzer: %s", err)
		return
	}

	// returns the state in a custom struct for Phase0, Altair of Bellatrix
	stateMetrics, err := BuildEpochTask(&analyzer, 320031) // epoch 10000
	if err != nil {
		t.Errorf("could not build epoch task: %s", err)
		return
	}
	assert.Equal(t, stateMetrics.GetMetricsBase().CurrentState.NumActiveVals, uint(60849))
	assert.Equal(t,
		stateMetrics.GetMetricsBase().CurrentState.MissedBlocks,
		[]phase0.Slot{320011, 320023})

}

func TestAltairEpoch(t *testing.T) {

	analyzer, err := BuildChainAnalyzer()
	if err != nil {
		t.Errorf("could not build analyzer: %s", err)
		return
	}

	// returns the state in a custom struct for Phase0, Altair of Bellatrix
	stateMetrics, err := BuildEpochTask(&analyzer, 2375711) // epoch 74240
	if err != nil {
		t.Errorf("could not build epoch task: %s", err)
		return
	}

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

	analyzer, err := BuildChainAnalyzer()
	if err != nil {
		t.Errorf("could not build analyzer: %s", err)
		return
	}

	// returns the state in a custom struct for Phase0, Altair of Bellatrix
	stateMetrics, err := BuildEpochTask(&analyzer, 6565759) // epoch 205179
	if err != nil {
		t.Errorf("could not build epoch task: %s", err)
		return
	}

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

	analyzer, err := BuildChainAnalyzer()
	if err != nil {
		t.Errorf("could not build analyzer: %s", err)
		return
	}
	// returns the state in a custom struct for Phase0, Altair of Bellatrix
	stateMetrics, err := BuildEpochTask(&analyzer, 6565823) // epoch 205181
	if err != nil {
		t.Errorf("could not build epoch task: %s", err)
		return
	}

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

	blockAnalyzer, err := BuildChainAnalyzerWithEL()

	if err != nil {
		t.Errorf("could not build analyzer: %s", err)
		return
	}

	// Test regular case

	block, err := blockAnalyzer.cli.RequestBeaconBlock(6564725)

	assert.Equal(t, block.Proposed, true)

	assert.Equal(t, strings.ReplaceAll(string(block.Graffiti[:]), "\u0000", ""), "Stakewise_aonif")
	assert.Equal(t, len(block.Attestations), 65)
	assert.Equal(t, len(block.Deposits), 0)
	assert.Equal(t, block.ProposerIndex, phase0.ValidatorIndex(646459))
	assert.Equal(t, block.Proposed, true)
	assert.Equal(t, block.Slot, phase0.Slot(6564725))
	assert.Equal(t, block.ExecutionPayload.BlockHash.String(), "0xdbdb4d20266578de916de5b052f500c9d92633b7d9017e9193e4b4f90c086c89")
	assert.Equal(t, block.ExecutionPayload.BlockNumber, uint64(17384171))
	assert.Equal(t, block.ExecutionPayload.FeeRecipient.String(), "0x6b333B20fBae3c5c0969dd02176e30802e2fbBdB")
	assert.Equal(t, block.ExecutionPayload.GasLimit, uint64(30000000))
	assert.Equal(t, block.ExecutionPayload.GasUsed, uint64(22774075))
	assert.Equal(t, block.ExecutionPayload.Timestamp, uint64(1685600723))
	assert.Equal(t, len(block.ExecutionPayload.Transactions), 222)
	assert.Equal(t, len(block.ExecutionPayload.Withdrawals), 16)

	tx := block.ExecutionPayload.Transactions[0]
	receipt, err := blockAnalyzer.cli.GetTransactionReceipt(tx, block.Slot, block.ExecutionPayload.BlockNumber, block.ExecutionPayload.Timestamp)
	var parsedTx = &types.Transaction{}
	if err := parsedTx.UnmarshalBinary(tx); err != nil {
		t.Errorf("could not unmarshal transaction: %s", err)
		return
	}
	transaction, err := spec.ParseTransactionFromReceipt(parsedTx, receipt, block.Slot, block.ExecutionPayload.BlockNumber, block.ExecutionPayload.Timestamp, uint64(0))

	if err != nil {
		t.Errorf("could not retrieve transaction details: %s", err)
		return
	}
	assert.Equal(t, transaction.Hash.String(), "0xa8ee3de535f01a6df2e117af8d7142ea811ffeeda3a1b4e604ad357db2924ec4")

	// Test missed

	block, err = blockAnalyzer.cli.RequestBeaconBlock(6564753)

	assert.Equal(t, block.Proposed, false)

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

	blockAnalyzer, err := BuildChainAnalyzer()

	if err != nil {
		t.Errorf("could not build analyzer: %s", err)
		return
	}

	// Test regular case

	block, err := blockAnalyzer.cli.RequestBeaconBlock(4709993)

	assert.Equal(t, block.Proposed, true)

	assert.Equal(t, strings.ReplaceAll(string(block.Graffiti[:]), "\u0000", ""), "Lighthouse/v3.1.0-aa022f4")
	assert.Equal(t, len(block.Attestations), 128)
	assert.Equal(t, len(block.Deposits), 0)
	assert.Equal(t, block.ProposerIndex, phase0.ValidatorIndex(361543))
	assert.Equal(t, block.Proposed, true)
	assert.Equal(t, block.Slot, phase0.Slot(4709993))
	assert.Equal(t, block.ExecutionPayload.BlockHash.String(), "0xdf8c3becf096dd2f50ae3848a6c3c9bc6b6b68eb5c00051d76b8dff82919e2db")
	assert.Equal(t, block.ExecutionPayload.BlockNumber, uint64(15547218))
	assert.Equal(t, block.ExecutionPayload.FeeRecipient.String(), "0xb64a30399f7F6b0C154c2E7Af0a3ec7B0A5b131a")
	assert.Equal(t, block.ExecutionPayload.GasLimit, uint64(30000000))
	assert.Equal(t, block.ExecutionPayload.GasUsed, uint64(29993683))
	assert.Equal(t, block.ExecutionPayload.Timestamp, uint64(1663343939))
	assert.Equal(t, len(block.ExecutionPayload.Transactions), 441)
	assert.Equal(t, len(block.ExecutionPayload.Withdrawals), 0)

	tx := block.ExecutionPayload.Transactions[0]

	receipt, err := blockAnalyzer.cli.GetTransactionReceipt(tx, block.Slot, block.ExecutionPayload.BlockNumber, block.ExecutionPayload.Timestamp)
	var parsedTx = &types.Transaction{}
	if err := parsedTx.UnmarshalBinary(tx); err != nil {
		t.Errorf("could not unmarshal transaction: %s", err)
		return
	}

	transaction, err := spec.ParseTransactionFromReceipt(parsedTx, receipt, block.Slot, block.ExecutionPayload.BlockNumber, block.ExecutionPayload.Timestamp, uint64(0))

	if err != nil {
		t.Errorf("could not retrieve transaction details: %s", err)
		return
	}
	assert.Equal(t, transaction.Hash.String(), "0x2ec2cc18a5db329ef76b71db72f47611b49d51b780f5e0140c455320d1278d41")

	// Test missed

	block, err = blockAnalyzer.cli.RequestBeaconBlock(4709992)

	assert.Equal(t, block.Proposed, false)

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

	blockAnalyzer, err := BuildChainAnalyzer()

	if err != nil {
		t.Errorf("could not build analyzer: %s", err)
		return
	}

	// Test regular case

	block, err := blockAnalyzer.cli.RequestBeaconBlock(4636687)

	assert.Equal(t, block.Proposed, true)

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

	block, err = blockAnalyzer.cli.RequestBeaconBlock(4709992)

	assert.Equal(t, block.Proposed, false)

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

	blockAnalyzer, err := BuildChainAnalyzer()

	if err != nil {
		t.Errorf("could not build analyzer: %s", err)
		return
	}

	// Test regular case

	block, err := blockAnalyzer.cli.RequestBeaconBlock(2372310)

	assert.Equal(t, block.Proposed, true)

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

	block, err = blockAnalyzer.cli.RequestBeaconBlock(2372309)

	assert.Equal(t, block.Proposed, false)

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

func TestTransactionGasWhenELIsProvided(t *testing.T) {
	blockAnalyzer, err := BuildChainAnalyzerWithEL()

	if err != nil {
		t.Errorf("could not build analyzer: %s", err)
		return
	}

	// Test regular case
	block, err := blockAnalyzer.cli.RequestBeaconBlock(8020631) //block number 18826815
	if err != nil {
		t.Errorf("could not download block: %s", err)
		return
	}
	assert.Equal(t, block.Proposed, true)

	tx := block.ExecutionPayload.Transactions[10]
	receipt, err := blockAnalyzer.cli.GetTransactionReceipt(tx, block.Slot, block.ExecutionPayload.BlockNumber, block.ExecutionPayload.Timestamp)
	if err != nil {
		t.Errorf("could not retrieve transaction receipt: %s", err)
		return
	}
	var parsedTx = &types.Transaction{}
	if err := parsedTx.UnmarshalBinary(tx); err != nil {
		t.Errorf("could not unmarshal transaction: %s", err)
		return
	}

	transaction, err := spec.ParseTransactionFromReceipt(parsedTx, receipt, block.Slot, block.ExecutionPayload.BlockNumber, block.ExecutionPayload.Timestamp, uint64(0))

	if err != nil {
		t.Errorf("could not retrieve transaction details: %s", err)
		return
	}

	// Test retrieved transaction gas price is the same with that expected to be effective gas price
	assert.Equal(t, transaction.Hash.String(), "0x240a131dd882a06681c43cfa13f8e5013e7bdea5e7285710643c40e3321c014a")
	assert.Equal(t, transaction.Gas, phase0.Gwei(51617))
}

func TestTransactionGasWhenELNotProvided(t *testing.T) {
	blockAnalyzer, err := BuildChainAnalyzer()

	if err != nil {
		t.Errorf("could not build analyzer: %s", err)
		return
	}

	// Test regular case
	block, err := blockAnalyzer.cli.RequestBeaconBlock(8020631) //block number 18826815
	if err != nil {
		t.Errorf("could not download block: %s", err)
		return
	}
	assert.Equal(t, block.Proposed, true)

	tx := block.ExecutionPayload.Transactions[10]
	receipt, err := blockAnalyzer.cli.GetTransactionReceipt(tx, block.Slot, block.ExecutionPayload.BlockNumber, block.ExecutionPayload.Timestamp)
	if err != nil {
		t.Errorf("could not retrieve transaction receipt: %s", err)
		return
	}
	var parsedTx = &types.Transaction{}
	if err := parsedTx.UnmarshalBinary(tx); err != nil {
		t.Errorf("could not unmarshal transaction: %s", err)
		return
	}

	transaction, err := spec.ParseTransactionFromReceipt(parsedTx, receipt, block.Slot, block.ExecutionPayload.BlockNumber, block.ExecutionPayload.Timestamp, uint64(0))

	if err != nil {
		t.Errorf("could not retrieve transaction details: %s", err)
		return
	}

	// Test retrieved transaction gas price is not same with that expected to be effective gas price
	assert.Equal(t, transaction.Hash.String(), "0x240a131dd882a06681c43cfa13f8e5013e7bdea5e7285710643c40e3321c014a")
	assert.Equal(t, transaction.Gas, phase0.Gwei(120000))
}

func TestBlockSizeIsSetWhenELIsProvided(t *testing.T) {
	blockAnalyzer, err := BuildChainAnalyzerWithEL()

	if err != nil {
		t.Errorf("could not build analyzer: %s", err)
		return
	}

	block, _ := blockAnalyzer.cli.RequestBeaconBlock(5610381) //block number 16442285
	assert.Equal(t, block.ExecutionPayload.PayloadSize, uint32(69157))
}

func TestBlockSizeNotSetWhenELNotProvided(t *testing.T) {
	blockAnalyzer, err := BuildChainAnalyzer()

	if err != nil {
		t.Errorf("could not build analyzer: %s", err)
		return
	}

	block, _ := blockAnalyzer.cli.RequestBeaconBlock(5610381) //block number 16442285
	assert.Equal(t, block.ExecutionPayload.PayloadSize, uint32(0))
}

func TestBlockGasFees(t *testing.T) {

	analyzer, err := BuildChainAnalyzer()
	if err != nil {
		t.Errorf("could not build analyzer: %s", err)
		return
	}

	block, err := analyzer.cli.RequestBeaconBlock(8790975)
	if err != nil {
		t.Errorf("could not download block: %s", err)
		return
	}

	reward, burn, err := block.BlockGasFees()

	if err != nil {
		t.Errorf("could not calculate block gas fees: %s", err)
		return
	}

	assert.Equal(t, reward, uint64(44861896127679906))
	assert.Equal(t, burn, uint64(317246355753369564))

}
