package spec

import (
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/sirupsen/logrus"
)

var (
	log = logrus.WithField(
		"module", "spec",
	)
)

func IsActive(validator phase0.Validator, epoch phase0.Epoch) bool {
	if validator.ActivationEpoch <= epoch &&
		epoch < validator.ExitEpoch {
		return true
	}
	return false
}
