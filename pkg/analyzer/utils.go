package analyzer

import (
	"time"

	"github.com/sirupsen/logrus"
)

const (
	ValidatorSetSize = 500000 // Estimation of current number of validators, used for channel length declaration
	maxWorkers       = 50
	minReqTime       = 100 * time.Millisecond // max 10 queries per second, dont spam beacon node
)

var (
	log = logrus.WithField(
		"module", "analyzer",
	)
)
