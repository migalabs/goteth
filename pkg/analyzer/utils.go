package analyzer

import (
	"time"

	"github.com/sirupsen/logrus"
)

const (
	ValidatorSetSize = 500000 // Estimation of current number of validators, used for channel length declaration
	maxWorkers       = 50
	minReqTime       = 10 * time.Second
)

var (
	log = logrus.WithField(
		"module", "analyzer",
	)
)
