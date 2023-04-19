package spec

/*
Phase0
*/

const (
	MaxEffectiveInc        = 32
	BaseRewardFactor       = 64
	BaseRewardPerEpoch     = 4
	EffectiveBalanceInc    = 1000000000
	SlotsPerEpoch          = 32
	ProposerRewardQuotient = 8
	SlotsPerHistoricalRoot = 8192
	SlotSeconds            = 12
	EpochSlots             = 32
)

/*
Altair
*/
const (
	// spec weight constants
	TimelySourceWeight = 14
	TimelyTargetWeight = 26
	TimelyHeadWeight   = 14

	SyncRewardWeight  = 2
	ProposerWeight    = 8
	WeightDenominator = 64
	SyncCommitteeSize = 512
)

var (
	ParticipatingFlagsWeight = [3]int{TimelySourceWeight, TimelyTargetWeight, TimelyHeadWeight}
)

type ModelType int8

const (
	BlockModel ModelType = iota
	EpochModel
	PoolSummaryModel
	ProposerDutyModel
	ValidatorLastStatusModel
	ValidatorRewardsModel
	WithdrawalModel
)

type ValidatorStatus int8

const (
	QUEUE_STATUS ValidatorStatus = iota
	ACTIVE_STATUS
	EXIT_STATUS
	SLASHED_STATUS
)
