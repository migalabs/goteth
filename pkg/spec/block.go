package spec

import (
	"encoding/binary"
	"fmt"
	"time"

	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/altair"
	"github.com/attestantio/go-eth2-client/spec/bellatrix"
	"github.com/attestantio/go-eth2-client/spec/capella"
	"github.com/attestantio/go-eth2-client/spec/electra"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/migalabs/goteth/pkg/utils"
	"github.com/sirupsen/logrus"
)

// This Wrapper is meant to include all common objects across Ethereum Hard Fork Specs
type AgnosticBlock struct {
	HardForkVersion       spec.DataVersion
	Slot                  phase0.Slot
	StateRoot             phase0.Root
	Root                  phase0.Root
	ParentRoot            phase0.Root
	ProposerIndex         phase0.ValidatorIndex
	Graffiti              [32]byte
	Proposed              bool
	Attestations          []*phase0.Attestation // For electra blocks, Attestations is nil
	VotesIncluded         uint64
	NewVotesIncluded      uint64
	Deposits              []*phase0.Deposit
	ProposerSlashings     []*phase0.ProposerSlashing
	AttesterSlashings     []*phase0.AttesterSlashing // For electra blocks, AttesterSlashings is nil
	VoluntaryExits        []*phase0.SignedVoluntaryExit
	SyncAggregate         *altair.SyncAggregate
	ExecutionPayload      AgnosticExecutionPayload
	BLSToExecutionChanges []*capella.SignedBLSToExecutionChange
	Reward                BlockRewards
	SSZsize               uint32
	SnappySize            uint32
	CompressionTime       time.Duration
	DecompressionTime     time.Duration
	ManualReward          phase0.Gwei
	// Electra
	ElectraAttestations      []*electra.Attestation
	ElectraAttesterSlashings []*electra.AttesterSlashing
	ExecutionRequests        *electra.ExecutionRequests
}

// This Wrapper is meant to include all common objects across Ethereum Hard Fork Specs
type AgnosticExecutionPayload struct {
	FeeRecipient         bellatrix.ExecutionAddress
	GasLimit             uint64
	GasUsed              uint64
	Timestamp            uint64
	BaseFeePerGas        uint64
	BlockHash            phase0.Hash32
	Transactions         []bellatrix.Transaction
	AgnosticTransactions []AgnosticTransaction
	BlockNumber          uint64
	Withdrawals          []*capella.Withdrawal
	PayloadSize          uint32
}

func (f AgnosticBlock) Type() ModelType {
	return BlockModel
}

func (p AgnosticBlock) BlockGasFees() (uint64, uint64, error) {
	reward := uint64(0)
	burn := uint64(0)
	baseFeePerGas := p.ExecutionPayload.BaseFeePerGas

	if len(p.ExecutionPayload.AgnosticTransactions) == 0 {
		return reward, burn, fmt.Errorf("cannot calculate block reward: no transactions appended")
	}

	for _, tx := range p.ExecutionPayload.AgnosticTransactions {
		priorityFee := (tx.GasPrice - baseFeePerGas) * tx.Gas
		baseFee := baseFeePerGas * uint64(tx.Gas)
		reward += priorityFee
		burn += baseFee
	}

	return reward, burn, nil

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
	case spec.DataVersionElectra:
		return NewElectraBlock(block), nil
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

	root, err := block.Root()
	if err != nil {
		log.Fatalf("could not read root from block %d", block.Phase0.Message.Slot)
	}

	return AgnosticBlock{
		HardForkVersion:   spec.DataVersionPhase0,
		Slot:              block.Phase0.Message.Slot,
		ParentRoot:        block.Phase0.Message.ParentRoot,
		Root:              root,
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
			BaseFeePerGas: 0,
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
	root, err := block.Root()
	if err != nil {
		log.Fatalf("could not read root from block %d", block.Altair.Message.Slot)
	}
	return AgnosticBlock{
		HardForkVersion:   spec.DataVersionAltair,
		Slot:              block.Altair.Message.Slot,
		Root:              root,
		ParentRoot:        block.Altair.Message.ParentRoot,
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
			BaseFeePerGas: 0,
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
	root, err := block.Root()
	if err != nil {
		log.Fatalf("could not read root from block %d", block.Bellatrix.Message.Slot)
	}
	return AgnosticBlock{
		HardForkVersion:   spec.DataVersionBellatrix,
		Slot:              block.Bellatrix.Message.Slot,
		Root:              root,
		ParentRoot:        block.Bellatrix.Message.ParentRoot,
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
			BaseFeePerGas: binary.BigEndian.Uint64(block.Bellatrix.Message.Body.ExecutionPayload.BaseFeePerGas[:]),
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
	root, err := block.Root()
	if err != nil {
		log.Fatalf("could not read root from block %d", block.Capella.Message.Slot)
	}
	return AgnosticBlock{
		HardForkVersion:   spec.DataVersionCapella,
		Slot:              block.Capella.Message.Slot,
		Root:              root,
		ParentRoot:        block.Capella.Message.ParentRoot,
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
			BaseFeePerGas: binary.BigEndian.Uint64(block.Capella.Message.Body.ExecutionPayload.BaseFeePerGas[:]),
			BlockHash:     block.Capella.Message.Body.ExecutionPayload.BlockHash,
			Transactions:  block.Capella.Message.Body.ExecutionPayload.Transactions,
			BlockNumber:   block.Capella.Message.Body.ExecutionPayload.BlockNumber,
			Withdrawals:   block.Capella.Message.Body.ExecutionPayload.Withdrawals,
			PayloadSize:   uint32(0),
		}, // snappy
		BLSToExecutionChanges: block.Capella.Message.Body.BLSToExecutionChanges,
		SSZsize:               compressionMetrics.SSZsize,
		SnappySize:            compressionMetrics.SnappySize,
		CompressionTime:       compressionMetrics.CompressionTime,
		DecompressionTime:     compressionMetrics.DecompressionTime,
	}
}

func NewDenebBlock(block spec.VersionedSignedBeaconBlock) AgnosticBlock {
	// make the compression of the block
	compressionMetrics, err := utils.CompressConsensusSignedBlock(block.Deneb)
	if err != nil {
		logrus.Errorf("unable to compress deneb block %d - %s", block.Deneb.Message.Slot, err.Error())
	}
	root, err := block.Root()
	if err != nil {
		log.Fatalf("could not read root from block %d", block.Deneb.Message.Slot)
	}
	return AgnosticBlock{
		HardForkVersion:   spec.DataVersionDeneb,
		Slot:              block.Deneb.Message.Slot,
		Root:              root,
		ParentRoot:        block.Deneb.Message.ParentRoot,
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
			BaseFeePerGas: block.Deneb.Message.Body.ExecutionPayload.BaseFeePerGas.Uint64(),
			BlockHash:     block.Deneb.Message.Body.ExecutionPayload.BlockHash,
			Transactions:  block.Deneb.Message.Body.ExecutionPayload.Transactions,
			BlockNumber:   block.Deneb.Message.Body.ExecutionPayload.BlockNumber,
			Withdrawals:   block.Deneb.Message.Body.ExecutionPayload.Withdrawals,
			PayloadSize:   uint32(0),
		}, // snappy
		BLSToExecutionChanges: block.Deneb.Message.Body.BLSToExecutionChanges,
		SSZsize:               compressionMetrics.SSZsize,
		SnappySize:            compressionMetrics.SnappySize,
		CompressionTime:       compressionMetrics.CompressionTime,
		DecompressionTime:     compressionMetrics.DecompressionTime,
	}
}

func NewElectraBlock(block spec.VersionedSignedBeaconBlock) AgnosticBlock {
	// make the compression of the block
	compressionMetrics, err := utils.CompressConsensusSignedBlock(block.Electra)
	if err != nil {
		logrus.Errorf("unable to compress deneb block %d - %s", block.Electra.Message.Slot, err.Error())
	}
	root, err := block.Root()
	if err != nil {
		log.Fatalf("could not read root from block %d", block.Electra.Message.Slot)
	}
	return AgnosticBlock{
		HardForkVersion:          spec.DataVersionElectra,
		Slot:                     block.Electra.Message.Slot,
		Root:                     root,
		ParentRoot:               block.Electra.Message.ParentRoot,
		ProposerIndex:            block.Electra.Message.ProposerIndex,
		Graffiti:                 block.Electra.Message.Body.Graffiti,
		Proposed:                 true,
		Attestations:             nil,
		ElectraAttestations:      block.Electra.Message.Body.Attestations,
		Deposits:                 block.Electra.Message.Body.Deposits,
		ProposerSlashings:        block.Electra.Message.Body.ProposerSlashings,
		AttesterSlashings:        nil,
		ElectraAttesterSlashings: block.Electra.Message.Body.AttesterSlashings,
		VoluntaryExits:           block.Electra.Message.Body.VoluntaryExits,
		SyncAggregate:            block.Electra.Message.Body.SyncAggregate,
		ExecutionPayload: AgnosticExecutionPayload{
			FeeRecipient:  block.Electra.Message.Body.ExecutionPayload.FeeRecipient,
			GasLimit:      block.Electra.Message.Body.ExecutionPayload.GasLimit,
			GasUsed:       block.Electra.Message.Body.ExecutionPayload.GasUsed,
			Timestamp:     block.Electra.Message.Body.ExecutionPayload.Timestamp,
			BaseFeePerGas: block.Electra.Message.Body.ExecutionPayload.BaseFeePerGas.Uint64(),
			BlockHash:     block.Electra.Message.Body.ExecutionPayload.BlockHash,
			Transactions:  block.Electra.Message.Body.ExecutionPayload.Transactions,
			BlockNumber:   block.Electra.Message.Body.ExecutionPayload.BlockNumber,
			Withdrawals:   block.Electra.Message.Body.ExecutionPayload.Withdrawals,
			PayloadSize:   uint32(0),
		}, // snappy
		BLSToExecutionChanges: block.Electra.Message.Body.BLSToExecutionChanges,
		SSZsize:               compressionMetrics.SSZsize,
		SnappySize:            compressionMetrics.SnappySize,
		CompressionTime:       compressionMetrics.CompressionTime,
		DecompressionTime:     compressionMetrics.DecompressionTime,
		ExecutionRequests:     block.Electra.Message.Body.ExecutionRequests,
	}
}
