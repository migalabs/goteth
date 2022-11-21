package cmd

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	cli "github.com/urfave/cli/v2"

	"github.com/cortze/eth2-state-analyzer/pkg/clientapi"
	"github.com/cortze/eth2-state-analyzer/pkg/state"
	"github.com/cortze/eth2-state-analyzer/pkg/utils"
)

var RewardsCommand = &cli.Command{
	Name:   "rewards",
	Usage:  "analyze the Beacon State of a given slot range",
	Action: LaunchRewardsCalculator,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "bn-endpoint",
			Usage: "beacon node endpoint (to request the BeaconStates)",
		},
		&cli.IntFlag{
			Name:  "init-slot",
			Usage: "init slot from where to start",
		},
		&cli.IntFlag{
			Name:  "final-slot",
			Usage: "init slot from where to finish",
		},
		&cli.StringFlag{
			Name:  "validator-indexes",
			Usage: "json file including the list of validator indexes",
		},
		&cli.StringFlag{
			Name:  "log-level",
			Usage: "log level: debug, warn, info, error",
		},
		&cli.StringFlag{
			Name:  "db-url",
			Usage: "example: postgresql://beaconchain:beaconchain@localhost:5432/beacon_states_kiln",
		},
		&cli.IntFlag{
			Name:  "workers-num",
			Usage: "example: 50",
		},
		&cli.IntFlag{
			Name:  "db-workers-num",
			Usage: "example: 50",
		}},
}

var logRewardsRewards = logrus.WithField(
	"module", "RewardsCommand",
)

var QueryTimeout = 90 * time.Second

// CrawlAction is the function that is called when running `eth2`.
func LaunchRewardsCalculator(c *cli.Context) error {
	coworkers := 1
	dbWorkers := 1
	logRewardsRewards.Info("parsing flags")
	// check if a config file is set
	if !c.IsSet("bn-endpoint") {
		return errors.New("bn endpoint not provided")
	}
	if !c.IsSet("init-slot") {
		return errors.New("final slot not provided")
	}
	if !c.IsSet("final-slot") {
		return errors.New("final slot not provided")
	}
	if !c.IsSet("validator-indexes") {
		return errors.New("validator indexes not provided")
	}
	if c.IsSet("log-level") {
		logrus.SetLevel(utils.ParseLogLevel(c.String("log-level")))
	}
	if !c.IsSet("db-url") {
		return errors.New("db-url not provided")
	}
	if !c.IsSet("workers-num") {
		logRewardsRewards.Infof("workers-num flag not provided, default: 1")
	} else {
		coworkers = c.Int("workers-num")
	}
	if !c.IsSet("db-workers-num") {
		logRewardsRewards.Infof("db-workers-num flag not provided, default: 1")
	} else {
		dbWorkers = c.Int("db-workers-num")
	}
	bnEndpoint := c.String("bn-endpoint")
	initSlot := uint64(c.Int("init-slot"))
	finalSlot := uint64(c.Int("final-slot"))
	dbUrl := c.String("db-url")

	validatorIndexes, err := utils.GetValIndexesFromJson(c.String("validator-indexes"))
	if err != nil {
		return err
	}

	// generate the httpAPI client
	cli, err := clientapi.NewAPIClient(c.Context, bnEndpoint, QueryTimeout)
	if err != nil {
		return err
	}

	// generate the state analyzer
	stateAnalyzer, err := state.NewStateAnalyzer(c.Context, cli, initSlot, finalSlot, validatorIndexes, dbUrl, coworkers, dbWorkers)
	if err != nil {
		return err
	}

	procDoneC := make(chan struct{})
	sigtermC := make(chan os.Signal)

	signal.Notify(sigtermC, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, syscall.SIGTERM)

	go func() {
		stateAnalyzer.Run()
		procDoneC <- struct{}{}
	}()

	select {
	case <-sigtermC:
		logRewardsRewards.Info("Sudden shutdown detected, controlled shutdown of the cli triggered")
		stateAnalyzer.Close()

	case <-procDoneC:
		logRewardsRewards.Info("Process successfully finish!")
	}
	close(sigtermC)
	close(procDoneC)

	return nil
}
