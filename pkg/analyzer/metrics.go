package analyzer

import (
	"fmt"
	"strings"
)

type DBMetrics struct {
	Block          bool
	Epoch          bool
	PoolSummary    bool
	ProposerDuties bool
	Validator      bool
	Withdrawals    bool
	Transaction    bool

	StateDownload bool
}

func NewMetrics(input string) (DBMetrics, error) {
	dbMetrics := DBMetrics{}

	for _, item := range strings.Split(input, ",") {

		switch item {
		case "block":
			dbMetrics.Block = true
		case "epoch":
			dbMetrics.Epoch = true
			dbMetrics.StateDownload = true
		case "pool_summary":
			dbMetrics.PoolSummary = true
			dbMetrics.StateDownload = true
		case "validator":
			dbMetrics.Validator = true
			dbMetrics.StateDownload = true
		case "withdrawal":
			dbMetrics.Withdrawals = true
		case "transaction":
			dbMetrics.Withdrawals = true
		default:
			return DBMetrics{}, fmt.Errorf("could not parse metric: %s", item)
		}
	}
	return dbMetrics, nil
}
