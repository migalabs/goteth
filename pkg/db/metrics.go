package db

import (
	"fmt"
	"strings"
)

type DBMetrics struct {
	Block            bool
	Epoch            bool
	ValidatorRewards bool
	Transactions     bool
}

func NewMetrics(input string) (DBMetrics, error) {
	dbMetrics := DBMetrics{}

	for _, item := range strings.Split(input, ",") {

		switch item {
		case "block":
			dbMetrics.Block = true
		case "epoch":
			dbMetrics.Epoch = true
		case "rewards":
			dbMetrics.ValidatorRewards = true
		case "transactions":
			dbMetrics.Transactions = true
		default:
			return DBMetrics{}, fmt.Errorf("could not parse metric: %s", item)
		}
	}
	return dbMetrics, nil
}
