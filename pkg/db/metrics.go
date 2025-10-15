package db

import (
	"fmt"
	"strings"
)

type DBMetrics struct {
	Block            bool
	Epoch            bool
	ValidatorRewards bool
	APIRewards       bool
	Transactions     bool
	BlobSidecars     bool
}

func NewMetrics(input string) (DBMetrics, error) {
	dbMetrics := DBMetrics{}

	for _, item := range strings.Split(input, ",") {

		switch item {
		case "block":
			dbMetrics.Block = true
		case "epoch":
			dbMetrics.Epoch = true
			dbMetrics.Block = true
		case "rewards":
			dbMetrics.ValidatorRewards = true
			dbMetrics.Epoch = true
			dbMetrics.Block = true
		case "api_rewards":
			dbMetrics.APIRewards = true
		case "transactions":
			dbMetrics.Transactions = true
			dbMetrics.Block = true
		case "blob_sidecars":
			dbMetrics.Block = true
			dbMetrics.BlobSidecars = true
		default:
			return DBMetrics{}, fmt.Errorf("could not parse metric: %s", item)
		}
	}
	return dbMetrics, nil
}
