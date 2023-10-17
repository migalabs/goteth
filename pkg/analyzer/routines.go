package analyzer

import (
	"time"

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

			go s.DownloadBlock(phase0.Slot(downloadSlot))
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

	log.Infof("Switch to head mode: following chain head")

	// -----------------------------------------------------------------------------------
	s.eventsObj.SubscribeToHeadEvents()
	s.eventsObj.SubscribeToFinalizedCheckpointEvents()
	s.eventsObj.SubscribeToReorgsEvents()
	ticker := time.NewTicker(utils.RoutineFlushTimeout)
	// loop over the list of slots that we need to analyze

	for {
		select {

		case headSlot := <-s.eventsObj.HeadChan: // wait for new head event
			// make the block query
			log.Tracef("received new head signal: %d", headSlot)

			for nextSlotDownload <= headSlot {

				if s.processerBook.NumFreePages() > 0 {
					s.downloadTaskChan <- nextSlotDownload
				}
				nextSlotDownload = nextSlotDownload + 1

			}
		case newFinalCheckpoint := <-s.eventsObj.FinalizedChan:
			s.dbClient.Persist(db.ChepointTypeFromCheckpoint(newFinalCheckpoint))
			finalizedSlot := phase0.Slot(newFinalCheckpoint.Epoch * spec.SlotsPerEpoch)

			go s.queue.CleanQueue(finalizedSlot - (2 * spec.SlotsPerEpoch))

		// case newReorg := <-s.eventsObj.ReorgChan:
		// 	s.dbClient.Persist(db.ReorgTypeFromReorg(newReorg))
		// 	baseSlot := newReorg.Slot - phase0.Slot(newReorg.Depth)

		// 	go func() { // launch fix async
		// 		for i := baseSlot; i < newReorg.Slot; i++ {
		// 			s.ProcessOrphan(i)
		// 			s.fixMetrics(i)
		// 		}
		// 	}()

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

	// obtain last slot in database
	nextSlotDownload, err := s.dbClient.ObtainLastSlot()
	if err != nil {
		log.Errorf("could not obtain last slot in database: %s", err)
	}
	// if we did not get a last slot from the database, or we were too close to the head
	// then start from the current finalized in the chain
	if nextSlotDownload == 0 || nextSlotDownload > finalizedBlock.Slot {
		log.Infof("continue from finalized slot %d, epoch %d", finalizedBlock.Slot, finalizedBlock.Slot/spec.SlotsPerEpoch)
		nextSlotDownload = finalizedBlock.Slot
	} else {
		// database detected
		log.Infof("database detected, continue from slot %d, epoch %d", nextSlotDownload, nextSlotDownload/spec.SlotsPerEpoch)
		nextSlotDownload = nextSlotDownload - (epochsToFinalizedTentative * spec.SlotsPerEpoch) // 2 epochs before

	}

	s.initSlot = nextSlotDownload

	log.Infof("filling to head...")
	s.wgMainRoutine.Add(1) // add because historical will defer it
	s.runHistorical(nextSlotDownload, headSlot)
	return headSlot
}

func (s *ChainAnalyzer) runHistorical(init phase0.Slot, end phase0.Slot) {
	defer s.wgMainRoutine.Done()

	ticker := time.NewTicker(20 * utils.RoutineFlushTimeout)

	log.Infof("Switch to historical mode: %d - %d", init, end)

	i := init

	for i <= end {
		if s.stop {
			log.Info("sudden shutdown detected, block downloader routine")
			return
		}
		if len(ticker.C) > 0 { // every ticker, clean queue
			<-ticker.C
			s.queue.CleanQueue(s.queue.HeadBlock.Slot - (5 * spec.SlotsPerEpoch))
		}

		if s.processerBook.NumFreePages() == 0 {
			log.Warnf("hit limit of concurrent processers")
			limitTicker := time.NewTicker(utils.RoutineFlushTimeout)
			<-limitTicker.C // if rate limit, wait for ticker
		}
		s.downloadTaskChan <- i
		i += 1

	}
	log.Infof("historical mode: all download tasks sent")

}
