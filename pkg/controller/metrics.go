package controller

import (
	"fmt"
	"strings"
)

type DBMetrics struct {
	Block               bool
	Epoch               bool
	PoolSummary         bool
	ProposerDuties      bool
	ValidatorLastStatus bool
	ValidatorRewards    bool
	Withdrawals         bool
}

func NewMetrics(input string) (DBMetrics, error) {
	dbMetrics := DBMetrics{}

	for _, item := range strings.Split(input, ",") {

		switch item {
		case "block":
			dbMetrics.Block = true
		case "epoch":
			dbMetrics.Epoch = true
		case "pool_summary":
			dbMetrics.PoolSummary = true
		case "validator_last_status":
			dbMetrics.ValidatorLastStatus = true
		case "validator":
			dbMetrics.ValidatorRewards = true
		case "withdrawals":
			dbMetrics.Withdrawals = true
		default:
			return DBMetrics{}, fmt.Errorf("could not parse metric: %s", item)
		}
	}
	return dbMetrics, nil
}
