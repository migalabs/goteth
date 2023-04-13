package model

import (
	"github.com/attestantio/go-eth2-client/spec/phase0"
)

type PoolSummary struct {
	PoolName      string
	Epoch         phase0.Epoch
	ValidatorList []ValidatorRewards
}

func (p *PoolSummary) AddValidator(input ValidatorRewards) {
	p.ValidatorList = append(p.ValidatorList, input)
}

func (f PoolSummary) InsertOp() bool {
	return true
}

func (f PoolSummary) DropOp() bool {
	return false
}
