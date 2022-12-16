package blocks

import (
	api "github.com/attestantio/go-eth2-client/api/v1"
)

func (s *BlockAnalyzer) HandleHeadEvent(event *api.Event) {
	log := log.WithField("routine", "head-event")
	if event.Data == nil {
		return
	}

	data := event.Data.(*api.HeadEvent) // cast to head event
	log.Infof("Received a new event: slot %d", data.Slot)
	s.chNewHead <- struct{}{}
}
