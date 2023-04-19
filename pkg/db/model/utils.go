package model

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

type Model interface { // simply to enforce a Model interface
	// For now we simply support insert operations
	Type() ModelType // whether insert is activated for this model
}
