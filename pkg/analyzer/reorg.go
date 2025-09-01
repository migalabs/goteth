package analyzer

import (
	"fmt"

	v1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/migalabs/goteth/pkg/spec"
)

func (s *ChainAnalyzer) AdvanceFinalized(newFinalizedSlot phase0.Slot) {

    finalizedEpoch := newFinalizedSlot / spec.SlotsPerEpoch

    stateKeys := s.downloadCache.StateHistory.GetKeyList()

    advance := false

    for _, epoch := range stateKeys {
        if epoch >= uint64(finalizedEpoch) {
            continue // only process epochs that are before the given epoch
        }
        advance = true // only set flag if there is something to do

        // First, check and fix blocks for this epoch so state computations see correct blocks
        // Track whether we corrected any block or detected state mismatch to decide if we must recompute state metrics
        blocksCorrected := false
        stateNeedsRewrite := false

        // Peek state mismatch but postpone rewrite until after blocks are fixed
        cacheState := s.downloadCache.StateHistory.Wait(epoch)
        finalizedStateRoot := s.cli.RequestStateRoot(phase0.Slot(cacheState.Slot))
        cacheStateRoot := cacheState.StateRoot
        if finalizedStateRoot != cacheStateRoot { // no match, reorg happened
            log.Warnf("cache state root: %s\nfinalized block root: %s", cacheStateRoot, finalizedStateRoot)
            log.Warnf("state root for state (slot=%d) incorrect, will redownload", cacheState.Slot)
            stateNeedsRewrite = true
        }

        // loop over slots in the epoch and fix blocks first
        for slot := (epoch * spec.SlotsPerEpoch); slot < ((epoch + 1) * spec.SlotsPerEpoch); slot++ {
            cacheBlock := s.downloadCache.BlockHistory.Wait(slot)
            finalizedBlockRoot := s.cli.RequestBlockRoot(phase0.Slot(cacheBlock.Slot))
            cacheBlockRoot := cacheBlock.Root

            if finalizedBlockRoot != cacheBlockRoot {
                log.Warnf("cache block root: %s\nfinalized block root: %s", cacheBlockRoot, finalizedBlockRoot)
                log.Warnf("block root for block (slot=%d) incorrect, redownload", cacheBlock.Slot)

                s.dbClient.DeleteBlockMetrics(phase0.Slot(slot))
                log.Infof("rewriting metrics for slot %d", slot)
                // write slot metrics
                s.ProcessBlock(phase0.Slot(slot))
                blocksCorrected = true
            }
        }

        // If we corrected any block in the epoch or state root mismatched, recompute state metrics now
        if stateNeedsRewrite || blocksCorrected {
            s.dbClient.DeleteStateMetrics(phase0.Epoch(epoch))
            log.Infof("rewriting metrics for epoch %d", epoch)
            s.ProcessStateTransitionMetrics(phase0.Epoch(epoch))
        }
    }

	s.downloadCache.CleanUpTo(newFinalizedSlot)

	if advance {
		log.Infof("checked states until slot %d, epoch %d", newFinalizedSlot, newFinalizedSlot/spec.SlotsPerEpoch)

	}
}

func (s *ChainAnalyzer) HandleReorg(newReorg v1.ChainReorgEvent) {
    depth := newReorg.Depth
    reorgSlot := newReorg.Slot

    reorgedSlots := uint64(0)

    cacheHeadBlock := s.downloadCache.GetHeadBlock()
    i := cacheHeadBlock.Slot

    // Track epochs touched during the reorg to recompute their state metrics after fixing blocks
    epochsTouched := make(map[uint64]struct{})

    for reorgedSlots <= depth { // for every slot in the reorg

        block := s.downloadCache.BlockHistory.Wait(SlotTo[uint64](i)) // first check that it was already in the cache
        if i < reorgSlot && block.Proposed {
            reorgedSlots += 1 // only count as reorged slot if there was a block porposed and we are not at the reorg slot
        }
        s.processerBook.WaitUntilInactive(fmt.Sprintf("%s%d", slotProcesserTag, i)) // wait until has been processed
        oldBlock := *block

        s.DownloadBlock(i) // -> inserts into the queue and replaces old block
        newBlock := s.downloadCache.BlockHistory.Wait(SlotTo[uint64](i))

        if newBlock.Root != oldBlock.Root { // only rewrite if stateroots are different
            if block.Proposed { // keep orphans -> if previous block was proposed and roots have changed
                s.dbClient.PersistOrphans([]spec.AgnosticBlock{oldBlock})
            }
            s.dbClient.DeleteBlockMetrics(i)
            log.Infof("rewriting metrics for slot %d", i)
            // write slot metrics
            s.ProcessBlock(i)
            // mark epoch for later state rewrite once blocks of this epoch are fixed
            epochsTouched[SlotTo[uint64](i)/spec.SlotsPerEpoch] = struct{}{}
        } else {
            log.Infof("reorg slot %d: block roots are the same", i)
        }
        i -= 1
    }

    // After fixing all affected blocks, rewrite the state metrics for touched epochs
    for epochKey := range epochsTouched {
        epoch := phase0.Epoch(epochKey)
        // ensure latest state is downloaded
        endSlot := phase0.Slot(epochKey*spec.SlotsPerEpoch + (spec.SlotsPerEpoch - 1))
        s.processerBook.WaitUntilInactive(fmt.Sprintf("%s%d", epochProcesserTag, endSlot))
        s.DownloadState(endSlot)
        s.dbClient.DeleteStateMetrics(epoch)
        log.Infof("rewriting metrics for epoch %d", epoch)
        s.ProcessStateTransitionMetrics(epoch)
    }

}
