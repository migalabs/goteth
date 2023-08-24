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

type BlockRewards struct {
	ExecutionOptimistic bool                `json:"execution_optimistic"`
	Data                BlockRewardsContent `json:"data"`
}

type BlockRewardsContent struct {
	ProposerIndex     phase0.ValidatorIndex `json:"proposer_index,string"`
	Total             phase0.Gwei           `json:"total,string"`
	Attestations      phase0.Gwei           `json:"attestations,string"`
	SyncAggregate     phase0.Gwei           `json:"sync_aggregate,string"`
	ProposerSlashings phase0.Gwei           `json:"proposer_slashings,string"`
	AttesterSlashings phase0.Gwei           `json:"attester_slashings,string"`
}
