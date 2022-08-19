package analyzer

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/altair"

	"github.com/cortze/eth2-state-analyzer/pkg/clientapi"
	"github.com/cortze/eth2-state-analyzer/pkg/custom_spec"
	"github.com/cortze/eth2-state-analyzer/pkg/db/postgresql"
	"github.com/cortze/eth2-state-analyzer/pkg/model"
	"github.com/cortze/eth2-state-analyzer/pkg/utils"
)

var (
	modName = "Analyzer"
	log     = logrus.WithField(
		"module", modName,
	)
	maxWorkers = 50
	minReqTime = 10 * time.Second
)

type StateAnalyzer struct {
	ctx              context.Context
	InitSlot         uint64
	FinalSlot        uint64
	ValidatorIndexes []uint64
	//ValidatorPubkeys []
	// map of [validatorIndexes]RewardMetrics
	Metrics    sync.Map
	SlotRanges []uint64
	//
	EpochTaskChan chan *EpochTask
	ValTaskChan   chan *ValTask
	// http CLient
	cli      *clientapi.APIClient
	dbClient *postgresql.PostgresDBService

	//
	initTime time.Time
}

func NewStateAnalyzer(ctx context.Context, httpCli *clientapi.APIClient, initSlot uint64, finalSlot uint64, valIdxs []uint64, idbUrl string) (*StateAnalyzer, error) {
	log.Infof("generating new State Analzyer from slots %d:%d, for validators %v", initSlot, finalSlot, valIdxs)
	// Check if the range of slots is valid
	if !utils.IsValidRangeuint64(initSlot, finalSlot) {
		return nil, errors.New("provided slot range isn't valid")
	}

	// check if valIdx where given
	if len(valIdxs) < 1 {
		return nil, errors.New("no validator indexes where provided")
	}

	// calculate the list of slots that we will analyze
	slotRanges := make([]uint64, 0)
	epochRange := uint64(0)

	// minimum slot is 31
	// force to be in the previous epoch than select by user
	initSlot = uint64(math.Max(31, float64(int(initSlot-custom_spec.SLOTS_PER_EPOCH))))
	initEpoch := int(initSlot / 32)
	// force to be on the last slot of the init epoch
	// epoch 0 ==> (0+1) * 32 - 1
	initSlot = uint64((initEpoch+1)*custom_spec.SLOTS_PER_EPOCH - 1)

	finalSlot = uint64(math.Max(31, float64(finalSlot)))
	finalEpoch := int(finalSlot / 32)
	// for the finalSlot go the last slot of the next epoch
	// remember rewards are calculated post epoch
	finalSlot = uint64((finalEpoch+3)*custom_spec.SLOTS_PER_EPOCH - 1)

	for i := initSlot; i <= (finalSlot); i += utils.SlotBase {
		slotRanges = append(slotRanges, i)
		epochRange++
	}
	log.Debug("slotRanges are:", slotRanges)

	var metrics sync.Map

	i_dbClient, err := postgresql.ConnectToDB(ctx, idbUrl)
	if err != nil {
		return nil, errors.Wrap(err, "unable to generate DB Client.")
	}

	return &StateAnalyzer{
		ctx:              ctx,
		InitSlot:         initSlot,
		FinalSlot:        finalSlot,
		ValidatorIndexes: valIdxs,
		SlotRanges:       slotRanges,
		Metrics:          metrics,
		EpochTaskChan:    make(chan *EpochTask, 10),
		ValTaskChan:      make(chan *ValTask, len(valIdxs)),
		cli:              httpCli,
		dbClient:         i_dbClient,
	}, nil
}

func (s *StateAnalyzer) Run() {
	// State requester
	var wg sync.WaitGroup

	wg.Add(1)
	// State requester + Task generator
	go func() {
		defer wg.Done()
		log.Info("Launching Beacon State Requester")
		// loop over the list of slots that we need to analyze
		var prevBState spec.VersionedBeaconState // to be checked, it may make calculation easier to store previous state
		var bstate *spec.VersionedBeaconState
		var err error
		for _, slot := range s.SlotRanges {
			ticker := time.NewTicker(minReqTime)
			select {
			case <-s.ctx.Done():
				log.Info("context has died, closing state requester routine")
				close(s.EpochTaskChan)
				return

			default:
				firstIteration := false
				// make the state query
				log.Debugf("requesting Beacon State from endpoint: slot %d", slot)
				if bstate != nil { // in case we already had a bstate (only false the first time)
					prevBState = *bstate
				} else {
					firstIteration = true
				}
				bstate, err = s.cli.Api.BeaconState(s.ctx, fmt.Sprintf("%d", slot))
				if !firstIteration {
					// only execute tasks if it is not the first iteration
					if err != nil {
						// close the channel (to tell other routines to stop processing and end)
						log.Errorf("Unable to retrieve Beacon State from the beacon node, closing requester routine. %s", err.Error())
						// close(s.EpochTaskChan)
						return
					}

					// we now only compose one single task that contains a list of validator indexes
					// compose the next task
					epochTask := &EpochTask{
						ValIdxs:   s.ValidatorIndexes,
						Slot:      slot,
						State:     bstate,
						PrevState: prevBState,
					}

					log.Debugf("sending task for slot: %d", slot)
					s.EpochTaskChan <- epochTask
				}

			}
			// check if the min Request time has been completed (to avoid spaming the API)
			<-ticker.C
		}
		log.Infof("All states for the slot ranges has been successfully retrieved, clossing go routine")
		close(s.EpochTaskChan)
	}()

	go func() {
		defer wg.Done()

		for {
			// check if the channel has been closed
			task, ok := <-s.EpochTaskChan
			if !ok {
				log.Warn("the task channel has been closed, finishing epoch routine")
				return
			}
			log.Debugf("task received for slot %d", task.Slot)
			// Proccess State
			log.Debug("analyzing the receved state")

			// returns the state in a custom struct for Phase0, Altair of Bellatrix
			customBState, err := custom_spec.BStateByForkVersion(task.State, task.PrevState, s.cli.Api)

			if err != nil {
				log.Errorf(err.Error())
			}

			valTaskSize := (len(task.ValIdxs) / maxWorkers)

			for i := 0; i < len(task.ValIdxs); i += valTaskSize {
				lastPosition := i + valTaskSize
				if lastPosition >= len(task.ValIdxs) {
					lastPosition = len(task.ValIdxs)
				}
				valTask := &ValTask{
					ValIdxs:     task.ValIdxs[i:lastPosition],
					CustomState: customBState,
				}
				s.ValTaskChan <- valTask
			}

			// create a model to be inserted into the db
			epochDBRow := model.NewEpochMetrics(
				customBState.CurrentEpoch(),
				customBState.CurrentSlot(),
				0,
				0,
				0,
				0,
				0,
				0,
				0,
				0,
				uint64(len(customBState.GetMissedBlocks())))

			err = s.dbClient.InsertNewEpochRow(epochDBRow)
			if err != nil {
				log.Errorf(err.Error())
			}

			epochDBRow.PrevNumAttestations = customBState.GetAttNum()
			epochDBRow.PrevNumAttValidators = customBState.GetAttestingValNum()
			epochDBRow.PrevNumValidators = customBState.GetNumVals()
			epochDBRow.TotalBalance = customBState.GetTotalActiveBalance()
			epochDBRow.TotalEffectiveBalance = customBState.GetTotalActiveEffBalance()

			epochDBRow.MissingSource = customBState.GetMissingFlag(int(altair.TimelySourceFlagIndex))
			epochDBRow.MissingTarget = customBState.GetMissingFlag(int(altair.TimelyTargetFlagIndex))
			epochDBRow.MissingHead = customBState.GetMissingFlag(int(altair.TimelyHeadFlagIndex))

			err = s.dbClient.UpdatePrevEpochMetrics(epochDBRow)
			if err != nil {
				log.Errorf(err.Error())
			}

			select {
			case <-s.ctx.Done():
				log.Info("context has died, closing state processer routine")
				close(s.EpochTaskChan)
				return

			default:

			}
		}
	}()

	// generate workers, validator tasks consumers
	coworkers := len(s.ValidatorIndexes)
	if coworkers > maxWorkers {
		coworkers = maxWorkers
	}
	for i := 0; i < coworkers; i++ {
		// state workers, receiving State and valIdx to measure performance
		wlog := logrus.WithField(
			"worker", i,
		)

		wlog.Info("Launching Task Worker")
		wg.Add(1)
		go func() {
			defer wg.Done()

			// keep iterating until the channel is closed due to finishing
			for {
				// check if the channel has been closed
				valTask, ok := <-s.ValTaskChan
				if !ok {
					wlog.Warn("the task channel has been closed, finishing worker routine")
					return
				}
				customBState := valTask.CustomState
				log.Debugf("Length of validator set: %d", len(valTask.ValIdxs))
				wlog.Debugf("task received for val %d - %d in slot %d", valTask.ValIdxs[0], valTask.ValIdxs[len(valTask.ValIdxs)-1], valTask.CustomState.CurrentSlot())
				// Proccess State
				wlog.Debug("analyzing the receved state")

				for _, valIdx := range valTask.ValIdxs {

					// get max reward at given epoch using the formulas
					maxReward, err := customBState.GetMaxReward(valIdx)

					if err != nil {
						log.Errorf("Error obtaining max reward: ", err.Error())
					}

					// calculate the current balance of validator
					balance, err := customBState.Balance(valIdx)

					if err != nil {
						log.Errorf("Error obtaining validator balance: ", err.Error())
					}
					//TODO: Added specific flag missing support for validators
					// TODO: But pending for optimizations before further processing
					// create a model to be inserted into the db
					validatorDBRow := model.NewValidatorRewards(
						valIdx,
						customBState.CurrentSlot(),
						customBState.CurrentEpoch(),
						balance,
						0, // reward is written after state transition
						maxReward,
						customBState.GetAttSlot(valIdx),
						customBState.GetAttInclusionSlot(valIdx),
						uint64(customBState.GetBaseReward(valIdx)),
						false,
						false,
						false)

					err = s.dbClient.InsertNewValidatorRow(validatorDBRow)
					if err != nil {
						log.Errorf(err.Error())
					}

					rewardSlot := int(customBState.PrevStateSlot())
					rewardEpoch := int(customBState.PrevStateEpoch())
					if rewardSlot >= 31 {
						reward := customBState.PrevEpochReward(valIdx)

						// log.Debugf("Slot %d Validator %d Reward: %d", rewardSlot, valIdx, reward)

						// keep in mind that rewards for epoch 10 can be seen at beginning of epoch 12,
						// after state_transition
						// https://notes.ethereum.org/@vbuterin/Sys3GLJbD#Epoch-processing
						validatorDBRow = model.NewValidatorRewards(valIdx,
							uint64(rewardSlot),
							uint64(rewardEpoch),
							0, // balance: was already filled in the last epoch
							int64(reward),
							0, // maxReward: was already calculated in the previous epoch
							0,
							0,
							0,
							false,
							false,
							false)

						err = s.dbClient.UpdateValidatorRowReward(validatorDBRow)
						if err != nil {
							log.Errorf(err.Error())
						}
					}

				}
			}

		}()
	}

	// Get init time
	s.initTime = time.Now()

	log.Info("State Analyzer initialized at", s.initTime)
	wg.Wait()

	analysisDuration := time.Since(s.initTime)
	log.Info("State Analyzer finished in ", analysisDuration)

}

//
type EpochTask struct {
	ValIdxs     []uint64
	Slot        uint64
	State       *spec.VersionedBeaconState
	PrevState   spec.VersionedBeaconState
	OnlyPrevAtt bool
}

type ValTask struct {
	ValIdxs     []uint64
	CustomState custom_spec.CustomBeaconState
	OnlyPrevAtt bool
}

// Exporter Functions
