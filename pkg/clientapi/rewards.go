package clientapi

import (
	"fmt"

	"github.com/attestantio/go-eth2-client/api"
	v1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec/phase0"
)

func (s *APIClient) RequestBlockRewards(slot phase0.Slot) (v1.BlockRewards, error) {

	rewards, err := s.Api.BlockRewards(s.ctx,
		&api.BlockRewardsOpts{
			Block: fmt.Sprintf("%d", slot),
		})
	if err != nil {
		return v1.BlockRewards{}, fmt.Errorf("unable to retrieve Block Rewards at slot %d: %s", slot, err.Error())
	}

	return *rewards.Data, err
}
