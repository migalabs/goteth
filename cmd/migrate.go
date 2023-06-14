package cmd

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	cli "github.com/urfave/cli/v2"

	"github.com/cortze/eth-cl-state-analyzer/pkg/db"
)

var MigrateCommand = &cli.Command{
	Name:   "migrate",
	Usage:  "force migrate version",
	Action: LaunchForceMigration,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "db-url",
			Usage: "example: postgresql://beaconchain:beaconchain@localhost:5432/beacon_states",
		},
		&cli.IntFlag{
			Name:  "version",
			Usage: "example: 5",
		}},
}

var logMigrate = logrus.WithField(
	"module", "MigrateCommand",
)

// CrawlAction is the function that is called when running `eth2`.
func LaunchForceMigration(c *cli.Context) error {
	logMigrate.Info("parsing flags")
	// check if a config file is set
	if !c.IsSet("db-url") {
		return errors.New("database endpoint not provided")
	}
	if !c.IsSet("version") {
		return errors.New("version not provided")
	}

	dbEndpoint := c.String("db-url")
	version := c.Int("version")

	dbClient, err := db.New(c.Context, dbEndpoint)
	if err != nil {
		return errors.New("could not connect to database")
	}

	procDoneC := make(chan struct{})
	sigtermC := make(chan os.Signal, 1)

	signal.Notify(sigtermC, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, syscall.SIGTERM)

	go func() {
		dbClient.ForceMigration(version)
		procDoneC <- struct{}{}
	}()

	select {
	case <-sigtermC:
		logBlocks.Info("Sudden shutdown detected, controlled shutdown of the cli triggered")
		dbClient.Finish()

	case <-procDoneC:
		logBlocks.Info("Process successfully finish!")
	}
	close(sigtermC)
	close(procDoneC)

	return nil
}
