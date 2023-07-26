package clientapi

import (
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/attestantio/go-eth2-client/spec/altair"
	"github.com/attestantio/go-eth2-client/spec/bellatrix"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/cortze/eth-cl-state-analyzer/pkg/spec"
	"github.com/prysmaticlabs/go-bitfield"
)

func (s APIClient) RequestBeaconBlock(slot phase0.Slot) (spec.AgnosticBlock, bool, error) {
	startTime := time.Now()
	newBlock, err := s.Api.SignedBeaconBlock(s.ctx, fmt.Sprintf("%d", slot))

	if newBlock == nil {
		log.Warnf("the beacon block at slot %d does not exist, missing block", slot)
		return s.CreateMissingBlock(slot), false, nil
	}
	if err != nil {
		// close the channel (to tell other routines to stop processing and end)
		return spec.AgnosticBlock{}, false, fmt.Errorf("unable to retrieve Beacon Block at slot %d: %s", slot, err.Error())
	}

	log.Infof("block at slot %d downloaded in %f seconds", slot, time.Since(startTime).Seconds())
	customBlock, err := spec.GetCustomBlock(*newBlock)

	if err != nil {
		// close the channel (to tell other routines to stop processing and end)
		return spec.AgnosticBlock{}, false, fmt.Errorf("unable to parse Beacon Block at slot %d: %s", slot, err.Error())
	}

	// fill in block size on custom block using RequestBlockByHash
	block, err := s.RequestBlockByHash(common.Hash(customBlock.ExecutionPayload.BlockHash))
	if err != nil {
		log.Error("cannot request block by hash: %s", err)
	}
	if block != nil {
		customBlock.Size = uint32(block.Size())
	}

	customBlock.StateRoot = s.RequestStateRoot(slot)
	return customBlock, true, nil
}

func (s APIClient) RequestFinalizedBeaconBlock() (spec.AgnosticBlock, bool, error) {

	finalityCheckpoint, _ := s.Api.Finality(s.ctx, "head")

	finalizedSlot := finalityCheckpoint.Finalized.Epoch * spec.SlotsPerEpoch

	return s.RequestBeaconBlock(phase0.Slot(finalizedSlot))
}

func (s APIClient) CreateMissingBlock(slot phase0.Slot) spec.AgnosticBlock {
	duties, err := s.Api.ProposerDuties(s.ctx, phase0.Epoch(slot/32), []phase0.ValidatorIndex{})
	proposerValIdx := phase0.ValidatorIndex(0)
	if err != nil {
		log.Errorf("could not request proposer duty: %s", err)
	} else {
		for _, duty := range duties {
			if duty.Slot == phase0.Slot(slot) {
				proposerValIdx = duty.ValidatorIndex
			}
		}
	}

	return spec.AgnosticBlock{
		Slot:              slot,
		StateRoot:         s.RequestStateRoot(slot),
		ProposerIndex:     proposerValIdx,
		Graffiti:          [32]byte{},
		Proposed:          false,
		Attestations:      make([]*phase0.Attestation, 0),
		Deposits:          make([]*phase0.Deposit, 0),
		ProposerSlashings: make([]*phase0.ProposerSlashing, 0),
		AttesterSlashings: make([]*phase0.AttesterSlashing, 0),
		VoluntaryExits:    make([]*phase0.SignedVoluntaryExit, 0),
		SyncAggregate: &altair.SyncAggregate{
			SyncCommitteeBits:      bitfield.NewBitvector512(),
			SyncCommitteeSignature: phase0.BLSSignature{}},
		ExecutionPayload: spec.AgnosticExecutionPayload{
			FeeRecipient:  bellatrix.ExecutionAddress{},
			GasLimit:      0,
			GasUsed:       0,
			Timestamp:     0,
			BaseFeePerGas: [32]byte{},
			BlockHash:     phase0.Hash32{},
			Transactions:  make([]bellatrix.Transaction, 0),
		},
	}
}

// RequestBlockByHash retrieves block from the execution client for the given hash
func (s APIClient) RequestBlockByHash(hash common.Hash) (*types.Block, error) {
	if s.ELApi == nil {
		return nil, fmt.Errorf("execution layer client is not initialized")
	}
	block, err := s.ELApi.BlockByHash(s.ctx, hash)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve block by hash %s: %s", hash.String(), err.Error())
	}
	return block, nil
}
func (s APIClient) RequestCurrentHead() phase0.Slot {
	head, err := s.Api.BeaconBlockHeader(s.ctx, "head")
	if err != nil {
		log.Panicf("could not request current head: %s", err)
	}

	return head.Header.Message.Slot
}
