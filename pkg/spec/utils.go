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

// https://github.com/ethereum/consensus-specs/blob/dev/specs/phase0/beacon-chain.md#is_active_validator
func IsActive(validator phase0.Validator, epoch phase0.Epoch) bool {
	if validator.ActivationEpoch <= epoch &&
		epoch < validator.ExitEpoch {
		return true
	}
	return false
}
