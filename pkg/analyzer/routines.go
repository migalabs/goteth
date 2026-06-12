package analyzer

import (
	"time"

	v1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/migalabs/goteth/pkg/db"
	"github.com/migalabs/goteth/pkg/spec"
	"github.com/migalabs/goteth/pkg/utils"
)

var (
	rateLimit = 5 // limits the number of goroutines per second
)

func (s *ChainAnalyzer) runDownloadBlocks() {
	defer s.wgDownload.Done()
	log.Info("Launching Beacon Block Requester")
	ticker := time.NewTicker(utils.RoutineFlushTimeout)

downloadRoutine:
	for {

		select {

		case downloadSlot := <-s.downloadTaskChan: // wait for new head event
			log.Tracef("received new download signal: %d", downloadSlot)

			go s.DownloadBlockCotrolled(phase0.Slot(downloadSlot))
			go s.ProcessBlock(downloadSlot)

			// if epoch boundary, download state
			if (downloadSlot % spec.SlotsPerEpoch) == (spec.SlotsPerEpoch - 1) { // last slot of epoch
				// new epoch
				go s.DownloadState(downloadSlot)
				go s.ProcessStateTransitionMetrics(phase0.Epoch(downloadSlot / spec.SlotsPerEpoch))
			}
		case <-ticker.C: // every certain amount of time check if need to finish
			if s.stop && len(s.downloadTaskChan) == 0 && s.cli.ActiveReqNum() == 0 && s.processerBook.ActivePages() == 0 {
				break downloadRoutine
			}
		}
	}
	log.Infof("Block Download routine finished")
}

func (s *ChainAnalyzer) runHead() {
	defer s.wgMainRoutine.Done()
	log.Info("launching head routine")
	nextSlotDownload := s.fillToHead()

	// Wait for blocks that may still be in-flight. During historical
	// processing, CleanUpTo evicts blocks older than 5 epochs, so only
	// blocks within the last 5 epochs can still be pending. Waiting from
	// initSlot would deadlock on evicted blocks (#253), and skipping via
	// Available() would miss in-flight downloads (#248).
	waitFrom := nextSlotDownload
	if nextSlotDownload > 5*spec.SlotsPerEpoch {
		waitFrom = nextSlotDownload - 5*spec.SlotsPerEpoch
	}
	if waitFrom < s.initSlot {
		waitFrom = s.initSlot
	}
	log.Infof("waiting for remaining historical blocks (%d to %d) to complete...", waitFrom, nextSlotDownload)
	for slot := waitFrom; slot <= nextSlotDownload; slot++ {
		if _, err := s.downloadCache.BlockHistory.Wait(s.ctx, SlotTo[uint64](slot)); err != nil {
			log.Errorf("context cancelled waiting for block at slot %d: %s", slot, err)
			return
		}
	}

	log.Infof("Switch to head mode: following chain head")

	nextSlotDownload = nextSlotDownload + 1

	// -----------------------------------------------------------------------------------
	s.eventsObj.SubscribeToHeadEvents()
	s.eventsObj.SubscribeToFinalizedCheckpointEvents()
	s.eventsObj.SubscribeToReorgsEvents()
	s.eventsObj.SubscribeToBlobSidecarsEvents()
	ticker := time.NewTicker(utils.RoutineFlushTimeout)
	// loop over the list of slots that we need to analyze

	for {
		select {

		case event := <-s.eventsObj.HeadChan: // wait for new head event
			// make the block query
			log.Tracef("received new head signal: %d", event.HeadEvent.Slot)
			s.dbClient.PersistHeadEvents([]db.HeadEvent{event})

			// Cache the state root from the Head SSE event for epoch-boundary slots.
			// This allows DownloadState to fetch the state by root instead of by slot,
			// avoiding a race condition in Lighthouse v8.1.0+ where the Head event is
			// emitted before canonical_head is updated.
			lastSlotOfEpoch := (event.HeadEvent.Slot/spec.SlotsPerEpoch+1)*spec.SlotsPerEpoch - 1
			if event.HeadEvent.Slot == lastSlotOfEpoch {
				s.setEpochBoundaryStateRoot(lastSlotOfEpoch, event.HeadEvent.State)
			}

			for nextSlotDownload <= event.HeadEvent.Slot {

				if s.processerBook.NumFreePages() > 0 {
					s.downloadTaskChan <- nextSlotDownload
					nextSlotDownload = nextSlotDownload + 1
				}

			}
		case newFinalCheckpoint := <-s.eventsObj.FinalizedChan:
			s.dbClient.PersistFinalized([]v1.FinalizedCheckpointEvent{newFinalCheckpoint})
			finalizedSlot := phase0.Slot(newFinalCheckpoint.Epoch * spec.SlotsPerEpoch)

			go s.AdvanceFinalized(finalizedSlot - (2 * spec.SlotsPerEpoch))

		case newReorg := <-s.eventsObj.ReorgChan:
			s.dbClient.PersistReorgs([]v1.ChainReorgEvent{newReorg})
			go s.HandleReorg(newReorg)

		case newBlobSidecarEvent := <-s.eventsObj.BlobSidecarChan:
			s.dbClient.PersistBlobSidecarsEvents([]spec.BlobSideCarEventWraper{newBlobSidecarEvent})

		case <-s.ctx.Done():
			log.Info("context has died, closing block requester routine")
			return

		case <-ticker.C:
			if s.stop {
				log.Info("sudden shutdown detected, block downloader routine")
				return
			}
		}

	}
}

func (s *ChainAnalyzer) fillToHead() phase0.Slot {
	// ------ fill from last epoch in database to current head -------

	// obtain current finalized
	finalizedBlock, err := s.cli.RequestFinalizedBeaconBlock()
	if err != nil {
		log.Panicf("could not request the finalized block: %s", err)
	}

	// obtain current head
	headSlot := s.cli.RequestCurrentHead()
	s.DownloadBlock(headSlot) // inserts in the queue the headblock

	// obtain last slot in database
	dbHead, err := s.dbClient.RetrieveLastSlot()
	if err != nil {
		log.Fatalf("could not get head block from database: %s", err)
	}
	nextSlotDownload := spec.FirstSlotInEpoch(dbHead)

	if err != nil {
		log.Errorf("could not obtain last slot in database: %s", err)
	}
	// if we did not get a last slot from the database, or we were too close to the head
	// then start from two epochs before current finalized in the chain
	if nextSlotDownload == 0 || nextSlotDownload > finalizedBlock.Slot {
		log.Infof("continue from finalized slot %d, epoch %d", finalizedBlock.Slot, finalizedBlock.Slot/spec.SlotsPerEpoch)
		nextSlotDownload = finalizedBlock.Slot - (epochsToFinalizedTentative * spec.SlotsPerEpoch) // 2 epochs before

	} else {
		// database detected
		log.Infof("database detected, continue from slot %d, epoch %d", nextSlotDownload, nextSlotDownload/spec.SlotsPerEpoch)
		nextSlotDownload = nextSlotDownload - (epochsToFinalizedTentative * spec.SlotsPerEpoch) // 2 epochs before
	}
	nextSlotDownload = nextSlotDownload / spec.SlotsPerEpoch * spec.SlotsPerEpoch
	s.initSlot = nextSlotDownload / spec.SlotsPerEpoch * spec.SlotsPerEpoch
	s.startEpochAggregation = phase0.Epoch(spec.EpochAtSlot(s.initSlot) + 2)
	s.endEpochAggregation = s.startEpochAggregation + phase0.Epoch(s.rewardsAggregationEpochs-1)

	log.Infof("filling to head...")

	// Re-run historical if the chain head moved more than half an epoch
	// during the previous pass. Hands off to runHead with a small gap.
	//
	// The handoff gap is half SlotsPerEpoch (not the full epoch) because
	// processerBook is sized to SlotsPerEpoch (see NewChainAnalyzer in
	// chain_analyzer.go). Returning with a gap equal to the pool capacity
	// lets runHead's first enqueue burst saturate the pool immediately;
	// any cross-epoch BlockHistory.Wait dependency in the first batch of
	// ProcessStateTransitionMetrics goroutines then deadlocks the pool —
	// the exact failure mode this loop was added to avoid in the first
	// place. Half an epoch leaves room for those dependencies to drain.
	handoffGapSlots := phase0.Slot(spec.SlotsPerEpoch / 2)
	for {
		s.wgMainRoutine.Add(1) // add because historical will defer it
		s.runHistorical(nextSlotDownload, headSlot)

		// runHistorical returns immediately when s.stop is set. Without
		// this guard the outer loop would re-query the head and call
		// runHistorical again — and since the chain keeps advancing,
		// handoffThreshold is always exceeded and the loop spins
		// indefinitely on shutdown.
		if s.stop {
			return headSlot
		}

		nextSlotDownload = headSlot + 1
		newHead := s.cli.RequestCurrentHead()
		handoffThreshold := headSlot + handoffGapSlots
		if newHead <= handoffThreshold {
			return headSlot
		}
		log.Infof("head moved %d slots during catch-up, looping historical", newHead-headSlot)
		headSlot = newHead
	}
}

func (s *ChainAnalyzer) runHistorical(init phase0.Slot, end phase0.Slot) {
	defer s.wgMainRoutine.Done()

	log.Infof("Switch to historical mode: %d - %d", init, end)

	i := init
	for i <= end {
		if s.stop {
			log.Info("sudden shutdown detected, block downloader routine")
			return
		}
		if s.processerBook.NumFreePages() == 0 {
			log.Debugf("hit limit of concurrent processers")
			limitTicker := time.NewTicker(utils.RoutineFlushTimeout)
			<-limitTicker.C // if rate limit, wait for ticker
			continue
		}
		if i%spec.SlotsPerEpoch == 0 { // every time a new epoch is crossed
			finalizedSlot, err := s.cli.RequestFinalizedBeaconBlock()

			if err != nil {
				log.Fatalf("could not request finalized slot: %s", err)
			}

			if i >= finalizedSlot.Slot {
				// keep 2 epochs before finalized, needed to calculate epoch metrics
				s.AdvanceFinalized(finalizedSlot.Slot - spec.SlotsPerEpoch*5) // includes check and clean
			} else if i > (5 * spec.SlotsPerEpoch) {
				// keep 5 epochs before current downloading slot, need 3 at least for epoch metrics
				// magic number, 2 extra if processer takes long
				cleanUpToSlot := i - (5 * spec.SlotsPerEpoch)
				s.downloadCache.CleanUpTo(cleanUpToSlot) // only clean, no check, keep
			}
		}

		select {
		case s.downloadTaskChan <- i:
			i += 1
		case <-s.ctx.Done():
			log.Info("context cancelled, stopping historical download")
			return
		}

	}
	log.Infof("historical mode: all download tasks sent")

}
