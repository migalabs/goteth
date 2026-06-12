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
	ctx                 context.Context
	dbClient            *db.DBService  // client to communicate with psql
	eventsObj           events.Events  // object to receive signals from beacon node (needed to trigger the deletes)
	stop                bool           // used to know if the tool should stop
	windowEpochSize     int            // number of epochs of data to maintain in the database
	deleteCadenceEpochs int            // only emit DELETE once the boundary has advanced by this many epochs
	lastDeletedBoundary phase0.Epoch   // last boundary we successfully sent a DELETE for
	deleteFiredOnce     bool           // becomes true after the first DELETE so the cadence check kicks in
	routineSyncGroup    sync.WaitGroup // to check if the routine is running
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
		iConfig.BnEndpoint, iConfig.MaxRequestRetries)

	if err != nil {
		return &ValidatorWindowRunner{
			ctx: pCtx,
		}, errors.Wrap(err, "unable to generate API Client.")
	}

	cadence := iConfig.DeleteCadenceEpochs
	if cadence < 1 {
		cadence = 1
	}

	return &ValidatorWindowRunner{
		ctx:                 pCtx,
		dbClient:            idbClient,
		eventsObj:           events.NewEventsObj(pCtx, cli),
		windowEpochSize:     iConfig.NumEpochs,
		deleteCadenceEpochs: cadence,
		routineSyncGroup:    sync.WaitGroup{},
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

			// Each DELETE FROM t_validator_rewards_summary becomes a ClickHouse
			// mutation that rewrites every part holding in-window epochs. On
			// networks with ~1M validators that mutation takes minutes per fire;
			// firing on every finalized checkpoint (~6:24 min) saturates the
			// merge thread and stalls the head. Only emit the DELETE once the
			// boundary has advanced by deleteCadenceEpochs since the last fire.
			if s.deleteFiredOnce {
				if windowLowerEpochBoundary <= s.lastDeletedBoundary {
					log.Debugf("skipping delete: boundary %d not advanced past last fire %d",
						windowLowerEpochBoundary, s.lastDeletedBoundary)
					continue
				}
				advanced := windowLowerEpochBoundary - s.lastDeletedBoundary
				if advanced < phase0.Epoch(s.deleteCadenceEpochs) {
					log.Debugf("skipping delete: boundary advanced %d/%d epochs since last fire (boundary=%d)",
						advanced, s.deleteCadenceEpochs, windowLowerEpochBoundary)
					continue
				}
			}

			log.Infof("database head epoch: %d", dbHeadEpoch)
			log.Infof("deleting validator rewards from %d epoch backwards", windowLowerEpochBoundary)
			err = s.dbClient.DeleteValidatorRewardsUntil(windowLowerEpochBoundary)
			if err != nil {
				log.Errorf("could not delete the validators: %s", err)
				s.EndProcesses()
				return
			}
			s.lastDeletedBoundary = windowLowerEpochBoundary
			s.deleteFiredOnce = true
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
