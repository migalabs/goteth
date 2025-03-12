package spec

import (
	"github.com/attestantio/go-eth2-client/spec/bellatrix"
	"github.com/attestantio/go-eth2-client/spec/phase0"
)

type ConsolidationRequestResult uint8

const (
	ConsolidationRequestResultUnknown ConsolidationRequestResult = 0
	ConsolidationRequestResultSuccess ConsolidationRequestResult = 1

	// global errors
	ConsolidationRequestResultTotalBalanceTooLow ConsolidationRequestResult = 10
	ConsolidationRequestResultQueueFull          ConsolidationRequestResult = 11
	ConsolidationRequestResultRequestUsedAsExit  ConsolidationRequestResult = 12

	// source validator errors
	ConsolidationRequestResultSrcNotFound             ConsolidationRequestResult = 20
	ConsolidationRequestResultSrcInvalidCredentials   ConsolidationRequestResult = 21
	ConsolidationRequestResultSrcInvalidSender        ConsolidationRequestResult = 22
	ConsolidationRequestResultSrcNotActive            ConsolidationRequestResult = 23
	ConsolidationRequestResultSrcNotOldEnough         ConsolidationRequestResult = 24
	ConsolidationRequestResultSrcHasPendingWithdrawal ConsolidationRequestResult = 25
	ConsolidationRequestResultSrcExitAlreadyInitiated ConsolidationRequestResult = 26

	// target validator errors
	ConsolidationRequestResultTgtNotFound             ConsolidationRequestResult = 30
	ConsolidationRequestResultTgtInvalidCredentials   ConsolidationRequestResult = 31
	ConsolidationRequestResultTgtInvalidSender        ConsolidationRequestResult = 32
	ConsolidationRequestResultTgtNotCompounding       ConsolidationRequestResult = 33
	ConsolidationRequestResultTgtNotActive            ConsolidationRequestResult = 34
	ConsolidationRequestResultTgtExitAlreadyInitiated ConsolidationRequestResult = 35
)

type ConsolidationRequest struct {
	Slot          phase0.Slot
	Index         uint64
	SourceAddress bellatrix.ExecutionAddress
	SourcePubkey  phase0.BLSPubKey
	TargetPubkey  phase0.BLSPubKey
	Result        ConsolidationRequestResult
}

func (f ConsolidationRequest) Type() ModelType {
	return ConsolidationRequestModel
}

func (f ConsolidationRequest) ToArray() []interface{} {
	rows := []interface{}{
		f.Slot,
		f.Index,
		f.SourceAddress,
		f.SourcePubkey,
		f.TargetPubkey,
	}
	return rows
}
