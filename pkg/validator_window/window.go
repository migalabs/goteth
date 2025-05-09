package validatorwindow

import (
	"context"
	"sync"
	"time"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/migalabs/goteth/pkg/clientapi"
	"github.com/migalabs/goteth/pkg/config"
	"github.com/migalabs/goteth/pkg/db"
	"github.com/migalabs/goteth/pkg/events"
	"github.com/migalabs/goteth/pkg/utils"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var (
	log = logrus.WithField(
		"module", "val-window",
	)
)

type ValidatorWindowRunner struct {
	ctx              context.Context
	dbClient         *db.DBService  // client to communicate with psql
	eventsObj        events.Events  // object to receive signals from beacon node (needed to trigger the deletes)
	stop             bool           // used to know if the tool should stop
	windowEpochSize  int            // number of epochs of data to maintain in the database
	routineSyncGroup sync.WaitGroup // to check if the routine is running
}

func NewValidatorWindow(
	pCtx context.Context,
	iConfig config.ValidatorWindowConfig) (*ValidatorWindowRunner, error) {

	// database
	idbClient, err := db.New(pCtx, iConfig.DBUrl)
	if err != nil {
		return &ValidatorWindowRunner{
			ctx: pCtx,
		}, errors.Wrap(err, "unable to generate DB Client.")
	}
	err = idbClient.Connect()

	if err != nil {
		return &ValidatorWindowRunner{
			ctx: pCtx,
		}, errors.Wrap(err, "unable to connect DB Client.")
	}

	// beacon node
	cli, err := clientapi.NewAPIClient(pCtx,
		iConfig.BnEndpoint, "", "", "", iConfig.MaxRequestRetries)

	if err != nil {
		return &ValidatorWindowRunner{
			ctx: pCtx,
		}, errors.Wrap(err, "unable to generate API Client.")
	}

	return &ValidatorWindowRunner{
		ctx:              pCtx,
		dbClient:         idbClient,
		eventsObj:        events.NewEventsObj(pCtx, cli),
		windowEpochSize:  iConfig.NumEpochs,
		routineSyncGroup: sync.WaitGroup{},
	}, nil
}

func (s *ValidatorWindowRunner) Run() {

	s.eventsObj.SubscribeToFinalizedCheckpointEvents() // every new finalized checkpoint, trigger deletes
	s.eventsObj.SubscribeToHeadEvents()                // for monitorization
	ticker := time.NewTicker(utils.RoutineFlushTimeout)
	s.routineSyncGroup.Add(1)
	defer s.routineSyncGroup.Done()

	for {
		select {

		case <-s.eventsObj.FinalizedChan:

			dbHeadEpoch, err := s.dbClient.RetrieveLastEpoch()
			if err != nil {
				log.Errorf("could not detect current head epoch in database: %s", err)
				s.EndProcesses()
				return
			}

			if dbHeadEpoch < phase0.Epoch(s.windowEpochSize) {
				log.Infof("database head epoch: %d is less than window epoch size: %d", dbHeadEpoch, s.windowEpochSize)
				continue
			}
			windowLowerEpochBoundary := dbHeadEpoch - phase0.Epoch(s.windowEpochSize)

			log.Infof("database head epoch: %d", dbHeadEpoch)
			log.Infof("deleting validator rewards from %d epoch backwards", windowLowerEpochBoundary)
			err = s.dbClient.DeleteValidatorRewardsUntil(windowLowerEpochBoundary)
			if err != nil {
				log.Errorf("could not delete the validators: %s", err)
				s.EndProcesses()
				return
			}
		case <-ticker.C:
			if s.stop {
				return
			}
		}
	}
}

func (s *ValidatorWindowRunner) Close() {
	s.stop = true
	s.routineSyncGroup.Wait()
	s.EndProcesses()
}

func (s *ValidatorWindowRunner) EndProcesses() {
	s.dbClient.Finish()
}
