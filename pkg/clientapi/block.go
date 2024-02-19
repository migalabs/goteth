package clientapi

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/attestantio/go-eth2-client/api"
	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/altair"
	"github.com/attestantio/go-eth2-client/spec/bellatrix"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	local_spec "github.com/migalabs/goteth/pkg/spec"
	"github.com/migalabs/goteth/pkg/utils"
	bitfield "github.com/prysmaticlabs/go-bitfield"
)

var (
	slotKeyTag string = "slot="
)

func (s *APIClient) RequestBeaconBlock(slot phase0.Slot) (*local_spec.AgnosticBlock, error) {
	routineKey := fmt.Sprintf("%s%d", slotKeyTag, slot)
	s.blocksBook.Acquire(routineKey)
	defer s.blocksBook.FreePage(routineKey)

	log.Debugf("downloading block at slot %d", slot)

	startTime := time.Now()
	err := errors.New("first attempt")
	var newBlock *api.Response[*spec.VersionedSignedBeaconBlock]

	attempts := 0
	for err != nil && attempts < maxRetries {

		newBlock, err = s.Api.SignedBeaconBlock(s.ctx, &api.SignedBeaconBlockOpts{
			Block: fmt.Sprintf("%d", slot),
		})

		if newBlock == nil {
			log.Warnf("the beacon block at slot %d does not exist, missing block", slot)
			return s.CreateMissingBlock(slot), nil
		}
		if errors.Is(err, context.DeadlineExceeded) {
			ticker := time.NewTicker(utils.RoutineFlushTimeout)
			log.Warnf("retrying request: %s", routineKey)
			<-ticker.C
		}
		attempts += 1

	}
	if err != nil {
		// close the channel (to tell other routines to stop processing and end)
		return &local_spec.AgnosticBlock{}, fmt.Errorf("unable to retrieve Beacon Block at slot %d: %s", slot, err.Error())
	}
	customBlock, err := local_spec.GetCustomBlock(*newBlock.Data)

	if err != nil {
		// close the channel (to tell other routines to stop processing and end)
		return &local_spec.AgnosticBlock{}, fmt.Errorf("unable to parse Beacon Block at slot %d: %s", slot, err.Error())
	}

	// fill in block size on custom block using RequestBlockByHash
	// shows error inside function if ELApi is not defined
	block, err := s.RequestExecutionBlockByHash(common.Hash(customBlock.ExecutionPayload.BlockHash))
	if err != nil {
		log.Error("cannot request block by hash: %s", err)
	}
	if block != nil {
		customBlock.ExecutionPayload.PayloadSize = uint32(block.Size())
	}

	customBlock.StateRoot = s.RequestStateRoot(slot)

	if s.Metrics.APIRewards {
		reward, err := s.RequestBlockRewards(slot)
		if err != nil {
			log.Error("cannot request block reward: %s", err)
		}

		customBlock.Reward = reward
	}
	log.Infof("block at slot %d downloaded in %f seconds", slot, time.Since(startTime).Seconds())

	return &customBlock, nil
}

func (s *APIClient) RequestFinalizedBeaconBlock() (*local_spec.AgnosticBlock, error) {

	finalityCheckpoint, _ := s.Api.Finality(s.ctx, &api.FinalityOpts{
		State: "head",
	})

	finalizedSlot := finalityCheckpoint.Data.Finalized.Epoch * local_spec.SlotsPerEpoch

	return s.RequestBeaconBlock(phase0.Slot(finalizedSlot))
}

func (s *APIClient) RequestBlockRoot(slot phase0.Slot) phase0.Root {

	root, err := s.Api.BeaconBlockRoot(s.ctx, &api.BeaconBlockRootOpts{
		Block: fmt.Sprintf("%d", slot),
	})
	if err != nil {
		log.Panicf("could not download the block root at %d: %s", slot, err)
	}

	if root == nil { // block root may be empty
		return phase0.Root{}
	}

	return *root.Data
}

func (s *APIClient) CreateMissingBlock(slot phase0.Slot) *local_spec.AgnosticBlock {
	duties, err := s.Api.ProposerDuties(s.ctx, &api.ProposerDutiesOpts{
		Indices: []phase0.ValidatorIndex{},
		Epoch:   phase0.Epoch(slot / 32),
	})
	proposerValIdx := phase0.ValidatorIndex(0)
	if err != nil {
		log.Errorf("could not request proposer duty: %s", err)
	} else {
		for _, duty := range duties.Data {
			if duty.Slot == phase0.Slot(slot) {
				proposerValIdx = duty.ValidatorIndex
			}
		}
	}

	return &local_spec.AgnosticBlock{
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
			SyncCommitteeSignature: phase0.BLSSignature{},
		},
		ExecutionPayload: local_spec.AgnosticExecutionPayload{
			FeeRecipient:  bellatrix.ExecutionAddress{},
			GasLimit:      0,
			GasUsed:       0,
			Timestamp:     0,
			BaseFeePerGas: [32]byte{},
			BlockHash:     phase0.Hash32{},
			Transactions:  make([]bellatrix.Transaction, 0),
			PayloadSize:   uint32(0),
		}, // snappy
		SSZsize:           uint32(0),
		SnappySize:        uint32(0),
		CompressionTime:   0 * time.Second,
		DecompressionTime: 0 * time.Second,
	}
}

// RequestBlockByHash retrieves block from the execution client for the given hash
func (s *APIClient) RequestExecutionBlockByHash(hash common.Hash) (*types.Block, error) {

	if s.ELApi == nil {
		return nil, nil
	}
	emptyHash := common.Hash{}

	if hash == emptyHash {
		return nil, nil // empty hash, not even try (probably we are before Bellatrix)
	}

	routineKey := "block=" + hash.String()
	s.txBook.Acquire(routineKey)
	defer s.txBook.FreePage(routineKey)

	block, err := s.ELApi.BlockByHash(s.ctx, hash)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve block by hash %s: %s", hash.String(), err.Error())
	}
	return block, nil
}

func (s *APIClient) RequestCurrentHead() phase0.Slot {

	head, err := s.Api.BeaconBlockHeader(s.ctx, &api.BeaconBlockHeaderOpts{
		Block: "head",
	})
	if err != nil {
		log.Panicf("could not request current head: %s", err)
	}

	return head.Data.Header.Message.Slot
}
