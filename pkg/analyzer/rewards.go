package analyzer

import (
	"github.com/cortze/eth2-state-analyzer/pkg/custom_spec"
)

const (
// participationRate   = 0.945 // about to calculate participation rate
)

func GetValidatorBalance(customBState custom_spec.CustomBeaconState, valIdx uint64) (uint64, error) {

	balance, err := customBState.Balance(valIdx)

	if err != nil {
		return 0, err
	}

	return balance, nil
}
