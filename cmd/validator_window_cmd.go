package cmd

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/migalabs/goteth/pkg/config"
	"github.com/migalabs/goteth/pkg/utils"

	"github.com/sirupsen/logrus"
	cli "github.com/urfave/cli/v2"

	validatorwindow "github.com/migalabs/goteth/pkg/validator_window"
)

var ValidatorWindowCommand = &cli.Command{
	Name:   "val-window",
	Usage:  "Removes old rows from the validator rewards table according to given parameters",
	Action: LaunchValidatorWindow,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:        "log-level",
			Usage:       "Log level: debug, warn, info, error",
			EnvVars:     []string{"ANALYZER_LOG_LEVEL"},
			DefaultText: "info",
		},
		&cli.StringFlag{
			Name:        "db-url",
			Usage:       "Database where to persist the metrics",
			EnvVars:     []string{"ANALYZER_DB_URL"},
			DefaultText: "postgres://user:password@localhost:5432/goteth",
		},
		&cli.StringFlag{
			Name:        "num-epochs",
			Usage:       "Defines the number of epochs to maintain in the database from the database head backwards",
			EnvVars:     []string{"NUM_EPOCHS"},
			DefaultText: "100",
		},
		&cli.StringFlag{
			Name:        "bn-endpoint",
			Usage:       "Beacon node endpoint (to request the Beacon States and Blocks)",
			EnvVars:     []string{"ANALYZER_BN_ENDPOINT"},
			DefaultText: "http://localhost:5052",
		},
		&cli.IntFlag{
			Name:        "max-request-retries",
			Usage:       "Number of retries to make when a request fails",
			EnvVars:     []string{"ANALYZER_MAX_REQUEST_RETRIES"},
			DefaultText: "3",
		},
	},
}

// CrawlAction is the function that is called when running `eth2`.
func LaunchValidatorWindow(c *cli.Context) error {

	conf := config.NewValidatorWindowConfig()
	conf.Apply(c)

	logrus.SetLevel(utils.ParseLogLevel(conf.LogLevel))

	// generate the ValidatorWindowRunner
	valWindowRunner, err := validatorwindow.NewValidatorWindow(c.Context, *conf)
	if err != nil {
		return err
	}

	procDoneC := make(chan struct{})
	sigtermC := make(chan os.Signal, 1)

	signal.Notify(sigtermC, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, syscall.SIGTERM)

	go func() {
		valWindowRunner.Run()
		procDoneC <- struct{}{}
	}()

	select {
	case <-sigtermC:
		logCmdChain.Info("Sudden shutdown detected, controlled shutdown of the cli triggered")
		valWindowRunner.Close()

	case <-procDoneC:
		logCmdChain.Info("Process successfully finished!")
	}
	close(sigtermC)
	close(procDoneC)

	return nil
}
