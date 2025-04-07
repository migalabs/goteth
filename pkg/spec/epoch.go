package spec

import (
	"github.com/attestantio/go-eth2-client/spec/phase0"
)

type Epoch struct {
	Epoch                         phase0.Epoch
	Slot                          phase0.Slot
	NumAttestations               int
	NumAttValidators              int
	NumValidators                 int
	TotalBalance                  float32
	AttEffectiveBalance           phase0.Gwei
	SourceAttEffectiveBalance     phase0.Gwei
	TargetAttEffectiveBalance     phase0.Gwei
	HeadAttEffectiveBalance       phase0.Gwei
	TotalEffectiveBalance         phase0.Gwei
	MissingSource                 int
	MissingTarget                 int
	MissingHead                   int
	Timestamp                     int64
	NumSlashedVals                int
	NumActiveVals                 int
	NumExitedVals                 int
	NumInActivationVals           int
	SyncCommitteeParticipation    uint64
	DepositsNum                   int
	TotalDepositsAmount           phase0.Gwei
	WithdrawalsNum                int
	TotalWithdrawalsAmount        phase0.Gwei
	NewProposerSlashings          int
	NewAttesterSlashings          int
	ConsolidationRequestsNum      int
	WithdrawalRequestsNum         int
	ConsolidationsProcessedNum    uint64
	ConsolidationsProcessedAmount phase0.Gwei
}

func (f Epoch) Type() ModelType {
	return EpochModel
}
