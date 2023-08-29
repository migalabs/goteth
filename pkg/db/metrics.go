package db

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
	Transactions        bool

	BlockDownload   bool
	StateDownload   bool
	RewardsDownload bool
}

func NewMetrics(input string) (DBMetrics, error) {
	dbMetrics := DBMetrics{}

	for _, item := range strings.Split(input, ",") {

		switch item {
		case "block":
			dbMetrics.Block = true
			dbMetrics.BlockDownload = true
		case "epoch":
			dbMetrics.Epoch = true
			dbMetrics.StateDownload = true
		case "pool_summary":
			dbMetrics.PoolSummary = true
		case "validator_last_status":
			dbMetrics.ValidatorLastStatus = true
			dbMetrics.StateDownload = true
		case "validator":
			dbMetrics.ValidatorRewards = true
			dbMetrics.StateDownload = true
			dbMetrics.RewardsDownload = true
		case "withdrawals":
			dbMetrics.Withdrawals = true
			dbMetrics.BlockDownload = true
		case "transactions":
			dbMetrics.Transactions = true
			dbMetrics.BlockDownload = true
		default:
			return DBMetrics{}, fmt.Errorf("could not parse metric: %s", item)
		}
	}
	return dbMetrics, nil
}
