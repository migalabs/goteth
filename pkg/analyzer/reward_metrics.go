package analyzer

import (
	"errors"
	"sync"

	api "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/phase0"

	"github.com/cortze/eth2-state-analyzer/pkg/utils"
)

type RewardMetrics struct {
	m                 sync.Mutex
	baseslot          uint64
	innerCnt          uint64
	validatorIdx      uint64
	ValidatorBalances []uint64  // Gwei ?¿
	MaxRewards        []uint64  // Gwei ?¿
	Rewards           []uint64  // Gweis ?¿
	RewardPercentage  []float64 // %

	MissingSources []uint64
	MissingHeads   []uint64
	MissingTargets []uint64
}

func NewRewardMetrics(initslot uint64, epochRange uint64, validatorIdx uint64) (*RewardMetrics, error) {
	log.Debugf("len of the epoch range %d", epochRange)
	return &RewardMetrics{
		baseslot:          initslot,
		validatorIdx:      validatorIdx,
		ValidatorBalances: make([]uint64, epochRange),
		MaxRewards:        make([]uint64, epochRange),
		Rewards:           make([]uint64, epochRange),
		RewardPercentage:  make([]float64, epochRange),
		MissingSources:    make([]uint64, epochRange),
		MissingHeads:      make([]uint64, epochRange),
		MissingTargets:    make([]uint64, epochRange),
	}, nil
}

// Supposed to be
func (m *RewardMetrics) CalculateEpochPerformance(bState *spec.VersionedBeaconState, validators *map[phase0.ValidatorIndex]*api.Validator, totalActiveBalance uint64) error {

	validatorBalance, err := GetValidatorBalance(bState, m.validatorIdx)
	if err != nil {
		return err
	}
	// add always the balance to the array
	log.Debugf("starting performance calculation of validator %d", m.validatorIdx)
	m.m.Lock()
	defer func() {
		log.Debugf("finishing the performance calculation of validator %d", m.validatorIdx)
		m.m.Unlock()
	}()
	m.ValidatorBalances[m.innerCnt] = uint64(validatorBalance)

	if m.innerCnt != 0 {
		// calculate Reward from the previous Balance
		reward := validatorBalance - m.ValidatorBalances[m.innerCnt-1]
		log.Debugf("reward for validator %d = %d", m.validatorIdx, reward)

		// Add Reward
		m.Rewards[m.innerCnt] = reward

		// Proccess Max-Rewards
		maxReward, err := GetMaxReward(m.validatorIdx, validators, totalActiveBalance)
		if err != nil {
			return err
		}
		log.Debugf("max reward for validator %d = %d", m.validatorIdx, maxReward)
		m.MaxRewards[m.innerCnt] = maxReward

		// Proccess Reward-Performance-Ratio
		rewardPerf := (float64(reward) * 100) / float64(maxReward)
		m.RewardPercentage[m.innerCnt] = rewardPerf
		log.Debugf("reward performance for %d = %f%", m.validatorIdx, rewardPerf)

		// TODO:
		// Add number of missing sources

		// Add number of missing head

		// Add number of missing targets

	}
	// do not forget to increment the internal counter
	m.innerCnt++
	log.Debugf("done with validator %d", m.validatorIdx)

	return nil
}

func (m *RewardMetrics) GetEpochMetrics(slot uint64) (SingleEpochMetrics, error) {
	var epochMetrics SingleEpochMetrics

	// calculate the index
	idx := m.GetIndexFromslot(slot)
	if idx < 0 {
		log.Errorf("requested metrics for slot: %d couldn't be found. Max slot is %d", slot, m.baseslot+(32*uint64(len(m.Rewards))))
		return epochMetrics, errors.New("requested slot can not be found on the analyzed data")
	}

	m.m.Lock()
	// if the index is okey, compose the Singe epoch metrics
	epochMetrics.ValidatorIdx = m.validatorIdx
	epochMetrics.Slot = m.validatorIdx
	epochMetrics.Epoch = utils.GetEpochFromSlot(slot)
	epochMetrics.ValidatorBalance = m.ValidatorBalances[idx]
	epochMetrics.MaxReward = m.MaxRewards[idx]
	epochMetrics.Reward = m.Rewards[idx]
	epochMetrics.RewardPercentage = m.RewardPercentage[idx]
	epochMetrics.MissingSource = m.MissingSources[idx]
	epochMetrics.MissingHead = m.MissingHeads[idx]
	epochMetrics.MissingTarget = m.MissingTargets[idx]
	epochMetrics.ValidatorIdx = m.validatorIdx
	m.m.Unlock()
	return epochMetrics, nil
}

func (m *RewardMetrics) GetIndexFromslot(slot uint64) int {
	idx := -1
	m.m.Lock()
	defer m.m.Unlock()
	idx = int(slot/m.baseslot) - 1
	if idx >= len(m.Rewards) {
		return -1
	}

	return idx
}

func (m *RewardMetrics) AddslotPerformance() error {

	return nil
}

type SingleEpochMetrics struct {
	ValidatorIdx     uint64
	Slot             uint64
	Epoch            uint64
	ValidatorBalance uint64  // Gwei ?¿
	MaxReward        uint64  // Gwei ?¿
	Reward           uint64  // Gweis ?¿
	RewardPercentage float64 // %

	MissingSource uint64
	MissingHead   uint64
	MissingTarget uint64
}
