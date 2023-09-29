package spec

import (
	"fmt"
	"time"

	"github.com/cortze/eth-cl-state-analyzer/pkg/utils"

	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/altair"
	"github.com/attestantio/go-eth2-client/spec/bellatrix"
	"github.com/attestantio/go-eth2-client/spec/capella"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/sirupsen/logrus"
)

// This Wrapper is meant to include all common objects across Ethereum Hard Fork Specs
type AgnosticBlock struct {
	Slot              phase0.Slot
	StateRoot         phase0.Root
	ProposerIndex     phase0.ValidatorIndex
	Graffiti          [32]byte
	Proposed          bool
	Attestations      []*phase0.Attestation
	Deposits          []*phase0.Deposit
	ProposerSlashings []*phase0.ProposerSlashing
	AttesterSlashings []*phase0.AttesterSlashing
	VoluntaryExits    []*phase0.SignedVoluntaryExit
	SyncAggregate     *altair.SyncAggregate
	ExecutionPayload  AgnosticExecutionPayload
	Reward            BlockRewards
	SSZsize           uint32
	SnappySize        uint32
	CompressionTime   time.Duration
	DecompressionTime time.Duration
}

// This Wrapper is meant to include all common objects across Ethereum Hard Fork Specs
type AgnosticExecutionPayload struct {
	FeeRecipient  bellatrix.ExecutionAddress
	GasLimit      uint64
	GasUsed       uint64
	Timestamp     uint64
	BaseFeePerGas [32]byte
	BlockHash     phase0.Hash32
	Transactions  []bellatrix.Transaction
	BlockNumber   uint64
	Withdrawals   []*capella.Withdrawal
	PayloadSize   uint32
}

func (f AgnosticBlock) Type() ModelType {
	return BlockModel
}

func (p AgnosticExecutionPayload) BaseFeeToInt() int {
	return 0 // not implemented yet
}

func GetCustomBlock(block spec.VersionedSignedBeaconBlock) (AgnosticBlock, error) {
	switch block.Version {
	case spec.DataVersionPhase0:
		return NewPhase0Block(block), nil
	case spec.DataVersionAltair:
		return NewAltairBlock(block), nil
	case spec.DataVersionBellatrix:
		return NewBellatrixBlock(block), nil
	case spec.DataVersionCapella:
		return NewCapellaBlock(block), nil
	case spec.DataVersionDeneb:
		return NewDenebBlock(block), nil
	default:
		return AgnosticBlock{}, fmt.Errorf("could not figure out the Beacon Block Fork Version: %s", block.Version)
	}
}

func NewPhase0Block(block spec.VersionedSignedBeaconBlock) AgnosticBlock {
	// make the compression of the block
	compressionMetrics, err := utils.CompressConsensusSignedBlock(block.Phase0)
	if err != nil {
		logrus.Errorf("unable to compress phase0 block %d - %s", block.Phase0.Message.Slot, err.Error())
	}
	return AgnosticBlock{
		Slot:              block.Phase0.Message.Slot,
		ProposerIndex:     block.Phase0.Message.ProposerIndex,
		Graffiti:          block.Phase0.Message.Body.Graffiti,
		Proposed:          true,
		Attestations:      block.Phase0.Message.Body.Attestations,
		Deposits:          block.Phase0.Message.Body.Deposits,
		ProposerSlashings: block.Phase0.Message.Body.ProposerSlashings,
		AttesterSlashings: block.Phase0.Message.Body.AttesterSlashings,
		VoluntaryExits:    block.Phase0.Message.Body.VoluntaryExits,
		SyncAggregate:     &altair.SyncAggregate{},
		ExecutionPayload: AgnosticExecutionPayload{
			FeeRecipient:  bellatrix.ExecutionAddress{},
			GasLimit:      0,
			GasUsed:       0,
			Timestamp:     0,
			BaseFeePerGas: [32]byte{},
			BlockHash:     phase0.Hash32{},
			Transactions:  make([]bellatrix.Transaction, 0),
			BlockNumber:   0,
			Withdrawals:   make([]*capella.Withdrawal, 0),
			PayloadSize:   uint32(0),
		}, // snappy
		SSZsize:           compressionMetrics.SSZsize,
		SnappySize:        compressionMetrics.SnappySize,
		CompressionTime:   compressionMetrics.CompressionTime,
		DecompressionTime: compressionMetrics.DecompressionTime,
	}
}

func NewAltairBlock(block spec.VersionedSignedBeaconBlock) AgnosticBlock {
	// make the compression of the block
	compressionMetrics, err := utils.CompressConsensusSignedBlock(block.Altair)
	if err != nil {
		logrus.Errorf("unable to compress altair block %d - %s", block.Altair.Message.Slot, err.Error())
	}
	return AgnosticBlock{
		Slot:              block.Altair.Message.Slot,
		ProposerIndex:     block.Altair.Message.ProposerIndex,
		Graffiti:          block.Altair.Message.Body.Graffiti,
		Proposed:          true,
		Attestations:      block.Altair.Message.Body.Attestations,
		Deposits:          block.Altair.Message.Body.Deposits,
		ProposerSlashings: block.Altair.Message.Body.ProposerSlashings,
		AttesterSlashings: block.Altair.Message.Body.AttesterSlashings,
		VoluntaryExits:    block.Altair.Message.Body.VoluntaryExits,
		SyncAggregate:     block.Altair.Message.Body.SyncAggregate,
		ExecutionPayload: AgnosticExecutionPayload{
			FeeRecipient:  bellatrix.ExecutionAddress{},
			GasLimit:      0,
			GasUsed:       0,
			Timestamp:     0,
			BaseFeePerGas: [32]byte{},
			BlockHash:     phase0.Hash32{},
			Transactions:  make([]bellatrix.Transaction, 0),
			BlockNumber:   0,
			Withdrawals:   make([]*capella.Withdrawal, 0),
			PayloadSize:   uint32(0),
		}, // snappy
		SSZsize:           compressionMetrics.SSZsize,
		SnappySize:        compressionMetrics.SnappySize,
		CompressionTime:   compressionMetrics.CompressionTime,
		DecompressionTime: compressionMetrics.DecompressionTime,
	}
}

func NewBellatrixBlock(block spec.VersionedSignedBeaconBlock) AgnosticBlock {
	// make the compression of the block
	compressionMetrics, err := utils.CompressConsensusSignedBlock(block.Bellatrix)
	if err != nil {
		logrus.Errorf("unable to compress bellatrix block %d - %s", block.Bellatrix.Message.Slot, err.Error())
	}
	return AgnosticBlock{
		Slot:              block.Bellatrix.Message.Slot,
		ProposerIndex:     block.Bellatrix.Message.ProposerIndex,
		Graffiti:          block.Bellatrix.Message.Body.Graffiti,
		Proposed:          true,
		Attestations:      block.Bellatrix.Message.Body.Attestations,
		Deposits:          block.Bellatrix.Message.Body.Deposits,
		ProposerSlashings: block.Bellatrix.Message.Body.ProposerSlashings,
		AttesterSlashings: block.Bellatrix.Message.Body.AttesterSlashings,
		VoluntaryExits:    block.Bellatrix.Message.Body.VoluntaryExits,
		SyncAggregate:     block.Bellatrix.Message.Body.SyncAggregate,
		ExecutionPayload: AgnosticExecutionPayload{
			FeeRecipient:  block.Bellatrix.Message.Body.ExecutionPayload.FeeRecipient,
			GasLimit:      block.Bellatrix.Message.Body.ExecutionPayload.GasLimit,
			GasUsed:       block.Bellatrix.Message.Body.ExecutionPayload.GasUsed,
			Timestamp:     block.Bellatrix.Message.Body.ExecutionPayload.Timestamp,
			BaseFeePerGas: block.Bellatrix.Message.Body.ExecutionPayload.BaseFeePerGas,
			BlockHash:     block.Bellatrix.Message.Body.ExecutionPayload.BlockHash,
			Transactions:  block.Bellatrix.Message.Body.ExecutionPayload.Transactions,
			BlockNumber:   block.Bellatrix.Message.Body.ExecutionPayload.BlockNumber,
			Withdrawals:   make([]*capella.Withdrawal, 0),
			PayloadSize:   uint32(0),
		}, // snappy
		SSZsize:           compressionMetrics.SSZsize,
		SnappySize:        compressionMetrics.SnappySize,
		CompressionTime:   compressionMetrics.CompressionTime,
		DecompressionTime: compressionMetrics.DecompressionTime,
	}
}

func NewCapellaBlock(block spec.VersionedSignedBeaconBlock) AgnosticBlock {
	// make the compression of the block
	compressionMetrics, err := utils.CompressConsensusSignedBlock(block.Capella)
	if err != nil {
		logrus.Errorf("unable to compress capella block %d - %s", block.Capella.Message.Slot, err.Error())
	}
	return AgnosticBlock{
		Slot:              block.Capella.Message.Slot,
		ProposerIndex:     block.Capella.Message.ProposerIndex,
		Graffiti:          block.Capella.Message.Body.Graffiti,
		Proposed:          true,
		Attestations:      block.Capella.Message.Body.Attestations,
		Deposits:          block.Capella.Message.Body.Deposits,
		ProposerSlashings: block.Capella.Message.Body.ProposerSlashings,
		AttesterSlashings: block.Capella.Message.Body.AttesterSlashings,
		VoluntaryExits:    block.Capella.Message.Body.VoluntaryExits,
		SyncAggregate:     block.Capella.Message.Body.SyncAggregate,
		ExecutionPayload: AgnosticExecutionPayload{
			FeeRecipient:  block.Capella.Message.Body.ExecutionPayload.FeeRecipient,
			GasLimit:      block.Capella.Message.Body.ExecutionPayload.GasLimit,
			GasUsed:       block.Capella.Message.Body.ExecutionPayload.GasUsed,
			Timestamp:     block.Capella.Message.Body.ExecutionPayload.Timestamp,
			BaseFeePerGas: block.Capella.Message.Body.ExecutionPayload.BaseFeePerGas,
			BlockHash:     block.Capella.Message.Body.ExecutionPayload.BlockHash,
			Transactions:  block.Capella.Message.Body.ExecutionPayload.Transactions,
			BlockNumber:   block.Capella.Message.Body.ExecutionPayload.BlockNumber,
			Withdrawals:   block.Capella.Message.Body.ExecutionPayload.Withdrawals,
			PayloadSize:   uint32(0),
		}, // snappy
		SSZsize:           compressionMetrics.SSZsize,
		SnappySize:        compressionMetrics.SnappySize,
		CompressionTime:   compressionMetrics.CompressionTime,
		DecompressionTime: compressionMetrics.DecompressionTime,
	}
}

func NewDenebBlock(block spec.VersionedSignedBeaconBlock) AgnosticBlock {
	// make the compression of the block
	compressionMetrics, err := utils.CompressConsensusSignedBlock(block.Deneb)
	if err != nil {
		logrus.Errorf("unable to compress deneb block %d - %s", block.Deneb.Message.Slot, err.Error())
	}
	return AgnosticBlock{
		Slot:              block.Deneb.Message.Slot,
		ProposerIndex:     block.Deneb.Message.ProposerIndex,
		Graffiti:          block.Deneb.Message.Body.Graffiti,
		Proposed:          true,
		Attestations:      block.Deneb.Message.Body.Attestations,
		Deposits:          block.Deneb.Message.Body.Deposits,
		ProposerSlashings: block.Deneb.Message.Body.ProposerSlashings,
		AttesterSlashings: block.Deneb.Message.Body.AttesterSlashings,
		VoluntaryExits:    block.Deneb.Message.Body.VoluntaryExits,
		SyncAggregate:     block.Deneb.Message.Body.SyncAggregate,
		ExecutionPayload: AgnosticExecutionPayload{
			FeeRecipient:  block.Deneb.Message.Body.ExecutionPayload.FeeRecipient,
			GasLimit:      block.Deneb.Message.Body.ExecutionPayload.GasLimit,
			GasUsed:       block.Deneb.Message.Body.ExecutionPayload.GasUsed,
			Timestamp:     block.Deneb.Message.Body.ExecutionPayload.Timestamp,
			BaseFeePerGas: block.Deneb.Message.Body.ExecutionPayload.BaseFeePerGas.Bytes32(),
			BlockHash:     block.Deneb.Message.Body.ExecutionPayload.BlockHash,
			Transactions:  block.Deneb.Message.Body.ExecutionPayload.Transactions,
			BlockNumber:   block.Deneb.Message.Body.ExecutionPayload.BlockNumber,
			Withdrawals:   block.Deneb.Message.Body.ExecutionPayload.Withdrawals,
			PayloadSize:   uint32(0),
		}, // snappy
		SSZsize:           compressionMetrics.SSZsize,
		SnappySize:        compressionMetrics.SnappySize,
		CompressionTime:   compressionMetrics.CompressionTime,
		DecompressionTime: compressionMetrics.DecompressionTime,
	}
}
