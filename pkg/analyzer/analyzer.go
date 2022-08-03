package analyzer

import (
	"context"
	"fmt"
	"math"
	"os"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	api "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/phase0"

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
	maxWorkers = 10
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
	// minimum slot is 0
	// its already a uint64
	initSlot = uint64(math.Max(31, float64(initSlot)))
	finalSlot = uint64(math.Max(31, float64(finalSlot)))
	initEpoch := int(initSlot / 32)
	finalEpoch := int(finalSlot / 32)
	// force to be on the last slot of the init epoch
	// epoch 0 ==> (0+1) * 32 - 1
	initSlot = uint64((initEpoch+1)*custom_spec.SLOTS_PER_EPOCH - 1)
	// for the finalSlot go the last slot of the next epoch
	// remember rewards are calculated post epoch
	finalSlot = uint64((finalEpoch+2)*custom_spec.SLOTS_PER_EPOCH - 1)
	for i := initSlot; i < (finalSlot + utils.SlotBase); i += utils.SlotBase {
		slotRanges = append(slotRanges, i)
		epochRange++
	}
	log.Debug("slotRanges are:", slotRanges)

	var metrics sync.Map
	// Compose the metrics array with each of the RewardMetrics
	for _, val := range valIdxs {
		mets, err := NewRewardMetrics(initSlot, epochRange, val)
		if err != nil {
			return nil, errors.Wrap(err, "unable to generate RewarMetrics.")
		}
		metrics.Store(val, mets)
	}

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
		EpochTaskChan:    make(chan *EpochTask, len(valIdxs)),
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
		for idx, slot := range s.SlotRanges {
			ticker := time.NewTicker(minReqTime)
			select {
			case <-s.ctx.Done():
				log.Info("context has died, closing state requester routine")
				close(s.EpochTaskChan)
				return

			default:
				// make the state query
				log.Debug("requesting Beacon State from endpoint")
				if bstate != nil { // in case we already had a bstate (only false the first time)
					prevBState = *bstate
				}
				bstate, err = s.cli.Api.BeaconState(s.ctx, fmt.Sprintf("%d", slot))
				if err != nil {
					// close the channel (to tell other routines to stop processing and end)
					log.Errorf("Unable to retrieve Beacon State from the beacon node, closing requester routine. %s", err.Error())
					// close(s.EpochTaskChan)
					return
				}

				log.Debug("requesting Validator list from endpoint")
				validatorFilter := make([]phase0.ValidatorIndex, 0)
				activeValidators, err := s.cli.Api.Validators(s.ctx, fmt.Sprintf("%d", slot), validatorFilter)
				if err != nil {
					// close the channel (to tell other routines to stop processing and end)
					log.Errorf("Unable to retrieve Validators from the beacon node, closing requester routine. %s", err.Error())
					close(s.EpochTaskChan)
					return
				}

				var totalActiveBalance uint64 = 0
				var totalEffectiveBalance uint64 = 0

				for _, val := range activeValidators {
					// only count active validators
					if !val.Status.IsActive() {
						continue
					}
					// since it's active
					totalActiveBalance += uint64(val.Balance)
					totalEffectiveBalance += uint64(val.Validator.EffectiveBalance)

				}

				// we now only compose one single task that contains a list of validator indexes
				// compose the next task
				valTask := &EpochTask{
					ValIdxs:               s.ValidatorIndexes,
					Slot:                  slot,
					State:                 bstate,
					PrevState:             prevBState,
					TotalValidatorStatus:  &activeValidators,
					TotalEffectiveBalance: totalEffectiveBalance,
					TotalActiveBalance:    totalActiveBalance,
				}

				// to be checked, as we may change how we calcuate rewards
				if idx == len(s.SlotRanges)-1 {
					valTask.OnlyPrevAtt = true
				}

				log.Debugf("sending task for slot: %d", slot)
				s.EpochTaskChan <- valTask

			}
			// check if the min Request time has been completed (to avoid spaming the API)
			<-ticker.C
		}
		log.Infof("All states for the slot ranges has been successfully retrieved, clossing go routine")
		close(s.EpochTaskChan)
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

			// keep iterrating until the channel is closed due to finishing
			for {
				// cehck if the channel has been closed
				task, ok := <-s.EpochTaskChan
				if !ok {
					wlog.Warn("the task channel has been closed, finishing worker routine")
					return
				}
				wlog.Debugf("task received for slot %d", task.Slot)
				// Proccess State
				wlog.Debug("analyzing the receved state")

				// returns the state in a custom struct for Phase0, Altair of Bellatrix
				customBState, err := custom_spec.BStateByForkVersion(task.State, task.PrevState, s.cli.Api)

				if err != nil {
					log.Errorf(err.Error())
				}

				// to be checked how to calculate epoch rewards, this way might be easier
				// TODO: Analyze rewards for the given Validator
				for _, valIdx := range task.ValIdxs {
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

					// create a model to be inserted into the db
					validatorDBRow := model.NewValidatorRewards(valIdx,
						customBState.CurrentSlot(),
						customBState.CurrentEpoch(),
						balance,
						0, // reward: will be filled in the next epoch
						maxReward,
						0) // attestingSlot: to be used in the future, 0 for now

					err = s.dbClient.InsertNewValidatorRow(validatorDBRow)
					if err != nil {
						log.Errorf(err.Error())
					}

					// we now fill the reward of the previous epoch
					// with the current balance difference we see the result of performing duties
					// the last epoch
					reward := customBState.PrevEpochReward(valIdx)
					log.Debugf("Slot %d Validator %d Reward: %d", customBState.PrevStateSlot(), valIdx, reward)

					validatorDBRow = model.NewValidatorRewards(valIdx,
						customBState.PrevStateSlot(),
						customBState.PrevStateEpoch(),
						0, // balance: was already filled in the last epoch
						int64(reward),
						0, // maxReward: was already calculated in the previous epoch
						0) // attestingSlot: to be used in the future, 0 for now

					err = s.dbClient.UpdateValidatorRowReward(validatorDBRow)
					if err != nil {
						log.Errorf(err.Error())
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
	ValIdxs               []uint64
	Slot                  uint64
	State                 *spec.VersionedBeaconState
	PrevState             spec.VersionedBeaconState
	TotalValidatorStatus  *map[phase0.ValidatorIndex]*api.Validator
	TotalEffectiveBalance uint64
	TotalActiveBalance    uint64
	OnlyPrevAtt           bool
}

// Exporter Functions

func (s *StateAnalyzer) ExportToCsv(outputFolder string) error {
	// check if the folder exists
	csvRewardsFile, err := os.Create(outputFolder + "/validator_rewards.csv")
	if err != nil {
		return err
	}
	csvMaxRewardFile, err := os.Create(outputFolder + "/validator_max_rewards.csv")
	if err != nil {
		return err
	}
	csvPercentageFile, err := os.Create(outputFolder + "/validator_rewards_percentage.csv")
	if err != nil {
		return err
	}
	// write headers on the csvs
	headers := "slot,total"
	for _, val := range s.ValidatorIndexes {
		headers += "," + fmt.Sprintf("%d", val)
	}
	csvRewardsFile.WriteString(headers + "\n")
	csvMaxRewardFile.WriteString(headers + "\n")
	csvPercentageFile.WriteString(headers + "\n")

	for _, slot := range s.SlotRanges {
		rowRewards := fmt.Sprintf("%d", slot)
		rowMaxRewards := fmt.Sprintf("%d", slot)
		rowRewardsPerc := fmt.Sprintf("%d", slot)

		auxRowRewards := ""
		auxRowMaxRewards := ""
		auxRowRewardsPerc := ""

		var totRewards int64
		var totMaxRewards uint64
		var totPerc float64

		// iter through the validator results
		for _, val := range s.ValidatorIndexes {

			m, ok := s.Metrics.Load(val)
			if !ok {
				log.Errorf("validator %d has no metrics", val)
			}
			met := m.(*RewardMetrics)
			valMetrics, err := met.GetEpochMetrics(slot)
			if err != nil {
				return err
			}
			// s.dbClient.InsertNewValidatorRow(valMetrics)

			totRewards += valMetrics.Reward
			totMaxRewards += valMetrics.MaxReward
			totPerc += valMetrics.RewardPercentage

			auxRowRewards += "," + fmt.Sprintf("%d", valMetrics.Reward)
			auxRowMaxRewards += "," + fmt.Sprintf("%d", valMetrics.MaxReward)
			auxRowRewardsPerc += "," + fmt.Sprintf("%.3f", valMetrics.RewardPercentage)

		}

		rowRewards += fmt.Sprintf(",%d", totRewards) + auxRowRewards
		rowMaxRewards += fmt.Sprintf(",%d", totMaxRewards) + auxRowMaxRewards
		rowRewardsPerc += fmt.Sprintf(",%.3f", totPerc/float64(len(s.ValidatorIndexes))) + auxRowRewardsPerc

		// end up with the line
		csvRewardsFile.WriteString(rowRewards + "\n")
		csvMaxRewardFile.WriteString(rowMaxRewards + "\n")
		csvPercentageFile.WriteString(rowRewardsPerc + "\n")
	}

	csvRewardsFile.Close()
	csvMaxRewardFile.Close()
	csvPercentageFile.Close()

	return nil
}
