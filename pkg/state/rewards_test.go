package state

import (
	"testing"
)

func TestParticipationRatePhase0(t *testing.T) {

	// ctx := context.Background()
	// log = logrus.WithField(
	// 	"cli", "CliName",
	// )
	// bnEndpoint := "https://20PdJoS82pnejJJ9joDMnbjsQ32:0c9b868d8621332ea91c7fc24c5fc34f@eth2-beacon-mainnet.infura.io"
	// queryEndpoint := time.Duration(time.Second * 90)
	// cli, err := clientapi.NewAPIClient(ctx, bnEndpoint, queryEndpoint)
	// if err != nil {
	// 	fmt.Println("could not create API client", err)
	// }

	// for i := 95; i < 200; i = i + 32 {
	// 	bstate, err := cli.Api.BeaconState(ctx, fmt.Sprintf("%d", i))

	// 	phase0State := custom_spec.NewPhase0Spec(bstate)

	// 	if err != nil {
	// 		fmt.Println("could not parse State Fork Version", err)
	// 	}

	// 	valsVotes := phase0State.PreviousEpochAttestations() // max one vote per validator
	// 	doubleVotes := phase0State.GetDoubleVotes()

	// 	previousEpochAggregations := phase0State.PreviousEpochAggregations()

	// 	totalVotesPreviousEpoch := 0
	// 	for _, aggregation := range previousEpochAggregations {
	// 		totalVotesPreviousEpoch += int(aggregation.AggregationBits.Count())
	// 	}

	// 	fmt.Println(valsVotes, doubleVotes, totalVotesPreviousEpoch)

	// 	assert.Equal(t, totalVotesPreviousEpoch, int(valsVotes+doubleVotes))
	// }

}

func TestParticipationRateAltair(t *testing.T) {

	// ctx := context.Background()
	// log = logrus.WithField(
	// 	"cli", "CliName",
	// )

	// bnEndpoint := "http://localhost:5052"
	// queryEndpoint := time.Duration(time.Second * 90)
	// cli, err := clientapi.NewAPIClient(ctx, bnEndpoint, queryEndpoint)
	// if err != nil {
	// 	fmt.Println("could not create API client", err)
	// }

	// for i := 2000; i < 2033; i = i + 32 {
	// 	bstate, err := cli.Api.BeaconState(ctx, fmt.Sprintf("%d", i))

	// 	altairState := custom_spec.NewAltairSpec(bstate)

	// 	if err != nil {
	// 		fmt.Println("could not parse State Fork Version", err)
	// 	}

	// 	valsVotes := altairState.PreviousEpochAttestations() // max one vote per validator
	// 	missedVotes := altairState.PreviousEpochMissedAttestations()
	// 	totalVals := altairState.PreviousEpochValNum()

	// 	fmt.Println(valsVotes, missedVotes, totalVals)

	// 	assert.Equal(t, totalVals, uint64(valsVotes+missedVotes))
	// }

}

func TestParticipationRateBellatrix(t *testing.T) {

	// ctx := context.Background()
	// log = logrus.WithField(
	// 	"cli", "CliName",
	// )

	// bnEndpoint := "http://localhost:5052"
	// queryEndpoint := time.Duration(time.Second * 2000)
	// cli, err := clientapi.NewAPIClient(ctx, bnEndpoint, queryEndpoint)
	// if err != nil {
	// 	fmt.Println("could not create API client", err)
	// }

	// for i := 30000; i < 30032; i = i + 32 {
	// 	bstate, err := cli.Api.BeaconState(ctx, fmt.Sprintf("%d", i))

	// 	if err != nil {
	// 		fmt.Println(err)
	// 		t.Fail()
	// 		return
	// 	}

	// 	bellatrixState := custom_spec.NewBellatrixSpec(bstate)

	// 	if err != nil {
	// 		fmt.Println("could not parse State Fork Version", err)
	// 	}

	// 	valsVotes := bellatrixState.PreviousEpochAttestations() // max one vote per validator
	// 	missedVotes := bellatrixState.PreviousEpochMissedAttestations()
	// 	totalVals := bellatrixState.PreviousEpochValNum()

	// 	fmt.Println(valsVotes, missedVotes, totalVals)

	// 	assert.Equal(t, totalVals, uint64(valsVotes+missedVotes))
	// }

}
