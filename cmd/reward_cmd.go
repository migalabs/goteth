package cmd

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	cli "github.com/urfave/cli/v2"

	"github.com/cortze/eth-cl-state-analyzer/pkg/clientapi"
	"github.com/cortze/eth-cl-state-analyzer/pkg/state"
	"github.com/cortze/eth-cl-state-analyzer/pkg/utils"
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
			Name:  "log-level",
			Usage: "log level: debug, warn, info, error",
		},
		&cli.StringFlag{
			Name:  "db-url",
			Usage: "example: postgresql://beaconchain:beaconchain@localhost:5432/beacon_states_kiln",
		},
		&cli.BoolFlag{
			Name:  "missing-vals",
			Usage: "example: true",
		},
		&cli.IntFlag{
			Name:  "workers-num",
			Usage: "example: 50",
		},
		&cli.IntFlag{
			Name:  "db-workers-num",
			Usage: "example: 50",
		},
		&cli.StringFlag{
			Name:  "download-mode",
			Usage: "example: hybrid,historical,finalized. Default: hybrid",
		},
		&cli.StringFlag{
			Name:  "custom-pools",
			Usage: "example: pools.csv. Columns: f_val_idx,pool_name",
		},
		&cli.StringFlag{
			Name:        "metrics",
			Usage:       "example: epoch,validator, epoch. Empty for all",
			DefaultText: "epoch",
		}},
}

var logRewardsRewards = logrus.WithField(
	"module", "RewardsCommand",
)

var QueryTimeout = 90 * time.Second

// TODO: work on a better config in next releases
func LaunchRewardsCalculator(c *cli.Context) error {
	coworkers := 1
	dbWorkers := 1
	downloadMode := "hybrid"
	customPools := ""
	metrics := "epoch" // By default we only track epochs, other metrics consume too much disk
	missingVals := false
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
	if c.IsSet("log-level") {
		logrus.SetLevel(utils.ParseLogLevel(c.String("log-level")))
	}
	if !c.IsSet("db-url") {
		return errors.New("db-url not provided")
	}

	if !c.IsSet("download-mode") {
		logRewardsRewards.Infof("download mode flag not provided, default: hybrid")
	} else {
		downloadMode = c.String("download-mode")
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

	if c.IsSet("custom-pools") {
		customPools = c.String("custom-pools")
	}
	if c.IsSet("missing-vals") {
		missingVals = c.Bool("missing-vals")
	}
	if c.IsSet("metrics") {
		metrics = c.String("metrics")
	}

	// generate the httpAPI client
	cli, err := clientapi.NewAPIClient(c.Context, bnEndpoint, QueryTimeout)
	if err != nil {
		return err
	}

	// generate the state analyzer
	stateAnalyzer, err := state.NewStateAnalyzer(c.Context, cli, initSlot, finalSlot, dbUrl, coworkers, dbWorkers, downloadMode, customPools, missingVals, metrics)
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
